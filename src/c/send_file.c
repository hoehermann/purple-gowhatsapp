#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

static void
gowhatsapp_free_xfer(PurpleXfer *xfer)
{
}

static void
gowhatsapp_xfer_send_init(PurpleXfer *xfer)
{
    PurpleAccount *account = purple_xfer_get_account(xfer);
    const char *who = purple_xfer_get_remote_user(xfer);
    const char *filename = purple_xfer_get_local_filename(xfer);
    char *error = gowhatsapp_go_send_file(account, (char *)who, (char *)filename);
    if (error && error[0]) {
        purple_xfer_error(purple_xfer_get_type(xfer), account, who, error); 
        purple_xfer_cancel_local(xfer);
    } else {
        purple_xfer_set_bytes_sent(xfer, purple_xfer_get_size(xfer));
        purple_xfer_set_completed(xfer, TRUE);
    }
    g_free(error);
}

PurpleXfer *
gowhatsapp_new_xfer(PurpleConnection *pc, const char *who)
{
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleXfer *xfer = purple_xfer_new(account, PURPLE_XFER_TYPE_SEND, who);
    purple_xfer_set_init_fnc(xfer, gowhatsapp_xfer_send_init);
    return xfer;
}

void
gowhatsapp_send_file(PurpleConnection *pc, const gchar *who, const gchar *filename)
{
    PurpleXfer *xfer = gowhatsapp_new_xfer(pc, who);

    if (filename && *filename) {
        purple_xfer_request_accepted(xfer, filename);
    } else {
        purple_xfer_request(xfer);
    }
}

void
gowhatsapp_chat_send_file(PurpleConnection *pc, int id, const char *filename)
{
    PurpleConversation *conv = purple_find_chat(pc, id);
    if (conv != NULL) {
        const gchar *who = purple_conversation_get_data(conv, "name");
        if (who != NULL) {
            gowhatsapp_send_file(pc, who, (gchar *)filename);
        }
    }
    // TODO: display error if conv or who are NULL
}
