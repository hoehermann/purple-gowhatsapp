cmake_minimum_required(VERSION 3.20) # for cmake_path

if (NOT (CMAKE_C_COMPILER_ID STREQUAL "GNU" OR CMAKE_C_COMPILER_ID STREQUAL "Clang"))
  message(FATAL_ERROR "This project uses cgo which is incompatible with anything but the Clang or GNU C Compiler (see https://github.com/golang/go/issues/36283). Your compiler is ${CMAKE_C_COMPILER_ID}.")
endif()

# assume go is in path
set(CMAKE_Go_COMPILER "go" CACHE STRING "Path to the go compiler binary")
# have a self-contained GOPATH
set(GOPATH "${CMAKE_CURRENT_BINARY_DIR}/_go")

# hack to depend on latest version using go mod tidy
add_custom_command(
  COMMENT "Updating go modules..."
  OUTPUT "${CMAKE_SOURCE_DIR}/go.mod" "${CMAKE_SOURCE_DIR}/go.sum"
  DEPENDS "${CMAKE_SOURCE_DIR}/go.mod.in"
  WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E copy "${CMAKE_SOURCE_DIR}/go.mod.in" "${CMAKE_SOURCE_DIR}/go.mod"
  COMMAND ${CMAKE_COMMAND} -E env GOPATH=${GOPATH} ${CMAKE_Go_COMPILER} mod tidy
)

# custom target for downloading modules explicitly
add_custom_target(download-modules
  COMMENT "Fetching go modules..."
  DEPENDS "${CMAKE_SOURCE_DIR}/go.mod"
  WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env GOPATH=${GOPATH} ${CMAKE_Go_COMPILER} mod download
)

# custom target for cleaning module cache
# see https://github.com/hoehermann/purple-gowhatsapp/issues/170
add_custom_target(clean-modcache
  COMMENT "Cleaning go module cache..."
  WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env GOPATH=${GOPATH} ${CMAKE_Go_COMPILER} clean -modcache
)

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
  STRING(REGEX REPLACE ".(lib|dll|a|so)$" "" LIBRARY ${LIBRARY}) # please do not
  STRING(REGEX REPLACE "^lib" "" LIBRARY ${LIBRARY}) # do not do this
  list(APPEND CGO_LDFLAGS "-l${LIBRARY}")
endforeach()
if (WIN32 AND CMAKE_CXX_COMPILER_ID STREQUAL "GNU" AND CMAKE_C_COMPILER_VERSION VERSION_GREATER 4.7.2)
  # might actually work without this up to 7.1.0, but I did not check
  # TODO: find out if clang need this
  list(APPEND CGO_LDFLAGS "-static-libgcc")
endif()
#message(STATUS "CGO_LDFLAGS: ${CGO_LDFLAGS}")

# build cgo library
# partly taken from https://github.com/mutse/go-cmake
set(GO_SRCS
  ${CMAKE_SOURCE_DIR}/bridge.go
  ${CMAKE_SOURCE_DIR}/login.go
  ${CMAKE_SOURCE_DIR}/handler.go
  ${CMAKE_SOURCE_DIR}/send_message.go
  ${CMAKE_SOURCE_DIR}/handle_message.go
  ${CMAKE_SOURCE_DIR}/logger.go
  ${CMAKE_SOURCE_DIR}/send_file.go
  ${CMAKE_SOURCE_DIR}/mark_read.go
  ${CMAKE_SOURCE_DIR}/presence.go
  ${CMAKE_SOURCE_DIR}/profile.go
  ${CMAKE_SOURCE_DIR}/send_file_checks.go
  ${CMAKE_SOURCE_DIR}/message_cache.go
  ${CMAKE_SOURCE_DIR}/groups.go
  ${CMAKE_SOURCE_DIR}/bridge.c
  ${CMAKE_SOURCE_DIR}/bridge.h
  ${CMAKE_SOURCE_DIR}/constants.c
  ${CMAKE_SOURCE_DIR}/constants.h
  ${CMAKE_SOURCE_DIR}/opusreader.h
  ${CMAKE_SOURCE_DIR}/opusreader.c
)
set(OUTPUT_PREFIX libwhatsmeow)
add_custom_command(
  OUTPUT ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h
  DEPENDS ${GO_SRCS} "${CMAKE_SOURCE_DIR}/go.mod"
  WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env
  GOPATH="${GOPATH}"
  CGO_CFLAGS="${CGO_CFLAGS}"
  CGO_LDFLAGS="${CGO_LDFLAGS}"
  ${CMAKE_Go_COMPILER} build -buildmode=c-archive
  -o "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a"
  ${CMAKE_Go_FLAGS}
)
add_custom_target(${OUTPUT_PREFIX} DEPENDS ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h)

# prepare output file paths for inclusion in C part
set(GO_WHATSAPP_INCLUDE_DIRS "${CMAKE_CURRENT_BINARY_DIR}" "${CMAKE_SOURCE_DIR}")
set(GO_WHATSAPP_LIBRARIES "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a")
