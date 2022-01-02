#list(APPEND CMAKE_MODULE_PATH "${CMAKE_SOURCE_DIR}/dependencies/go-cmake/cmake")
# alternatives to https://github.com/mutse/go-cmake
#https://github.com/krumberg/cmake_go
#https://github.com/cpconduce/go_cmake
#set(ENV{CFLAGS} "-IT:/Programme/Programmieren/MinGW/include")
set(CMAKE_Go_COMPILER go)
#include(${CMAKE_SOURCE_DIR}/dependencies/go-cmake/cmake/golang.cmake)
#include(${CMAKE_SOURCE_DIR}/dependencies/go-cmake/cmake/flags.cmake)
#add_go_library(purple-go-whatsapp STATIC
#"login.go"
#)
#set(GO_WHATSAPP_INCLUDE_DIRS ${CMAKE_CURRENT_BINARY_DIR} PARENT_SCOPE)
#set(GO_WHATSAPP_LIBRARIES ${CMAKE_CURRENT_BINARY_DIR}/libpurple-go-whatsapp.a PARENT_SCOPE)
#set(GO_WHATSAPP_HEADERS ${CMAKE_CURRENT_BINARY_DIR}/libpurple-go-whatsapp.h PARENT_SCOPE) # TODO: connect these to the go target
#ExternalGoProject_Add(go-whatsapp github.com/Rhymen/go-whatsapp)
#add_dependencies(purple-go-whatsapp go-whatsapp)
#set(GO_WHATSAPP_LIBRARIES "${CMAKE_CURRENT_BINARY_DIR}/libpurple-go-whatsapp.a" PARENT_SCOPE)

# https://insujang.github.io/2019-12-10/cmake-for-custom-library-build-system-in-go/
set(GOPATH "${CMAKE_CURRENT_BINARY_DIR}/go")
#file(MAKE_DIRECTORY ${GOPATH})
# add_custom_target(go-qrcode go get -u github.com/skip2/go-qrcode)
# add_custom_target(go-mimetype go get -u github.com/gabriel-vasile/mimetype)
# add_custom_target(go-whatsapp go get -u github.com/Rhymen/go-whatsapp)
set(GO_SRCS
  interface.go
  login.go
)
# set(GO_LIBNAME purple-go-whatsapp.a)
add_custom_command(
  OUTPUT libpurple-go-whatsapp.a libpurple-go-whatsapp.h
  DEPENDS ${GO_SRCS}
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env GOPATH=${GOPATH} 
  ${CMAKE_Go_COMPILER} build -buildmode=c-archive
  -o "${CMAKE_CURRENT_BINARY_DIR}/libpurple-go-whatsapp.a"
  ${CMAKE_GO_FLAGS} ${GO_SRCS}
)
add_custom_target(purple-go-whatsapp DEPENDS libpurple-go-whatsapp.a libpurple-go-whatsapp.h)
# #Add a custom target for the library.
# add_custom_target(purple-go-whatsapp DEPENDS ${GO_LIBNAME})
set(GO_WHATSAPP_INCLUDE_DIRS "${CMAKE_CURRENT_BINARY_DIR}" PARENT_SCOPE)
set(GO_WHATSAPP_LIBRARIES "${CMAKE_CURRENT_BINARY_DIR}/libpurple-go-whatsapp.a" PARENT_SCOPE)
