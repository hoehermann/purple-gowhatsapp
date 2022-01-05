#include "gowhatsapp.h"

/*
 * Interprets a message received from whatsmeow. Handles login success and failure. Forwards errors.
 */
void
gowhatsapp_process_message(PurpleAccount *account, gowhatsapp_message_t *gwamsg)
{
    purple_debug_info(
        "gowhatsapp", "recieved message type %d for user %s at %ld id %s remote %s sender %s (fromMe %d, system %d): %s\n",
        (int)gwamsg->msgtype,
        gwamsg->username,
        gwamsg->timestamp,
        gwamsg->id,
        gwamsg->remoteJid,
        gwamsg->senderJid,
        gwamsg->fromMe,
        gwamsg->system,
        gwamsg->text
    );

    PurpleConnection *pc = purple_account_get_connection(account);

    if (!gwamsg->timestamp) {
        gwamsg->timestamp = time(NULL);
    }
    switch(gwamsg->msgtype) {
        case gowhatsapp_message_type_error:
            purple_connection_error_reason(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, gwamsg->text);
            break;
        case gowhatsapp_message_type_login:
            gowhatsapp_display_qrcode(pc, gwamsg->text, gwamsg->blob, gwamsg->blobsize);
            break;
        case gowhatsapp_message_type_text:
            //gowhatsapp_display_message(pc, gwamsg);
            break;
        default:
            purple_debug_info("gowhatsapp", "handling this message type is not implemented");
    }
}
