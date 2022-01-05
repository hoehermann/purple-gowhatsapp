#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * Handler for a message received by go-whatsapp.
 * Called inside of the GTK eventloop.
 *
 * @return Whether to execute again. Always FALSE.
 */
static gboolean
process_message_bridge(gpointer data)
{
    // query Pidgin for a list of all connections. bail if it does not exist
    gowhatsapp_message_t * gwamsg = (gowhatsapp_message_t *)data;
    PurpleConnection *pc = (PurpleConnection *)gwamsg->connection;
    int connection_exists = 0;
    {
        GList * connection = purple_connections_get_connecting();
        while (connection != NULL && connection_exists == 0) {
            connection_exists = connection->data == pc;
            connection = connection->next;
        }
    }
    {
        GList * connection = purple_connections_get_all();
        while (connection != NULL && connection_exists == 0) {
            connection_exists = connection->data == pc;
            connection = connection->next;
        }
    }
    if (connection_exists == 0) {
        purple_debug_info(
            "gowhatsapp", "Avoiding crash by not handling message for not-existant connection %p.\n", pc
        );
    } else {
        //TODO: gowhatsapp_process_message(gwamsg);
    }
    g_free(gwamsg->id);
    g_free(gwamsg->remoteJid);
    g_free(gwamsg->senderJid);
    g_free(gwamsg->text);
    g_free(gwamsg->blob);
    /*
    g_free(gwamsg->clientId);
    g_free(gwamsg->clientToken);
    g_free(gwamsg->serverToken);
    g_free(gwamsg->encKey_b64);
    g_free(gwamsg->macKey_b64);
    g_free(gwamsg->wid);
    */
    g_free(gwamsg);
    return FALSE;
}

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
    gowhatsapp_message_t *gwamsg_heap = g_memdup(&gwamsg_go, sizeof gwamsg_go);
    purple_timeout_add(
        0, // schedule for immediate execution
        process_message_bridge, // handle message in main thread
        gwamsg_heap // data to handle in main thread
    );
}
