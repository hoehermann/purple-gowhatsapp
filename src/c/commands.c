#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

enum gowhatsapp_command is_command(const char *message) {
    if (message[0] == '/') {
        if (g_str_has_prefix(message, "/contacts")) {
            return GOWHATSAPP_COMMAND_CONTACTS;
        } else if (g_str_has_prefix(message, "/participants") || g_str_has_prefix(message, "/members")) {
            return GOWHATSAPP_COMMAND_PARTICIPANTS;
        }
    }
    return GOWHATSAPP_COMMAND_NONE;
}

int execute_command(PurpleConnection *pc, const gchar *message, const gchar *who, PurpleConversation *conv) {
    PurpleAccount *account = purple_connection_get_account(pc);
    switch (is_command(message)) {
        case GOWHATSAPP_COMMAND_CONTACTS: {
            gowhatsapp_go_get_contacts(account);
        } break;
        case GOWHATSAPP_COMMAND_PARTICIPANTS: {
            PurpleConvChat *conv_chat = NULL;
            if (who == NULL || conv == NULL || (conv_chat = purple_conversation_get_chat_data(conv)) == NULL) {
                purple_debug_warning(GOWHATSAPP_NAME, "Trying to execute command /participants with incomplete data. who: %s, conv_chat: %p\n", who, conv_chat);
            } else {
                char **participants = gowhatsapp_go_query_group_participants(account, (char *)who);
                gowhatsapp_chat_set_participants(conv_chat, participants);
                g_strfreev(participants);
            }
        } break;
        default:
            // do nothing
    }
    return 0;
}
