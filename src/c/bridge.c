#include "gowhatsapp.h"
#include "constants.h"

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * Whether the given pointer actually refers to an existing account.
 */
int
gowhatsapp_account_exists(PurpleAccount *account)
{
    int account_exists = 0;
    for (GList *iter = purple_accounts_get_all(); iter != NULL && account_exists == 0; iter = iter->next) {
        PurpleAccount * acc = (PurpleAccount *)iter->data;
        account_exists = acc == account;
    }
    return account_exists;
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
    int account_exists = gowhatsapp_account_exists(gwamsg->account);
    if (account_exists == 0) {
        purple_debug_warning(GOWHATSAPP_NAME, "No account %p. Ignoring message.\n", gwamsg->account);
        return;
    }
    PurpleConnection *connection = purple_account_get_connection(gwamsg->account);
    if (connection == NULL) {
        purple_debug_info(GOWHATSAPP_NAME, "No active connection for account %p. Ignoring message.\n", gwamsg->account);
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
