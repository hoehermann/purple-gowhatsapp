# Dummy must be built by cgo-compatible compiler (gcc or clang). Assume gcc on Windows.
if(MSVC)
add_custom_command(
  OUTPUT purple-dummy.o
  DEPENDS purple-dummy.c
  COMMAND gcc -m32 -o "${CMAKE_CURRENT_BINARY_DIR}/purple-dummy.o" -c "${CMAKE_CURRENT_SOURCE_DIR}/purple-dummy.c"
)
add_custom_target(purple-dummy DEPENDS purple-dummy.o)
set(PURPLE_DUMMY_OBJ "${CMAKE_CURRENT_BINARY_DIR}/purple-dummy.o")
else()
add_library(purple-dummy OBJECT purple-dummy.c)
set(PURPLE_DUMMY_OBJ "${CMAKE_CURRENT_BINARY_DIR}/CMakeFiles/purple-dummy.dir/purple-dummy.c.o") # TODO: get this path properly
endif()

# assume go is in path
set(CMAKE_Go_COMPILER go)
# have a self-contained GOPATH
set(GOPATH "${CMAKE_CURRENT_BINARY_DIR}/go")

# hack to depend on latest version using go mod tidy
add_custom_command(
  OUTPUT "${CMAKE_CURRENT_SOURCE_DIR}/go.mod" "${CMAKE_CURRENT_SOURCE_DIR}/go.sum"
  DEPENDS "${CMAKE_CURRENT_SOURCE_DIR}/go.mod.in"
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E copy "${CMAKE_CURRENT_SOURCE_DIR}/go.mod.in" "${CMAKE_CURRENT_SOURCE_DIR}/go.mod"
  COMMAND ${CMAKE_COMMAND} -E env GOPATH=${GOPATH} ${CMAKE_Go_COMPILER} mod tidy
)

# build cgo library
# partly taken from https://github.com/mutse/go-cmake
set(GO_SRCS
  bridge.go
  init.go
  login.go
  handler.go
  send_message.go
  handle_message.go
  logger.go
  send_file.go
)
set(OUTPUT_PREFIX purple-go-whatsapp)
add_custom_command(
  OUTPUT ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h
  DEPENDS purple-dummy ${GO_SRCS} "${CMAKE_CURRENT_SOURCE_DIR}/go.mod"
  WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}"
  COMMAND ${CMAKE_COMMAND} -E env
  GOPATH=${GOPATH}
  CGO_LDFLAGS="${PURPLE_DUMMY_OBJ}"
  ${CMAKE_Go_COMPILER} build -buildmode=c-archive
  -o "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a"
  ${CMAKE_GO_FLAGS} ${GO_SRCS}
)
add_custom_target(${OUTPUT_PREFIX} DEPENDS ${OUTPUT_PREFIX}.a ${OUTPUT_PREFIX}.h)

# prepare output file paths for inclusion in C part
set(GO_WHATSAPP_INCLUDE_DIRS "${CMAKE_CURRENT_BINARY_DIR}" "${CMAKE_CURRENT_SOURCE_DIR}" PARENT_SCOPE)
set(GO_WHATSAPP_LIBRARIES "${CMAKE_CURRENT_BINARY_DIR}/${OUTPUT_PREFIX}.a" PARENT_SCOPE)
