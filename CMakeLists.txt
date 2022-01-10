cmake_minimum_required(VERSION 3.8)

project("purple-gowhatsapp" LANGUAGES C)

find_package(PkgConfig QUIET)
if (${PKG_CONFIG_FOUND})
    pkg_check_modules(PURPLE REQUIRED purple)
    pkg_get_variable(PURPLE_PLUGIN_DIR purple plugindir)
    pkg_get_variable(PURPLE_DATA_DIR purple datarootdir)
elseif(WIN32)
    message(STATUS "Trying win32 auto-configuration...")
    add_subdirectory(${CMAKE_SOURCE_DIR}/dependencies/win32)
elseif(${Purple_FOUND})
    message(STATUS "Purple configured manually. Proceeding without checks.")
else()
    message(FATAL "pkg-config not found. Please configure and set Purple_FOUND manually.")
endif()

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")

add_subdirectory(src/go)
add_subdirectory(src/c)

install(DIRECTORY "pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")
