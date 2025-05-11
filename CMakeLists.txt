cmake_minimum_required(VERSION 3.16...3.24)

project("purple-whatsmeow" C)

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
    if (OPUSFILE_INCLUDE_DIR AND OGG_INCLUDE_DIR)
        set(OPUSFILE_INCLUDE_DIRS ${OPUSFILE_INCLUDE_DIR} ${OPUSFILE_INCLUDE_DIR}/opus ${OGG_INCLUDE_DIR})
    endif()
    find_library(OPUSFILE_LIB opusfile)
    find_library(OPUS_LIB opus)
    find_library(OGG_LIB ogg)
    if (OPUSFILE_LIB AND OPUS_LIB AND OGG_LIB)
        set(OPUSFILE_LINK_LIBRARIES ${OPUSFILE_LIB} ${OPUS_LIB} ${OGG_LIB})
    endif()
endif()

set(Purple_DIR "scripts/purple-cmake" CACHE PATH "Where to find purple-config.cmake")
find_package(Purple REQUIRED)

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")
message(STATUS "PURPLE_PLUGIN_DIR: ${PURPLE_PLUGIN_DIR}")
message(STATUS "PURPLE_DATA_DIR: ${PURPLE_DATA_DIR}")
message(STATUS "PIXBUF_INCLUDE_DIRS: ${PIXBUF_INCLUDE_DIRS}") # might be supplied by either PkgConfig or Purple on win32
message(STATUS "OPUSFILE_INCLUDE_DIRS: ${OPUSFILE_INCLUDE_DIRS}")
message(STATUS "OPUSFILE_LINK_LIBRARIES: ${OPUSFILE_LINK_LIBRARIES}")

# read version string from file
file(READ "${CMAKE_SOURCE_DIR}/VERSION" VERSION)

include(scripts/go.cmake)
add_subdirectory(glue)

install(DIRECTORY "assets/pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")

include(scripts/cpack.cmake)
