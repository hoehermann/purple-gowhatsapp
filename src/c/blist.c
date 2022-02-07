#include "gowhatsapp.h"
#include "constants.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_subscribe_presence

PurpleGroup * gowhatsapp_get_purple_group() {
    PurpleGroup *group = purple_blist_find_group("Whatsapp");
    if (!group) {
        group = purple_group_new("Whatsapp");
        purple_blist_add_group(group, NULL);
    }
    return group;
}

void
gowhatsapp_assume_buddy_online(PurpleAccount *account, PurpleBuddy *buddy)
{
    if (purple_account_get_bool(account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
        purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_AWAY, NULL);
        purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_MOBILE, NULL);
    }
    // TODO: move somewhere else so function names are not misleading
    // this is only here because gowhatsapp_assume_buddy_online is alredy being called in all relevant situations
    // NOTE: subscriptions are valid while unavailable
    // but WhatsApp requires you to be available to receive presence updates
    gowhatsapp_go_subscribe_presence(account, buddy->name);

    if (purple_account_get_bool(account, GOWHATSAPP_GET_ICONS_OPTION, FALSE)) {
        const char *picture_date = purple_blist_node_get_string(&buddy->node, "picture_date");
        gowhatsapp_go_request_profile_picture(account, buddy->name, (char *)picture_date); // cgo does not suport const
    }
}

void
gowhatsapp_assume_all_buddies_online(PurpleAccount *account)
{
    g_return_if_fail(account != NULL);
    GSList *buddies = purple_find_buddies(account, NULL);
    while (buddies != NULL) {
        gowhatsapp_assume_buddy_online(account, buddies->data);
        buddies = g_slist_delete_link(buddies, buddies);
    }
}

/*
 * Ensure buddy in the buddy list.
 * Only has effect if "fetch contacts" is enabled.
 * Updates alias non-destructively.
 */
void gowhatsapp_ensure_buddy_in_blist(
    PurpleAccount *account, char *remoteJid, char *display_name
) {
    if (!purple_account_get_bool(account, GOWHATSAPP_FETCH_CONTACTS_OPTION, TRUE)) {
        return;
    }

    PurpleBuddy *buddy = purple_blist_find_buddy(account, remoteJid);

    if (!buddy) {
        PurpleGroup *group = gowhatsapp_get_purple_group();
        buddy = purple_buddy_new(account, remoteJid, display_name);
        purple_blist_add_buddy(buddy, NULL, group, NULL);
        gowhatsapp_assume_buddy_online(account, buddy);
    }

    // update name after checking against local alias and persisted name
    const char *local_alias = purple_buddy_get_alias(buddy);
    const char *published_name = purple_blist_node_get_string(&buddy->node, "published_name");
    if (!purple_strequal(local_alias, display_name) && !purple_strequal(published_name, display_name)) {
        serv_got_alias(purple_account_get_connection(account), remoteJid, display_name); // it seems buddy->server_alias is not persisted
        purple_blist_node_set_string(&buddy->node, "published_name", display_name); // explicitly persisting the new name
    }
}

/*
 * This is called after a buddy has been added to the buddy list 
 * (i.e. by manual user interaction).
 */
void
gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group)
{
    PurpleAccount *account = purple_connection_get_account(pc);
    gowhatsapp_assume_buddy_online(account, buddy);
}

// Group chat related functions

/*
 * Add group chat to blist. Updates existing group chat if found. 
 * Only changes blist if fetch contacts is set.
 */
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(
    PurpleAccount *account, const char *remoteJid, const char *topic
) {
    gboolean fetch_contacts = purple_account_get_bool(
        account, GOWHATSAPP_FETCH_CONTACTS_OPTION, TRUE
    );

    PurpleChat *chat = purple_blist_find_chat(account, remoteJid);

    if (chat == NULL && fetch_contacts) {
        GHashTable *comp = g_hash_table_new_full(g_str_hash, g_str_equal, NULL, g_free);
        g_hash_table_insert(comp, "name", g_strdup(remoteJid));
        chat = purple_chat_new(account, remoteJid, comp);
        PurpleGroup *group = gowhatsapp_get_purple_group();
        purple_blist_add_chat(chat, group, NULL);
    }

    if (topic != NULL && fetch_contacts) {
        // components uses free on key (unlike above)
        purple_blist_alias_chat(chat, topic);
    }

    return chat;
}

/*
 * Find group chat in blist.
 * 
 * This reimplements the default behaviour of purple_blist_find_chat 
 * in libpurple/blist.c and could be removed from here.
 * 
 * Largely borrowed from:
 * https://github.com/EionRobb/purple-discord/blob/master/libdiscord.c
 */
PurpleChat * 
gowhatsapp_find_blist_chat(PurpleAccount *account, const char *jid) 
{
    PurpleBlistNode *node;

    for (node = purple_blist_get_root();
        node != NULL;
        node = purple_blist_node_next(node, TRUE)) {
        if (PURPLE_IS_CHAT(node)) {
            PurpleChat *chat = PURPLE_CHAT(node);

            if (purple_chat_get_account(chat) != account) {
                continue;
            }

            GHashTable *components = purple_chat_get_components(chat);
            const gchar *chat_jid = g_hash_table_lookup(components, "name");

            if (purple_strequal(chat_jid, jid)) {
                return chat;
            }
        }
    }

    return NULL;
}
