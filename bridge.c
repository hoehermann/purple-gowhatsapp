#include "bridge.h"
/*
 * These are functions that will be called from the go part.
 */

/*
 * Whether the given pointer actually refers to an existing account.
 */
int gowhatsapp_account_exists(PurpleAccount *account) {
    int account_exists = 0;
    // this would be more elegant, but bitlbee does not implement purple_accounts_get_all()
    // see https://github.com/hoehermann/purple-gowhatsapp/issues/102
    // for (GList *iter = purple_accounts_get_all(); iter != NULL && account_exists == 0; iter = iter->next) {
    //     PurpleAccount * acc = (PurpleAccount *)iter->data;
    //     account_exists = acc == account;
    // }
    for (GList *iter = purple_connections_get_connecting(); iter != NULL && account_exists == 0; iter = iter->next) {
        PurpleAccount * acc = purple_connection_get_account(iter->data);
        account_exists = acc == account;
    }
    for (GList *iter = purple_connections_get_all(); iter != NULL && account_exists == 0; iter = iter->next) {
        PurpleAccount * acc = purple_connection_get_account(iter->data);
        account_exists = acc == account;
    }
    return account_exists;
}

#if !GLIB_CHECK_VERSION(2, 68, 0)
#define g_memdup2 g_memdup
#endif

/*
 * Handler for a message received by go-whatsapp.
 * Called by go-whatsapp (outside of the GTK eventloop).
 * 
 * Yes, this is indeed neccessary – we checked.
 */
void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg_go) {
    // copying Go-managed struct into heap
    // the strings inside the struct already reside in the heap, according to https://golang.org/cmd/cgo/#hdr-C_references_to_Go
    gowhatsapp_message_t *gwamsg_heap = g_memdup2(&gwamsg_go, sizeof gwamsg_go);
    purple_timeout_add(
        0, // schedule for immediate execution
        process_message_bridge, // handle message in main thread
        gwamsg_heap // data to handle in main thread
    );
}
