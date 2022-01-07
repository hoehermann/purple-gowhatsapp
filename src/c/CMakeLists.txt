if(WIN32 AND NOT ${CMAKE_SIZEOF_VOID_P} EQUAL 4)
    message(FATAL_ERROR "On Windows, Pidgin is 32 bit only.")
endif()

set(TARGET_NAME gowhatsapp)
add_library(${TARGET_NAME} SHARED 
    "gowhatsapp.h"
    "init.c"
    "login.c"
    "qrcode.c"
    "bridge.c"
    "process_message.c"
    "display_message.c"
    "groups.c"
    "blist.c"
    "constants.c"
    "message_filtering.c"
    "send_message.c"
    "handle_attachment.c"
    "send_file.c"
    "msc_adjustments.c"
)
add_dependencies(${TARGET_NAME} purple-go-whatsapp)

file(READ "${CMAKE_SOURCE_DIR}/VERSION" PLUGIN_VERSION)
target_compile_definitions(${TARGET_NAME} PRIVATE PLUGIN_VERSION=${PLUGIN_VERSION})
target_include_directories(${TARGET_NAME} PRIVATE ${GO_WHATSAPP_INCLUDE_DIRS} ${PURPLE_INCLUDE_DIRS})
target_link_directories(${TARGET_NAME} PRIVATE ${PURPLE_LIBRARY_DIRS})
target_link_libraries(${TARGET_NAME} PRIVATE ${GO_WHATSAPP_LIBRARIES} ${PURPLE_LIBRARIES})
set_target_properties(${TARGET_NAME} PROPERTIES PREFIX "lib")

message(STATUS "PURPLE_PLUGIN_DIR: ${PURPLE_PLUGIN_DIR}")
if (WIN32)
install(TARGETS ${TARGET_NAME} RUNTIME DESTINATION "${PURPLE_PLUGIN_DIR}")
else()
install(TARGETS ${TARGET_NAME} DESTINATION "${PURPLE_PLUGIN_DIR}")
endif()
