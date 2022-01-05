if(MSVC)
add_custom_command(
  OUTPUT purple-dummy.o
  DEPENDS purple-dummy.c
  COMMAND gcc -m32 -o "${CMAKE_CURRENT_BINARY_DIR}/purple-dummy.o" -c "${CMAKE_CURRENT_SOURCE_DIR}/purple-dummy.c"
)
add_custom_target(purple-dummy DEPENDS purple-dummy.o)
set(PURPLE_DUMMY_OBJ "${CMAKE_CURRENT_BINARY_DIR}/purple-dummy.o") # on linux, this remains empty
else()
add_library(purple-dummy OBJECT purple-dummy.c)
endif()

# partly taken from https://github.com/mutse/go-cmake
set(CMAKE_Go_COMPILER go)
set(GOPATH "${CMAKE_CURRENT_BINARY_DIR}/go")
set(GO_SRCS
  bridge.go
  init.go
  login.go
)
set(OUTPUT_PREFIX purple-go-whatsapp)
add_custom_command(
  OUTPUT ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h
  DEPENDS purple-dummy ${GO_SRCS} go.mod
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env
  GOPATH=${GOPATH}
  CGO_LDFLAGS="${PURPLE_DUMMY_OBJ}"
  ${CMAKE_Go_COMPILER} build -buildmode=c-archive
  -o "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a"
  ${CMAKE_GO_FLAGS} ${GO_SRCS}
)
add_custom_target(${OUTPUT_PREFIX} DEPENDS ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h)
set(GO_WHATSAPP_INCLUDE_DIRS "${CMAKE_CURRENT_BINARY_DIR}" PARENT_SCOPE)
set(GO_WHATSAPP_LIBRARIES "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a" PARENT_SCOPE)
