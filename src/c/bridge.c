#include "gowhatsapp.h"
#include "constants.h"

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * Look up the account by username
 */
PurpleAccount *
gowhatsapp_get_account(char *username)
{
    PurpleAccount *account = NULL;
    for (GList *iter = purple_accounts_get_all(); iter != NULL && account == NULL; iter = iter->next) {
        PurpleAccount * acc = (PurpleAccount *)iter->data;
        const char *protocol_id = purple_account_get_protocol_id(acc);
        const char *u = purple_account_get_username(acc);
        if (purple_strequal(protocol_id, GOWHATSAPP_PRPL_ID) && purple_strequal(username, u)) {
            account = acc;
        }
    }
    return account;
}

/*
 * Basic message processing.
 * Log messages are always processed.
 * Queries Pidgin for a list of all accounts.
 * Ignores message if no appropriate connection exists.
 */
static void
process_message(gowhatsapp_message_t * gwamsg) {
    if (gwamsg->msgtype == gowhatsapp_message_type_log) {
        // log messages do not need an active connection
        purple_debug(gwamsg->level, GOWHATSAPP_NAME, "%s", gwamsg->text);
        return;
    }
    // 
    PurpleAccount *account = gowhatsapp_get_account(gwamsg->username);
    if (account == NULL) {
        purple_debug_warning(GOWHATSAPP_NAME, "No account for username %s. Ignoring message.\n", gwamsg->username);
        return;
    }
    PurpleConnection *connection = purple_account_get_connection(account);
    if (connection == NULL) {
        purple_debug_info(GOWHATSAPP_NAME, "No active connection for account %s. Ignoring message.\n", gwamsg->username);
        return;
    }
    gowhatsapp_process_message(account, gwamsg);
}

/*
 * Handler for a message received by go-whatsapp.
 * Called inside of the GTK eventloop.
 * Releases almost all memory allocated by CGO on heap.
 *
 * @return Whether to execute again. Always FALSE.
 */
static gboolean
process_message_bridge(gpointer data)
{
    gowhatsapp_message_t * gwamsg = (gowhatsapp_message_t *)data;
    process_message(gwamsg);
    // always clean up data in heap
    g_free(gwamsg->remoteJid);
    g_free(gwamsg->senderJid);
    g_free(gwamsg->text);
    g_free(gwamsg->name);
    //g_free(gwamsg->blob); this is cleared after handling the attachment / qrcode
    g_free(gwamsg);
    return FALSE;
}

#if !(GLIB_CHECK_VERSION(2, 67, 3))
#define g_memdup2 g_memdup
#endif

/*
 * Handler for a message received by go-whatsapp.
 * Called by go-whatsapp (outside of the GTK eventloop).
 * 
 * Yes, this is indeed neccessary – we checked.
 */
void
gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg_go)
{
    // copying Go-managed struct into heap
    // the strings inside the struct already reside in the heap, according to https://golang.org/cmd/cgo/#hdr-C_references_to_Go
    gowhatsapp_message_t *gwamsg_heap = g_memdup2(&gwamsg_go, sizeof gwamsg_go);
    purple_timeout_add(
        0, // schedule for immediate execution
        process_message_bridge, // handle message in main thread
        gwamsg_heap // data to handle in main thread
    );
}
