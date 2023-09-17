#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

static int
send_message(PurpleConnection *pc, const gchar *who, const gchar *message, gboolean is_group) 
{
    // strip html similar to these reasons: https://github.com/majn/telegram-purple/issues/12 and https://github.com/majn/telegram-purple/commit/fffe7519d7269cf4e5029a65086897c77f5283ac
    char *msg = purple_markup_strip_html(message); // Note: This turns newlines into spaces and <br> tags into newlines
    PurpleAccount *account = purple_connection_get_account(pc);
    char *w = (char *)who; // cgo does not suport const
    return gowhatsapp_go_send_message(account, w, msg, is_group);
}

int
gowhatsapp_send_im(PurpleConnection *pc, const gchar *who, const gchar *message, PurpleMessageFlags flags){
    if (is_command(message)) {
        return execute_command(pc, message, who, NULL);
    } else {
        return send_message(pc, who, message, FALSE);
    }
}

int
gowhatsapp_send_chat(
    PurpleConnection *pc, int id, const gchar *message, PurpleMessageFlags flags
) {
    PurpleConversation *conv = purple_find_chat(pc, id);
    if (conv != NULL) {
        gchar *who = (gchar *)purple_conversation_get_data(conv, "name");
        if (who != NULL) {
            if (is_command(message)) {
                return execute_command(pc, message, who, conv);
            } else {
                int ret = send_message(pc, who, message, TRUE);
                if (ret > 0) {
                    // Group chats need an explicit local echo since the implicit echo is implemented for direct messages only.
                    // See https://keep.imfreedom.org/pidgin/pidgin/file/v2.14.12/libpurple/conversation.c#l191.
                    PurpleConvChat *conv_chat = purple_conversation_get_chat_data(conv);
                    PurpleAccount *account = purple_conversation_get_account(conv);
                    purple_conv_chat_write(conv_chat, purple_account_get_username(account), message, flags, time(NULL));
                }
                return ret;
            }
        }
    }
    return -6; // a negative value to indicate failure. chose ENXIO "no such address"
}
