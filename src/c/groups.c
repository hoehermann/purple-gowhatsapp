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

    pce = g_new0(struct proto_chat_entry, 1); // MEMCHECK: infos takes ownership
    pce->label = "JID";
    pce->identifier = "name";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    pce = g_new0(struct proto_chat_entry, 1); // MEMCHECK: infos takes ownership
    pce->label = "Group Name";
    pce->identifier = "topic";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    return infos; // MEMCHECK: caller takes ownership
}

/*
 * Takes a purple chat_name and prepares all information necessary to join the chat with that name.
 * 
 * Ideally, this would take the human readable WhatsApp group name and look up the appropriate JID.
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
    GHashTable *defaults = g_hash_table_new_full( // MEMCHECK: caller takes ownership
        g_str_hash, g_str_equal, NULL, g_free
    );
    if (chat_name != NULL) {
        g_hash_table_insert(defaults, "name", g_strdup(chat_name)); // MEMCHECK: g_strdup'ed string released by GHashTable's value_destroy_func g_free (see above)
        g_hash_table_insert(defaults, "topic", g_strdup("")); // MEMCHECK: g_strdup'ed string released by GHashTable's value_destroy_func g_free (see above)
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
        PurpleConvChat *chat = gowhatsapp_enter_group_chat(pc, remoteJid, NULL);
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
    WhatsappProtocolData *wpd = (WhatsappProtocolData *)purple_connection_get_protocol_data(pc);
    g_return_val_if_fail(wpd != NULL, NULL);
    PurpleRoomlist *roomlist = wpd->roomlist;
    if (roomlist != NULL) {
        purple_debug_info(GOWHATSAPP_NAME, "Already getting roomlist.");
        return roomlist;
    }
    roomlist = purple_roomlist_new(account); // MEMCHECK: caller takes ownership
    purple_roomlist_set_in_progress(roomlist, TRUE);
    GList *fields = NULL;
    fields = g_list_append(fields, purple_roomlist_field_new( // MEMCHECK: fields takes ownership
        PURPLE_ROOMLIST_FIELD_STRING, "Group Name", "topic", FALSE
    ));
    purple_roomlist_set_fields(roomlist, fields);
    wpd->roomlist = roomlist;
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
gowhatsapp_roomlist_add_room(PurpleConnection *pc, char *remoteJid, char *name) {
    WhatsappProtocolData *wpd = (WhatsappProtocolData *)purple_connection_get_protocol_data(pc);
    g_return_if_fail(wpd != NULL);
    PurpleRoomlist *roomlist = wpd->roomlist;
    if (roomlist != NULL) {
        if (remoteJid == NULL) {
            // list end marker, room-listing is finished and ready
            purple_roomlist_set_in_progress(roomlist, FALSE);
            purple_roomlist_unref(roomlist); // unref here, roomlist may remain in ui
            wpd->roomlist = NULL;
        } else {
            PurpleRoomlistRoom *room = purple_roomlist_room_new(PURPLE_ROOMLIST_ROOMTYPE_ROOM, remoteJid, NULL); // MEMCHECK: roomlist takes ownership 
            // purple_roomlist_room_new sets the room's name
            purple_roomlist_room_add_field(roomlist, room, name); // this sets the room's title
            purple_roomlist_room_add(roomlist, room);
        }
    }
}

/*
 * Handle incoming group information.
 * 
 * Group information is requested asynchronously when building the roomlist.
 * 
 * NOTE: The roomlist is requested automatically when the local user status is set to "available".
 */
void 
gowhatsapp_handle_group(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    // list the group in the roomlist (if it is currently being queried)
    gowhatsapp_roomlist_add_room(pc, gwamsg->remoteJid, gwamsg->name);
    // these all cannot handle the group list end marker
    if (gwamsg->remoteJid != NULL) {
        // adds the group to the buddy list (if fetching contacts is enabled, useful for human-readable titles)
        gowhatsapp_ensure_group_chat_in_blist(gwamsg->account, gwamsg->remoteJid, gwamsg->name);
        // this might be a delayed response to a query for participants of a currently active group chat
        PurpleConvChat *conv_chat = purple_conversations_find_chat_with_account(gwamsg->remoteJid, gwamsg->account);
        if (conv_chat != NULL) {
            gowhatsapp_chat_set_participants(conv_chat, gwamsg->participants);
        }
        // automatically join all chats (if user wants to)
        if (purple_account_get_bool(gwamsg->account, GOWHATSAPP_AUTO_JOIN_CHAT_OPTION, FALSE)) {
            gowhatsapp_enter_group_chat(pc, gwamsg->remoteJid, gwamsg->participants); 
        }
    }
}

// Helper functions regarding the purple conversation representing a WhatsApp group

/*
 * Adds participants to chat.
 */
void 
gowhatsapp_chat_set_participants(PurpleConvChat *conv_chat, char **participants) {
    // remove all users
    // TODO: gracefully remove participants who are in the purple chat, but not in the array of participants
    purple_conv_chat_clear_users(conv_chat);
    // now add all current users
    for(char **participant_ptr = participants; participant_ptr != NULL && *participant_ptr != NULL; participant_ptr++) {
        if (!gowhatsapp_user_in_conv_chat(conv_chat, *participant_ptr)) {
            PurpleConvChatBuddyFlags flags = 0;
            purple_conv_chat_add_user(conv_chat, *participant_ptr, NULL, flags, FALSE);
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
gowhatsapp_enter_group_chat(PurpleConnection *pc, const char *remoteJid, char **participants) 
{
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleConvChat *conv_chat = purple_conversations_find_chat_with_account(remoteJid, account);
    if (conv_chat == NULL) {
        // use hash of jid for chat id number
        PurpleConversation *conv = serv_got_joined_chat(pc, g_str_hash(remoteJid), remoteJid);
        if (conv != NULL) {
            // store the JID so it can be retrieved by get_chat_name
            purple_conversation_set_data(conv, "name", g_strdup(remoteJid)); // MEMCHECK: strdup'ed value eventually released by gowhatsapp_free_name
            conv_chat = purple_conversation_get_chat_data(conv);
            if (participants == NULL) {
                // list of participants is empty, request it explicitly and release it immediately
                char **participants = gowhatsapp_go_query_group_participants(account, (char *)remoteJid);
                gowhatsapp_chat_set_participants(conv_chat, participants);
                g_strfreev(participants);
            } else {
                // list of participants was given by caller, add particpants have caller release it later
                gowhatsapp_chat_set_participants(conv_chat, participants);
            }
        }
    }
    return conv_chat;
}

/*
 * purple_conversation_destroy does not implicitly release the data members.
 * We need to do it explicitly.
 */
void gowhatsapp_free_name(PurpleConversation *conv) {
    // TODO: find out why this is never called
    g_free(purple_conversation_get_data(conv, "name"));
    purple_conversation_set_data(conv, "name", NULL);
}

/*
 * Get the identifying aspect of a chat (as passed to serv_got_joined_chat) 
 * given the chat_info entries. In WhatsApp, this is the JID.
 * 
 * The JID is stored in the "name" field. In libpurple, this is the default.
 * It is reimplementerd here explicitly since it might be good for compatibility 
 * with bitlbee or spectrum.
 *
 * Borrowed from:
 * https://github.com/matrix-org/purple-matrix/blob/master/libmatrix.c
 */
char *gowhatsapp_get_chat_name(GHashTable *components)
{
    const char *jid = g_hash_table_lookup(components, "name");
    return g_strdup(jid); // MEMCHECK: strdup'ed value is released by caller
}

/*
 * Determines if user is participating in conversation.
 */
int gowhatsapp_user_in_conv_chat(PurpleConvChat *conv_chat, const char *userJid) {
    for (GList *users = purple_conv_chat_get_users(conv_chat); users != NULL; 
        users = users->next) {
        PurpleConvChatBuddy *buddy = (PurpleConvChatBuddy *) users->data;
        if (!strcmp(buddy->name, userJid)) {
            return TRUE;
        }
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
