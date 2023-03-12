#include "gowhatsapp.h"
#include "constants.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_query_contacts

static const char *gowhatsapp_message_type_string[] = {
    FOREACH_MESSAGE_TYPE(GENERATE_STRING)
};

/*
 * Interprets a message received from whatsmeow. Handles login success and failure. Forwards errors.
 */
void
gowhatsapp_process_message(gowhatsapp_message_t *gwamsg)
{
    if (gwamsg->msgtype < 0 || gwamsg->msgtype >= gowhatsapp_message_type_max) {
        purple_debug_info(GOWHATSAPP_NAME, "recieved invalid message type %d.\n", gwamsg->msgtype);
        return;
    }
    purple_debug_info(
        GOWHATSAPP_NAME, "recieved %s (subtype %d) for account %p remote %s (isGroup %d) sender %s (alias %s, isOutgoing %d) sent %ld: %s\n",
        gowhatsapp_message_type_string[gwamsg->msgtype],
        gwamsg->subtype,
        gwamsg->account,
        gwamsg->remoteJid,
        gwamsg->isGroup,
        gwamsg->senderJid,
        gwamsg->name,
        gwamsg->isOutgoing,
        gwamsg->timestamp,
        gwamsg->text
    );

    PurpleConnection *pc = purple_account_get_connection(gwamsg->account);

    if (!gwamsg->timestamp) {
        gwamsg->timestamp = time(NULL);
    }
    switch(gwamsg->msgtype) {
        case gowhatsapp_message_type_error:
            if (gwamsg->subtype == 0) {
                purple_connection_error(pc, PURPLE_CONNECTION_ERROR_NETWORK_ERROR, gwamsg->text);
            } else {
                purple_connection_error(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, gwamsg->text);
            }
            gowhatsapp_close_qrcode(gwamsg->account);
            break;
        case gowhatsapp_message_type_login:
            gowhatsapp_handle_qrcode(pc, gwamsg);
            break;
        case gowhatsapp_message_type_pairing_succeeded:
            gowhatsapp_close_qrcode(gwamsg->account);
            break;
        case gowhatsapp_message_type_credentials:
            gowhatsapp_store_credentials(gwamsg->account, gwamsg->text);
            break;
        case gowhatsapp_message_type_connected:
            gowhatsapp_close_qrcode(gwamsg->account);
            purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTED);
            gowhatsapp_set_presence(gwamsg->account, purple_account_get_active_status(gwamsg->account));
            gowhatsapp_assume_all_buddies_online(gwamsg->account); // this affects existing buddies only
            // after connecting, fetch contacts
            // calling it after gowhatsapp_assume_all_buddies_online is fine,
            // since the results are handled asynchronously anyway
            gowhatsapp_go_get_contacts(gwamsg->account);
            // NOTE: It looks like Spectrum wants all contacts to be updated before entering any group chat.
            // this is not guaranteed due to the asynchronous nature of request handling
            gowhatsapp_roomlist_get_list(pc);
            break;
        case gowhatsapp_message_type_disconnected:
            purple_connection_set_state(pc, PURPLE_CONNECTION_DISCONNECTED);
            gowhatsapp_close_qrcode(gwamsg->account);
            break;
        case gowhatsapp_message_type_text:
            gowhatsapp_display_text_message(pc, gwamsg, 0);
            break;
        case gowhatsapp_message_type_system:
            gowhatsapp_display_text_message(pc, gwamsg, PURPLE_MESSAGE_SYSTEM);
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
            gowhatsapp_handle_presence(gwamsg->account, gwamsg->remoteJid, gwamsg->subtype, gwamsg->timestamp);
            break;
        case gowhatsapp_message_type_attachment:
            gowhatsapp_handle_attachment(pc, gwamsg);
            break;
        case gowhatsapp_message_type_profile_picture:
            gowhatsapp_handle_profile_picture(gwamsg);
            break;
        case gowhatsapp_message_type_group:
            gowhatsapp_handle_group(pc, gwamsg);
            break;
        default:
            purple_debug_info(GOWHATSAPP_NAME, "handling this message type is not implemented");
            g_free(gwamsg->blob);
    }
}
