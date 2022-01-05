#include "gowhatsapp.h"

/*
 * Determines if given jid denotes a group chat.
 */
int gowhatsapp_remotejid_is_group_chat(char *remoteJid) {
    char *suffix = strrchr(remoteJid, '@');
    if (suffix != NULL)
        return strcmp(suffix, "@g.us") == 0;
    return FALSE;
}

/*
 * Determines if user is participating in conversation.
 */
int gowhatsapp_user_in_conv_chat(PurpleConvChat *conv_chat, const char *userJid) {
    GList *users = purple_conv_chat_get_users(conv_chat);

    while (users != NULL) {
        PurpleConvChatBuddy *buddy = (PurpleConvChatBuddy *) users->data;
        if (!strcmp(buddy->name, userJid)) {
            return TRUE;
        }
        users = users->next;
    }

    return FALSE;
}

/*
 * Topic is the topic of the chat if not already known (may be null),
 * senderJid may be null, but else is used to add users sending messages
 * to the chats.
 */
PurpleConvChat *gowhatsapp_find_group_chat(
    const char *remoteJid,
    const char *senderJid,
    const char *topic,
    PurpleConnection *pc
) {
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleConvChat *conv_chat =
        purple_conversations_find_chat_with_account(remoteJid, account);

    if (conv_chat == NULL) {
        gowhatsapp_ensure_group_chat_in_blist(account, remoteJid, topic);

        // use hash of jid for chat id number
        PurpleConversation *conv = serv_got_joined_chat(
            pc, g_str_hash(remoteJid), remoteJid
        );

        if (conv != NULL) {
            purple_conversation_set_data(conv, "remoteJid", g_strdup(remoteJid));
        }

        conv_chat = PURPLE_CONV_CHAT(conv);
    }

    if (conv_chat != NULL && senderJid != NULL) {
        if (!gowhatsapp_user_in_conv_chat(conv_chat, senderJid)) {
            purple_chat_conversation_add_user(
                conv_chat, senderJid, NULL, PURPLE_CBFLAGS_NONE, FALSE
            );
        }
    }

    return conv_chat;
}
