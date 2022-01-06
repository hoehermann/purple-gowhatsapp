#include "gowhatsapp.h"

void gowhatsapp_join_chat(PurpleConnection *pc, GHashTable *data) {
    const char *remoteJid = g_hash_table_lookup(data, "remoteJid");
    const char *topic = g_hash_table_lookup(data, "topic");
    if (remoteJid != NULL) {
        gowhatsapp_find_group_chat(remoteJid, NULL, topic, pc);
    }
}

/*
 * Get the name of a chat (as passed to serv_got_joined_chat) given the
 * chat_info entries. For us, this is the room id.
 *
 * Borrowed from:
 * https://github.com/matrix-org/purple-matrix/blob/master/libmatrix.c
 */
char *gowhatsapp_get_chat_name(GHashTable *components)
{
    const char *jid = g_hash_table_lookup(components, "remoteJid");
    return g_strdup(jid);
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

/*
 * This represents all group chats as a list of rooms.
 * Each room also has two fields:
 * 0. remoteJid (WhatsApp identifier)
 * 1. topic (Human readable topic)
 * Order is important, see gowhatsapp_roomlist_serialize.
 */
PurpleRoomlist *
gowhatsapp_roomlist_get_list(PurpleConnection *pc) {
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleRoomlist *roomlist = purple_roomlist_new(account);

    // TODO: free list?
    GList *fields = NULL;
    // order is important (see description)
    fields = g_list_append(fields, purple_roomlist_field_new(
        PURPLE_ROOMLIST_FIELD_STRING, "Remote JID", "remoteJid", FALSE
    ));
    fields = g_list_append(fields, purple_roomlist_field_new(
        PURPLE_ROOMLIST_FIELD_STRING, "Topic", "topic", FALSE
    ));

    purple_roomlist_set_fields(roomlist, fields);

    PurpleBuddyList *bl = purple_get_blist();
    PurpleBlistNode *group;

    // loop outline borrowed from
    // https://github.com/hoehermann/libpurple-signald/blob/master/groups.c
    for (group = bl->root; group != NULL; group = group->next) {
        PurpleBlistNode *node;
        for (node = group->child; node != NULL; node = node->next) {
            if (PURPLE_BLIST_NODE_IS_CHAT(node)) {
                PurpleChat *chat = (PurpleChat *)node;

                if (account != purple_chat_get_account(chat)) {
                    continue;
                }

                char *jid = g_hash_table_lookup(chat->components, "remoteJid");
                char *topic = g_hash_table_lookup(chat->components, "topic");

                PurpleRoomlistRoom *room = purple_roomlist_room_new(
                    PURPLE_ROOMLIST_ROOMTYPE_ROOM,
                    jid,
                    NULL
                );

                // order is important (see description)
                purple_roomlist_room_add_field(roomlist, room, jid);
                purple_roomlist_room_add_field(roomlist, room, topic);

                purple_roomlist_room_add(roomlist, room);
            }
        }
    }

    purple_roomlist_set_in_progress(roomlist, FALSE);

    return roomlist;
}

/*
 * This gets the identifying information (remoteJid) from a room (WhatsApp group chat).
 * This assumes gowhatsapp_roomlist_get_list filled field at index 0 with the remoteJid.
 */
gchar *
gowhatsapp_roomlist_serialize(PurpleRoomlistRoom *room) {
    GList *fields = purple_roomlist_room_get_fields(room);
    const gchar *remoteJid = g_list_nth_data(fields, 0);
    return g_strdup(remoteJid);
}

/*
 * Borrowed from
 * https://github.com/hoehermann/libpurple-signald/blob/master/groups.c
 */
void
gowhatsapp_set_chat_topic(PurpleConnection *pc, int id, const char *topic)
{
    // Nothing to do here. For some reason this callback has to be
    // registered if Pidgin is going to enable the "Alias..." menu
    // option in the conversation.
}
