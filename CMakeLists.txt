cmake_minimum_required(VERSION 3.18) # needed on win32

set(PROJECT_NAME "purple-gowhatsapp" C)
set(VERSION_MAJOR "1")
set(VERSION_MINOR "12")
set(VERSION_PATCH "0")
set(VERSION "${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}")

# For now, we won't be doing any cross-compiling, so specifying amd64 here will cover most cases.
# If you have a different processor and need to cross-compile, say, ARM, then this won't build the package you need until it's modified.
set(ARCHITECTURE "amd64")

project(${PROJECT_NAME})

set(Purple_DIR "${CMAKE_CURRENT_SOURCE_DIR}/submodules/purple-cmake" CACHE PATH "Where to find purple-config.cmake")
find_package(Purple REQUIRED)
message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")
message(STATUS "PURPLE_PLUGIN_DIR: ${PURPLE_PLUGIN_DIR}")
message(STATUS "PURPLE_DATA_DIR: ${PURPLE_DATA_DIR}")

find_package(PkgConfig QUIET)
if (${PKG_CONFIG_FOUND})
    pkg_check_modules(PIXBUF gdk-pixbuf-2.0)
    pkg_check_modules(OPUSFILE opusfile)
endif()
message(STATUS "PIXBUF_INCLUDE_DIRS: ${PIXBUF_INCLUDE_DIRS}")
if (NOT "${OPUSFILE_FOUND}")
    message(STATUS "opusfile not found via pkg-config, trying file-based search...")
    find_path(OPUSFILE_INCLUDE_DIR opusfile.h PATH_SUFFIXES opus)
    find_path(OGG_INCLUDE_DIR ogg/ogg.h)
    if(OPUSFILE_INCLUDE_DIR)
        set(OPUSFILE_INCLUDE_DIRS ${OPUSFILE_INCLUDE_DIR} ${OPUSFILE_INCLUDE_DIR}/opus ${OGG_INCLUDE_DIR})
    endif()
    find_library(OPUSFILE_LIB opusfile)
    find_library(OPUS_LIB opus)
    find_library(OGG_LIB ogg)
    if(OPUSFILE_LIB)
        set(OPUSFILE_LIBRARIES ${OPUSFILE_LIB} ${OPUS_LIB} ${OGG_LIB})
    endif()
endif()
message(STATUS "OPUSFILE_INCLUDE_DIRS: ${OPUSFILE_INCLUDE_DIRS}")
message(STATUS "OPUSFILE_LIBRARIES: ${OPUSFILE_LIBRARIES}")

add_subdirectory(src/go)
add_subdirectory(src/c)

install(DIRECTORY "pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")

# package settings
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Whatsapp protocol plug-in for libpurple with pidgin")
set(CPACK_PACKAGE_VENDOR "Bryant Eadon")
set(CPACK_PACKAGE_DESCRIPTION "whatsmeow is a go-based whatsapp protocol plug-in for libpurple based on the whatsapp business apis.")
set(CPACK_PACKAGE_CONTACT "bryant@balancebitsconsulting.com")
set(CPACK_PACKAGE_VERSION_MAJOR "${VERSION_MAJOR}")
set(CPACK_PACKAGE_VERSION_MINOR "${VERSION_MINOR}")
set(CPACK_PACKAGE_VERSION_PATCH "${VERSION_PATCH}")
set(CPACK_DEBIAN_PACKAGE_SHLIBDEPS ON)
set(CPACK_DEBIAN_PACKAGE_ARCHITECTURE "${ARCHITECTURE}")
set(CPACK_SOURCE_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}_${CPACK_DEBIAN_PACKAGE_ARCHITECTURE}")
set(CPACK_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}_${CPACK_DEBIAN_PACKAGE_ARCHITECTURE}")
set(CPACK_DEBIAN_PACKAGE_DEPENDS "libpurple0 (>= 2.10.0), libglib2.0-0 (>= 2.24), libjson-glib-1.0-0 (>= 0.8.0)")
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "Bryant Eadon") #required

set(PACK "DEB" CACHE STRING "Generate a Package")
set(CPACK_GENERATOR ${PACK})

include(CPack)
