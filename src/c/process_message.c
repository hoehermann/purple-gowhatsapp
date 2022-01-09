#include "gowhatsapp.h"
#include "constants.h"

/*
 * Interprets a message received from whatsmeow. Handles login success and failure. Forwards errors.
 */
void
gowhatsapp_process_message(gowhatsapp_message_t *gwamsg)
{
    purple_debug_info(
        GOWHATSAPP_NAME, "recieved message type %d for account %p at %ld remote %s (alias %s, isGroup %d) sender %s (fromMe %d): %s\n",
        (int)gwamsg->msgtype,
        gwamsg->account,
        gwamsg->timestamp,
        gwamsg->remoteJid,
        gwamsg->name,
        gwamsg->isGroup,
        gwamsg->senderJid,
        gwamsg->fromMe,
        gwamsg->text
    );

    PurpleConnection *pc = purple_account_get_connection(gwamsg->account);

    if (!gwamsg->timestamp) {
        gwamsg->timestamp = time(NULL);
    }
    switch(gwamsg->msgtype) {
        case gowhatsapp_message_type_error:
            purple_connection_error(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, gwamsg->text);
            break;
        case gowhatsapp_message_type_login:
            gowhatsapp_handle_qrcode(pc, gwamsg->text, gwamsg->name, gwamsg->blob, gwamsg->blobsize);
            break;
        case gowhatsapp_message_type_connected:
            gowhatsapp_close_qrcode(gwamsg->account);
            purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTED);
            gowhatsapp_assume_all_buddies_online(gwamsg->account);
            break;
        case gowhatsapp_message_type_disconnected:
            //purple_connection_set_state(pc, PURPLE_CONNECTION_DISCONNECTED);
            // during development, I want this to fail louder
            purple_connection_error(pc, PURPLE_CONNECTION_ERROR_NETWORK_ERROR, "Disconnected");
            break;
        case gowhatsapp_message_type_text:
            gowhatsapp_display_text_message(pc, gwamsg);
            break;
        case gowhatsapp_message_type_name:
            gowhatsapp_ensure_buddy_in_blist(gwamsg->account, gwamsg->remoteJid, gwamsg->name);
            break;
        case gowhatsapp_message_type_typing:
            serv_got_typing(pc, gwamsg->remoteJid, 0, PURPLE_TYPING);
            break;
        case gowhatsapp_message_type_typing_stopped:
            serv_got_typing_stopped(pc, gwamsg->remoteJid);
            break;
        case gowhatsapp_message_type_presence:
            gowhatsapp_handle_presence(gwamsg->account, gwamsg->remoteJid, gwamsg->level, gwamsg->timestamp);
            break;
        case gowhatsapp_message_type_attachment:
            gowhatsapp_handle_attachment(pc, gwamsg);
            break;
        default:
            purple_debug_info(GOWHATSAPP_NAME, "handling this message type is not implemented");
    }
}
