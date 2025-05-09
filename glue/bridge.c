#include "gowhatsapp.h"
#include "constants.h"
#include "../bridge.h"

/*
 * This is the C/gtk side of the go → C communication.
 */

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * Basic message processing.
 * Log messages are always processed.
 * Queries Pidgin for a list of all accounts.
 * Ignores message if no appropriate connection exists.
 */
static void process_message(gowhatsapp_message_t * gwamsg) {
    if (gwamsg->msgtype == gowhatsapp_message_type_log) {
        // log messages do not need an active connection
        purple_debug(gwamsg->subtype, GOWHATSAPP_NAME, "%s", gwamsg->text);
        return;
    }
    int account_exists = gowhatsapp_account_exists(gwamsg->account);
    if (account_exists == 0) {
        purple_debug_warning(GOWHATSAPP_NAME, "No account %p. Ignoring message.\n", gwamsg->account);
        return;
    }
    PurpleConnection *connection = purple_account_get_connection(gwamsg->account);
    if (connection == NULL) {
        purple_debug_warning(GOWHATSAPP_NAME, "No active connection for account %p. Ignoring message.\n", gwamsg->account);
        return;
    }
    gowhatsapp_process_message(gwamsg);
}

/*
 * Handler for a message received by go-whatsapp.
 * Called inside of the GTK eventloop.
 * Releases almost all memory allocated by CGO on heap.
 *
 * @return Whether to execute again. Always FALSE.
 */
gboolean process_message_bridge(gpointer data) {
    gowhatsapp_message_t * gwamsg = (gowhatsapp_message_t *)data;
    process_message(gwamsg);
    // always clean up data in heap
    g_free(gwamsg->remoteJid);
    g_free(gwamsg->senderJid);
    g_free(gwamsg->text);
    g_free(gwamsg->name);
    //g_free(gwamsg->blob); this is cleared after handling the attachment / qrcode
    g_strfreev(gwamsg->participants);
    g_free(gwamsg);
    return FALSE;
}
