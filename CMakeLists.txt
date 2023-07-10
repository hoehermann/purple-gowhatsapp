cmake_minimum_required(VERSION 3.10) # lowest version this was tested with

set(PROJECT_NAME "purple-gowhatsapp" C)
set(VERSION_MAJOR "1")
set(VERSION_MINOR "12")
set(VERSION_PATCH "0")

project(${PROJECT_NAME})

find_package(PkgConfig QUIET)
if (${PKG_CONFIG_FOUND})
    pkg_check_modules(PURPLE REQUIRED purple)
    pkg_get_variable(PURPLE_PLUGIN_DIR purple plugindir)
    pkg_get_variable(PURPLE_DATA_DIR purple datarootdir)
    pkg_check_modules(PIXBUF gdk-pixbuf-2.0)
    pkg_check_modules(OPUSFILE opusfile)
elseif(WIN32)
    message(STATUS "Trying win32 auto-configuration...")
    add_subdirectory(${CMAKE_SOURCE_DIR}/dependencies/win32)
elseif(${Purple_FOUND})
    message(STATUS "Purple was configured manually. Proceeding without checks.")
else()
    message(FATAL "pkg-config not found. Please configure manually and set Purple_FOUND to YES.")
endif()

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")

add_subdirectory(src/go)
add_subdirectory(src/c)

install(DIRECTORY "pixmaps" DESTINATION "${PURPLE_DATA_DIR}" FILES_MATCHING PATTERN "*.png")

# package settings
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "Whatsapp protocol plug-in for libpurple")
set(CPACK_PACKAGE_VENDOR "Bryant Eadon")
set(CPACK_PACKAGE_DESCRIPTION "whatsmeow is a whatsapp protocol plug-in for libpurple based on the whatsapp business apis")
set(CPACK_PACKAGE_CONTACT "bryant@balancebitsconsulting.com")
set(CPACK_PACKAGE_VERSION_MAJOR "${VERSION_MAJOR}")
set(CPACK_PACKAGE_VERSION_MINOR "${VERSION_MINOR}")
set(CPACK_PACKAGE_VERSION_PATCH "${VERSION_PATCH}")
set(CPACK_DEBIAN_PACKAGE_SHLIBDEPS ON)
set(CPACK_SOURCE_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}")
set(CPACK_PACKAGE_FILE_NAME "${CMAKE_PROJECT_NAME}_${VERSION}")
SET(CPACK_DEBIAN_PACKAGE_DEPENDS "libpurple0 (>= 2.10.0), libglib2.0-0 (>= 2.24), libjson-glib-1.0-0 (>= 0.8.0)")
set(CPACK_DEBIAN_PACKAGE_MAINTAINER "Bryant Eadon") #required

set(PACK "DEB" CACHE STRING "Generate a Package")
set(CPACK_GENERATOR ${PACK})

include(CPack)
