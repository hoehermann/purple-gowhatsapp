set(TARGET_NAME whatsmeow)
add_library(${TARGET_NAME} SHARED 
    "gowhatsapp.h"
    "init.c"
    "login.c"
    "qrcode.c"
    "bridge.c"
    "../bridge.c" # this needs to be compiled again since go's c-archive mode does not include the C object
    "../bridge.h"
    "process_message.c"
    "display_message.c"
    "groups.c"
    "blist.c"
    "send_message.c"
    "handle_attachment.c"
    "send_file.c"
    "presence.c"
    "options.c"
    "receipt.c"
    "history.c"
    "pixbuf.c"
    "pixbuf.h"
    "commands.c"
)
add_dependencies(${TARGET_NAME} libwhatsmeow)

target_compile_definitions(${TARGET_NAME} PRIVATE PLUGIN_VERSION=${VERSION})
target_include_directories(${TARGET_NAME} PRIVATE ${GO_WHATSAPP_INCLUDE_DIRS} ${PURPLE_INCLUDE_DIRS} ${PIXBUF_INCLUDE_DIRS} ${OPUSFILE_INCLUDE_DIRS})
target_link_directories(${TARGET_NAME} PRIVATE ${PURPLE_LIBRARY_DIRS} ${PIXBUF_LIBRARY_DIRS})
target_link_libraries(${TARGET_NAME} PRIVATE ${GO_WHATSAPP_LIBRARIES} ${PURPLE_LIBRARIES} ${PIXBUF_LIBRARIES} ${OPUSFILE_LINK_LIBRARIES})
set_target_properties(${TARGET_NAME} PROPERTIES PREFIX "lib")
if (WIN32 AND CMAKE_C_COMPILER_ID STREQUAL GNU AND CMAKE_C_COMPILER_VERSION VERSION_GREATER 4.7.2) # might actually work without this up to 7.1.0, but I did not check
    target_link_options(${TARGET_NAME} PRIVATE "-static-libgcc")
endif()
if (APPLE)
    target_link_libraries(${TARGET_NAME} PRIVATE "-framework CoreFoundation" "-framework Security" resolv)
endif()

if (WIN32)
    install(TARGETS ${TARGET_NAME} RUNTIME DESTINATION "${PURPLE_PLUGIN_DIR}")
else()
    install(TARGETS ${TARGET_NAME} DESTINATION "${PURPLE_PLUGIN_DIR}")
endif()
