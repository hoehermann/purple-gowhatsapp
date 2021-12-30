cmake_minimum_required(VERSION 3.8)

project("purple-gowhatsapp")

if(WIN32 AND NOT ${CMAKE_SIZEOF_VOID_P} EQUAL 4)
    message(FATAL_ERROR "On Windows, Pidgin is 32 bit only.")
endif()

find_package(PkgConfig QUIET)
if (PKG_CONFIG_FOUND)
    pkg_check_modules(PURPLE REQUIRED purple)
    pkg_get_variable(PURPLE_PLUGIN_DIRECTORY purple plugindir)
else()
    include(CheckIncludeFile)
    set(HAVE_PURPLE_H 0)
    LIST(APPEND CMAKE_REQUIRED_INCLUDES ${PURPLE_INCLUDE_DIRS})
    CHECK_INCLUDE_FILE(purple.h HAVE_PURPLE_H)
    if(WIN32 AND NOT ${HAVE_PURPLE_H})
        message(STATUS "Trying win32 auto-configuration...")
        add_subdirectory(dependencies/win32)
    endif()
endif()

message(STATUS "PURPLE_INCLUDE_DIRS: ${PURPLE_INCLUDE_DIRS}")
message(STATUS "PURPLE_LIBRARY_DIRS: ${PURPLE_LIBRARY_DIRS}")
message(STATUS "PURPLE_LIBRARIES: ${PURPLE_LIBRARIES}")

set(TARGET_NAME gowhatsapp)
add_library(${TARGET_NAME} SHARED 
"src/gowhatsapp.hpp"
"src/init.cpp"
"src/connection.cpp"
)

file(READ "VERSION" PLUGIN_VERSION)
target_compile_definitions(${TARGET_NAME} PRIVATE PLUGIN_VERSION=${PLUGIN_VERSION})
target_include_directories(${TARGET_NAME} PRIVATE ${PURPLE_INCLUDE_DIRS})
target_link_directories(${TARGET_NAME} PRIVATE ${PURPLE_LIBRARY_DIRS})
target_link_libraries(${TARGET_NAME} PRIVATE ${PURPLE_LIBRARIES})
set_target_properties(${TARGET_NAME} PROPERTIES PREFIX "lib")

message(STATUS "PURPLE_PLUGIN_DIRECTORY: ${PURPLE_PLUGIN_DIRECTORY}")
#install(TARGETS ${TARGET_NAME} RUNTIME
#DESTINATION ${PURPLE_PLUGIN_DIRECTORY})
install(TARGETS ${TARGET_NAME}
DESTINATION ${PURPLE_PLUGIN_DIRECTORY})
