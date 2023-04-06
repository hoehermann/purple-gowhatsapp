#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

/*
 * These functions implement some irc-style commands for advanced use in protocol bridges like Spectrum.
 */

static const char* command_string_presence = "/presence";

/*
 * Returns a non-zero value if the message looks like a command.
 */
enum gowhatsapp_command is_command(const char *message) {
    if (message[0] == '/') {
        if (g_str_has_prefix(message, "/versions")) {
            return GOWHATSAPP_COMMAND_VERSIONS;
        } else if (g_str_has_prefix(message, "/contacts")) {
            return GOWHATSAPP_COMMAND_CONTACTS;
        } else if (g_str_has_prefix(message, "/participants") || g_str_has_prefix(message, "/members")) {
            return GOWHATSAPP_COMMAND_PARTICIPANTS;
        } else if (g_str_has_prefix(message, command_string_presence)) {
            return GOWHATSAPP_COMMAND_PRESENCE;
        } else if (g_str_has_prefix(message, "/logout")) {
            return GOWHATSAPP_COMMAND_LOGOUT;
        }
    }
    return GOWHATSAPP_COMMAND_NONE;
}

/*
 * Populate list of participants for a specific conversation. For group chats only.
 */
static int execute_command_participants(PurpleAccount *account, const gchar *message, const gchar *who, PurpleConversation *conv) {
    PurpleConvChat *conv_chat = NULL;
    if (who == NULL || conv == NULL || (conv_chat = purple_conversation_get_chat_data(conv)) == NULL) {
        purple_debug_warning(GOWHATSAPP_NAME, "Trying to execute command /participants with incomplete data. who: %s, conv_chat: %p\n", who, conv_chat);
        return -1;
    } else {
        char **participants = gowhatsapp_go_query_group_participants(account, (char *)who);
        gowhatsapp_chat_set_participants(conv_chat, participants);
        g_strfreev(participants);
        return 0;
    }
}

/*
 * Overrides the outgoing presence string.
 * 
 * This means the plug-ins active state can differ from the presence advertised to WhatsApp servers.
 */
static int execute_command_presence(PurpleAccount *account, PurpleConnection *pc, const gchar *message) {
    if (g_str_has_prefix(message, command_string_presence)) {
        const char *presence_argument = message+strlen(command_string_presence);
        if (purple_strequal(presence_argument, GOWHATSAPP_STATUS_STR_AVAILABLE)) {
            purple_account_set_string(account, GOWHATSAPP_PRESENCE_OVERRIDE_KEY, GOWHATSAPP_STATUS_STR_AVAILABLE);
        } else if (purple_strequal(presence_argument, GOWHATSAPP_STATUS_STR_AWAY)) {
            purple_account_set_string(account, GOWHATSAPP_PRESENCE_OVERRIDE_KEY, GOWHATSAPP_STATUS_STR_AWAY);
        } else {
            purple_account_set_string(account, GOWHATSAPP_PRESENCE_OVERRIDE_KEY, NULL);
        }
        gowhatsapp_set_presence(account, purple_account_get_active_status(account));
        return 0;
    } else {
        return -1;
    }
}

static void conversation_write_versions(PurpleAccount *account, const gchar *who, PurpleConversation *conv) {
    if (NULL == conv) {
        conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, who, account);
    }
    const gchar *version = purple_plugin_get_version(purple_find_prpl(purple_account_get_protocol_id(account)));
    GHashTable *ui_info = purple_core_get_ui_info();
    const gchar *ui_version = g_hash_table_lookup(ui_info, "version");
    const gchar *ui_name = g_hash_table_lookup(ui_info, "name");
    char *msg = g_strdup_printf("This is %s %s built against purple %d.%d.%d running on purple %d.%d.%d. Host application is %s %s.", GOWHATSAPP_PRPL_ID, version, PURPLE_MAJOR_VERSION, PURPLE_MINOR_VERSION, PURPLE_MICRO_VERSION, purple_major_version, purple_minor_version, purple_micro_version, ui_name, ui_version);
    PurpleMessageFlags flags = PURPLE_MESSAGE_RECV|PURPLE_MESSAGE_NO_LOG;
    purple_conversation_write(conv, who, msg, flags, time(NULL));
    g_free(msg);
}

/*
 * Execute a command in the context of a conversation.
 */
int execute_command(PurpleConnection *pc, const gchar *message, const gchar *who, PurpleConversation *conv) {
    PurpleAccount *account = purple_connection_get_account(pc);
    switch (is_command(message)) {
        case GOWHATSAPP_COMMAND_VERSIONS: {
            conversation_write_versions(account, who, conv);
        } break;
        case GOWHATSAPP_COMMAND_CONTACTS: {
            gowhatsapp_go_get_contacts(account);
        } break;
        case GOWHATSAPP_COMMAND_PARTICIPANTS: {
            return execute_command_participants(account, message, who, conv);
        } break;
        case GOWHATSAPP_COMMAND_PRESENCE: {
            execute_command_presence(account, pc, message);
        } break;
        case GOWHATSAPP_COMMAND_LOGOUT: {
            gowhatsapp_go_logout(account);
        } break;
        default:
            // not a command – that is an error
            return -1;
    }
    return 0;
}
