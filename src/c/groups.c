#include "gowhatsapp.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_query_groups

// Core functions for working with chats in purple

/*
 * According to libpurple/prpl.h, this shall return a list of identifiers 
 * needed to join a group chat. By default, the first element of this list 
 * must be the identifying aspect, see purple_blist_find_chat in 
 * libpurple/blist.c. This is explicitly reimplemented by gowhatsapp_find_blist_chat.
 * For compatibility reasons, the WhatsApp JID remains the first element.
 * 
 * bitlbee expects this function to be present.
 * 
 * Borrowed from
 * https://github.com/hoehermann/libpurple-signald/blob/master/groups.c
 */
GList * gowhatsapp_chat_info(PurpleConnection *pc)
{
    GList *infos = NULL;

    struct proto_chat_entry *pce;

    pce = g_new0(struct proto_chat_entry, 1);
    pce->label = "JID";
    pce->identifier = "name";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    pce = g_new0(struct proto_chat_entry, 1);
    pce->label = "Group Name";
    pce->identifier = "topic";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    return infos;
}

/*
 * Takes a purple chat_name and prepares all information necessary to join the chat with that name.
 * 
 * Ideally, this would take then human readable WhatsApp group name and look up the appropriate JID.
 * For now, it expects the chat_name to be the group JID and simply wraps it.
 * 
 * The "topic" field denoting the human readable WhatsApp group name remains empty
 * since we do not know it here.
 * 
 * Note: In purple, "name" is implicitly set to the roomlist room name in 
 * purple_roomlist_room_join, see libpurple/roomlist.c
 * 
 * bitlbee expects this function to be present.
 */
GHashTable * gowhatsapp_chat_info_defaults(PurpleConnection *pc, const char *chat_name) 
{
    GHashTable *defaults = g_hash_table_new_full(
        g_str_hash, g_str_equal, NULL, g_free
    );
    if (chat_name != NULL) {
        g_hash_table_insert(defaults, "name", g_strdup(chat_name));
        g_hash_table_insert(defaults, "topic", g_strdup(""));
    }
    return defaults;
}

/*
 * The user wants to join a chat.
 * 
 * data is a table filled with the information needed to join the chat
 * as defined by chat_info_defaults. We only need the JID.
 * 
 * Note: In purple, "name" is implicitly set to the roomlist room name in 
 * purple_roomlist_room_join, see libpurple/roomlist.c
 * 
 * Since group chat participation is handled by WhatsApp, this function
 * does not actually send any requests to the server.
 */
void gowhatsapp_join_chat(PurpleConnection *pc, GHashTable *data) {
    const char *remoteJid = g_hash_table_lookup(data, "name");
    if (remoteJid != NULL) {
        // add chat to buddy list (optional)
        PurpleAccount *account = purple_connection_get_account(pc);
        const char *topic = g_hash_table_lookup(data, "topic");
        gowhatsapp_ensure_group_chat_in_blist(account, remoteJid, topic);
        // create conversation (important)
        PurpleConvChat *chat = gowhatsapp_enter_group_chat(pc, remoteJid);
    }
}

// Functions for listing rooms

/*
 * This requests a list of rooms representing the WhatsApp group chats.
 * The request is asynchronous. Responses are handled by gowhatsapp_roomlist_add_room.
 * 
 * A purple room has an identifying name – for WhatsApp that is the JID.
 * A purple room has a list of fields – in our case only WhatsApp group name.
 * 
 * Some services like spectrum expect the human readable group name field key to be "topic", 
 * see RoomlistProgress in https://github.com/SpectrumIM/spectrum2/blob/518ba5a/backends/libpurple/main.cpp#L1997
 * In purple, the roomlist field "name" gets overwritten in purple_roomlist_room_join, see libpurple/roomlist.c.
 */
PurpleRoomlist *
gowhatsapp_roomlist_get_list(PurpleConnection *pc) {
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleRoomlist *roomlist = (PurpleRoomlist *)purple_connection_get_protocol_data(pc);
    if (roomlist != NULL) {
        purple_debug_info(GOWHATSAPP_NAME, "Already getting roomlist.");
        return roomlist;
    }
    roomlist = purple_roomlist_new(account);
    purple_roomlist_set_in_progress(roomlist, TRUE);
    GList *fields = NULL;
    fields = g_list_append(fields, purple_roomlist_field_new(
        PURPLE_ROOMLIST_FIELD_STRING, "Group Name", "topic", FALSE
    ));
    purple_roomlist_set_fields(roomlist, fields);
    purple_connection_set_protocol_data(pc, roomlist);
    gowhatsapp_go_query_groups(account);
    return roomlist;
}

/*
 * This handles incoming WhatsApp group information,
 * adding groups as rooms to the roomlist.
 * 
 * In case there currently is no roomlist to populate, this does nothing.
 */
void
gowhatsapp_roomlist_add_room(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    PurpleRoomlist *roomlist = (PurpleRoomlist *)purple_connection_get_protocol_data(pc);
    if (roomlist != NULL) {
        if (gwamsg->remoteJid == NULL) {
            purple_roomlist_set_in_progress(roomlist, FALSE);
            purple_roomlist_unref(roomlist);
            purple_connection_set_protocol_data(pc, NULL);
        } else {
            PurpleRoomlistRoom *room = purple_roomlist_room_new(PURPLE_ROOMLIST_ROOMTYPE_ROOM, gwamsg->remoteJid, NULL);
            purple_roomlist_room_add_field(roomlist, room, gwamsg->name);
            purple_roomlist_room_add(roomlist, room);
        }
    }
}

// Helper functions regarding the purple conversation representing a WhatsApp group

/*
 * This handles incoming WhatsApp group information,
 * adding participants to all chats currently active.
 */
void 
gowhatsapp_chat_add_participants(gowhatsapp_message_t *gwamsg) {
    PurpleConvChat *conv_chat = purple_conversations_find_chat_with_account(gwamsg->remoteJid, gwamsg->account);
    if (conv_chat != NULL) { // only consider active chats
        for(char **iter = gwamsg->participants; iter != NULL && *iter != NULL; iter++) {
            PurpleConvChatBuddyFlags flags = 0;
            purple_conv_chat_add_user(conv_chat, *iter, NULL, flags, FALSE);
        }
    }
}

/*
 * This returns the conversation representing a group chat.
 * 
 * It creates a new conversation if necessary.
 * 
 * purple uses an int to identify conversations, WhatsApp uses the JID.
 * For this reason, the purple chat id is g_str_hash(remoteJid).
 * 
 * The actual JID is stored in the conversation data hash table so it can 
 * be retrieved by get_chat_name (see below).
 */
PurpleConvChat *
gowhatsapp_enter_group_chat(PurpleConnection *pc, const char *remoteJid) 
{
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleConvChat *conv_chat = purple_conversations_find_chat_with_account(remoteJid, account); // TODO: find out why this fails an assertion in purple_conversation_get_chat_data
    if (conv_chat == NULL) {
        // use hash of jid for chat id number
        PurpleConversation *conv = serv_got_joined_chat(pc, g_str_hash(remoteJid), remoteJid);
        if (conv != NULL) {
            purple_conversation_set_data(conv, "name", g_strdup(remoteJid));
            gowhatsapp_go_query_groups(account); // TODO: query this specific group
        }
        conv_chat = PURPLE_CONV_CHAT(conv);
    }
    return conv_chat;
}

/*
 * Get the identifying aspect of a chat (as passed to serv_got_joined_chat) 
 * given the chat_info entries. In WhatsApp, this is the JID.
 *
 * Borrowed from:
 * https://github.com/matrix-org/purple-matrix/blob/master/libmatrix.c
 */
char *gowhatsapp_get_chat_name(GHashTable *components)
{
    const char *jid = g_hash_table_lookup(components, "name");
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
