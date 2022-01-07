cmake_minimum_required(VERSION 3.8)

project("purple-gowhatsapp")

find_package(PkgConfig QUIET)
if (PKG_CONFIG_FOUND)
    pkg_check_modules(PURPLE REQUIRED purple)
    pkg_get_variable(PURPLE_PLUGIN_DIR purple plugindir)
    pkg_get_variable(PURPLE_DATA_DIR purple datarootdir)
else()
    include(CheckIncludeFile)
    set(HAVE_PURPLE_H 0)
    LIST(APPEND CMAKE_REQUIRED_INCLUDES ${PURPLE_INCLUDE_DIRS})
    CHECK_INCLUDE_FILE(purple.h HAVE_PURPLE_H)
    if(WIN32 AND NOT ${HAVE_PURPLE_H})
        message(STATUS "Trying win32 auto-configuration...")
        add_subdirectory(${CMAKE_SOURCE_DIR}/dependencies/win32)
    endif()
endif()

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")

add_subdirectory(src/go)
add_subdirectory(src/c)

install(DIRECTORY "pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")
