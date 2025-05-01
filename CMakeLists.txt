cmake_minimum_required(VERSION 3.20...3.24)

project("purple-whatsmeow" C)

if (NOT (CMAKE_CXX_COMPILER_ID STREQUAL "GNU" OR CMAKE_CXX_COMPILER_ID STREQUAL "Clang"))
  message(FATAL "This project uses cgo which is incompatible with anything but the Clang or GNU C Compiler (see https://github.com/golang/go/issues/36283).")
endif()

# TODO: add support for
# find_package(OpusFile CONFIG REQUIRED)
# target_link_libraries(main PRIVATE OpusFile::opusfile)
find_package(PkgConfig QUIET)
if (${PKG_CONFIG_FOUND})
    pkg_check_modules(PIXBUF gdk-pixbuf-2.0)
    pkg_check_modules(OPUSFILE opusfile)
endif()
if (NOT "${OPUSFILE_FOUND}")
    message(STATUS "opusfile not found via pkg-config, trying file-based search...")
    find_path(OPUSFILE_INCLUDE_DIR opusfile.h PATH_SUFFIXES opus)
    find_path(OGG_INCLUDE_DIR ogg/ogg.h)
    set(OPUSFILE_INCLUDE_DIRS ${OPUSFILE_INCLUDE_DIR} ${OPUSFILE_INCLUDE_DIR}/opus ${OGG_INCLUDE_DIR})
    find_library(OPUSFILE_LIB opusfile)
    find_library(OPUS_LIB opus)
    find_library(OGG_LIB ogg)
    set(OPUSFILE_LINK_LIBRARIES ${OPUSFILE_LIB} ${OPUS_LIB} ${OGG_LIB})
endif()

set(Purple_DIR "${CMAKE_CURRENT_SOURCE_DIR}/submodules/purple-cmake" CACHE PATH "Where to find purple-config.cmake")
find_package(Purple REQUIRED)

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")
message(STATUS "PURPLE_PLUGIN_DIR: ${PURPLE_PLUGIN_DIR}")
message(STATUS "PURPLE_DATA_DIR: ${PURPLE_DATA_DIR}")
message(STATUS "PIXBUF_INCLUDE_DIRS: ${PIXBUF_INCLUDE_DIRS}") # might be supplied by either PkgConfig or Purple on win32
message(STATUS "OPUSFILE_INCLUDE_DIRS: ${OPUSFILE_INCLUDE_DIRS}")
message(STATUS "OPUSFILE_LINK_LIBRARIES: ${OPUSFILE_LINK_LIBRARIES}")

set(SOURCES
src/logger.go
src/login.c
src/login.go
src/mark_read.go
src/message_cache.go
src/options.c
src/opusreader.c
src/opusreader.h
src/pixbuf.c
src/pixbuf.h
src/presence.c
src/presence.go
src/process_message.c
src/profile.go
src/purple_compat.h
src/qrcode.c
src/receipt.c
src/send_file.c
src/send_file.go
src/send_file_checks.go
src/send_message.c
src/send_message.go
src/vendor/modules.txt
src/blist.c
src/bridge.c
src/bridge.go
src/bridge.h
src/commands.c
src/constants.c
src/constants.h
src/display_message.c
src/go.mod
src/go.sum
src/gowhatsapp.h
src/groups.c
src/groups.go
src/handler.go
src/handle_attachment.c
src/handle_message.go
src/init.c
src/libwhatsmeow.h # circular dependency :(
)

# read version string from file
file(READ "${CMAKE_SOURCE_DIR}/VERSION" VERSION)

# assume go is in path
set(CMAKE_Go_COMPILER "go" CACHE STRING "Path to the go compiler binary")
# have a self-contained GOPATH
set(GOPATH "${CMAKE_CURRENT_BINARY_DIR}/_go")

# whe have all the include directories and libraries we need
# but compiler invocation is handled by go build
# so we need to put the information into the CGO environment variables
# this is so dirty
set(INCLUDE_DIRS ${PURPLE_INCLUDE_DIRS} ${PIXBUF_INCLUDE_DIRS} ${OPUSFILE_INCLUDE_DIRS})
set(LIBRARY_DIRS ${PURPLE_LIBRARY_DIRS})
set(LIBRARIES ${PURPLE_LIBRARIES} ${PIXBUF_LIBRARIES} ${OPUSFILE_LINK_LIBRARIES})

# compiler flags
set(CGO_CFLAGS "")
foreach(INCLUDE_DIR ${INCLUDE_DIRS})
  list(APPEND CGO_CFLAGS "-I${INCLUDE_DIR}")
endforeach()
list(APPEND CGO_CFLAGS "-DPLUGIN_VERSION=${VERSION}")
#message(STATUS "CGO_CFLAGS: ${CGO_CFLAGS}")

# linker flags
set(CGO_LDFLAGS "")
foreach(LIBRARY_DIR ${LIBRARY_DIRS})
  list(APPEND CGO_LDFLAGS "-L${LIBRARY_DIR}")
endforeach()
foreach(LIBRARY ${LIBRARIES})
  cmake_path(GET LIBRARY PARENT_PATH LIBRARY_DIR) # no
  list(APPEND CGO_LDFLAGS "-L${LIBRARY_DIR}")
  cmake_path(GET LIBRARY FILENAME LIBRARY) # oh no
  STRING(REGEX REPLACE ".[^.]+$" "" LIBRAR ${LIBRARY}) # please do not
  STRING(REGEX REPLACE "^lib" "" RAR ${LIBRAR}) # do not do this
  list(APPEND CGO_LDFLAGS "-l${RAR}")
endforeach()
if (WIN32 AND CMAKE_CXX_COMPILER_ID STREQUAL "GNU" AND CMAKE_C_COMPILER_VERSION VERSION_GREATER 4.7.2)
  # might actually work without this up to 7.1.0, but I did not check
  # TODO: find out if clang need this
  list(APPEND CGO_LDFLAGS "-static-libgcc")
endif()
#message(STATUS "CGO_LDFLAGS: ${CGO_LDFLAGS}")

set(OUTPUT_PREFIX libwhatsmeow)
if (WIN32)
  set(OUTPUT_SUFFIX "dll")
else()
  set(OUTPUT_SUFFIX "so")
endif()
set(OUTPUT_LIBRARY "${CMAKE_CURRENT_SOURCE_DIR}/src/${OUTPUT_PREFIX}.${OUTPUT_SUFFIX}")
add_custom_command(
  OUTPUT ${OUTPUT_LIBRARY} #"${CMAKE_CURRENT_SOURCE_DIR}/src/${OUTPUT_PREFIX}.h" # modifies our own sources
  DEPENDS ${SOURCES}
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/src" # in-tree build :(
  COMMAND ${CMAKE_COMMAND} -E env
  GOPATH=${GOPATH}
  CGO_CFLAGS="${CGO_CFLAGS}"
  CGO_LDFLAGS="${CGO_LDFLAGS}"
  ${CMAKE_Go_COMPILER} build -buildmode=c-shared -mod=vendor
  -o "${OUTPUT_PREFIX}.${OUTPUT_SUFFIX}"
)
add_custom_target(${OUTPUT_PREFIX} ALL DEPENDS ${OUTPUT_LIBRARY})

# TODO: find out how to turn this into a library, maybe https://gist.github.com/socantre/7ee63133a0a3a08f3990

install(FILES ${OUTPUT_LIBRARY} DESTINATION "${PURPLE_PLUGIN_DIR}")

install(DIRECTORY "pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")

include(cpack.cmake)
