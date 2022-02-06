#include "gowhatsapp.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_get_joined_groups

/*
 * I do not actually know what this does.
 */
void gowhatsapp_join_chat(PurpleConnection *pc, GHashTable *data) {
    const char *remoteJid = g_hash_table_lookup(data, "remoteJid");
    if (remoteJid != NULL) {
        gowhatsapp_enter_group_chat(pc, remoteJid);
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

PurpleConvChat *
gowhatsapp_enter_group_chat(PurpleConnection *pc, const char *remoteJid) 
{
    PurpleAccount *account = purple_connection_get_account(pc);
    // TODO: call gowhatsapp_ensure_group_chat_in_blist somewhere
    PurpleConvChat *conv_chat = purple_conversations_find_chat_with_account(remoteJid, account);
    if (conv_chat == NULL) {
        // use hash of jid for chat id number
        PurpleConversation *conv = serv_got_joined_chat(pc, g_str_hash(remoteJid), remoteJid);
        if (conv != NULL) {
            purple_conversation_set_data(conv, "remoteJid", g_strdup(remoteJid));
        }
        conv_chat = PURPLE_CONV_CHAT(conv);
    }
    return conv_chat;
}

/*
 * This represents all group chats as a list of rooms.
 * Each room also has two fields:
 * 0. remoteJid (WhatsApp identifier)
 * 1. topic (group name)
 * Order is important, see gowhatsapp_roomlist_serialize.
 */
PurpleRoomlist *
gowhatsapp_roomlist_get_list(PurpleConnection *pc) {
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleRoomlist *roomlist = purple_roomlist_new(account);
    purple_roomlist_set_in_progress(roomlist, TRUE);
    
    GList *fields = NULL;
    // order is important (see description)
    fields = g_list_append(fields, purple_roomlist_field_new(
        PURPLE_ROOMLIST_FIELD_STRING, "JID", "remoteJid", FALSE
    ));
    /*
     * BEWARE: 
     * In Pidgin, the roomlist field "name" might actually denote the group chat identifier.
     * It gets overwritten in purple_roomlist_room_join, see libpurple/roomlist.c.
     * Also, some services like spectrum expect the human readable group name to be "topic", 
     * see RoomlistProgress in https://github.com/SpectrumIM/spectrum2/blob/518ba5a/backends/libpurple/main.cpp#L1997
     */ 
    fields = g_list_append(fields, purple_roomlist_field_new(
        PURPLE_ROOMLIST_FIELD_STRING, "Group Name", "topic", FALSE
    ));
    purple_roomlist_set_fields(roomlist, fields);
    
    gowhatsapp_group_info_t *group_infos = gowhatsapp_go_get_joined_groups(account);
    for (gowhatsapp_group_info_t *group_info = group_infos; group_info->remoteJid != NULL; group_info++) {
        //purple_debug_info(GOWHATSAPP_NAME, "group_info: remoteJid: %s, name: %s, topic: %s", group_info->remoteJid, group_info->name, group_info->topic);
        PurpleRoomlistRoom *room = purple_roomlist_room_new(PURPLE_ROOMLIST_ROOMTYPE_ROOM, group_info->remoteJid, NULL);

        // order is important (see description)
        purple_roomlist_room_add_field(roomlist, room, group_info->remoteJid);
        purple_roomlist_room_add_field(roomlist, room, group_info->name);
        purple_roomlist_room_add_field(roomlist, room, group_info->topic);

        purple_roomlist_room_add(roomlist, room);
        
        // purple_roomlist_room_add_field does a strdup, free strings here
        g_free(group_info->remoteJid);
        g_free(group_info->name);
        g_free(group_info->topic);
    }
    g_free(group_infos);

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
GList * gowhatsapp_chat_info(PurpleConnection *pc)
{
    GList *infos = NULL;

    struct proto_chat_entry *pce;

    pce = g_new0(struct proto_chat_entry, 1);
    pce->label = "JID";
    pce->identifier = "remoteJid";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    pce = g_new0(struct proto_chat_entry, 1);
    pce->label = "Name";
    pce->identifier = "name";
    pce->required = TRUE;
    infos = g_list_append(infos, pce);

    return infos;
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
