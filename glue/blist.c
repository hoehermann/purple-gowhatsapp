#include "gowhatsapp.h"
#include "constants.h"
#include "libwhatsmeow.h" // for gowhatsapp_go_subscribe_presence

PurpleGroup * gowhatsapp_get_purple_group() {
    PurpleGroup *group = purple_blist_find_group("Whatsapp");
    if (!group) {
        group = purple_group_new("Whatsapp"); // MEMCHECK: caller takes ownership
        purple_blist_add_group(group, NULL);
    }
    return group;
}

void gowhatsapp_assume_buddy_away(PurpleAccount *account, PurpleBuddy *buddy) {
    g_return_if_fail(buddy != NULL);

    if (purple_account_get_bool(account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
        purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_AWAY, NULL);
        purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_MOBILE, NULL);
    }
}

/*
 * Ensure buddy in the buddy list.
 * Updates alias non-destructively.
 * 
 * identifier is the username (purple who)
 * name is the human readable name (purple alias).
 */
PurpleBuddy * gowhatsapp_ensure_buddy_in_blist(PurpleAccount *account, const char *identifier, const char *name) {
    if (purple_str_has_suffix(identifier, "@lid")) {
        // TODO: combine into existing non-hidden buddy
        return NULL;
    }

    PurpleBuddy *buddy = purple_blist_find_buddy(account, identifier);

    if (!buddy) {
        PurpleGroup *group = gowhatsapp_get_purple_group();
        buddy = purple_buddy_new(account, identifier, name); // MEMCHECK: blist takes ownership
        purple_blist_add_buddy(buddy, NULL, group, NULL);
        gowhatsapp_subscribe_presence_updates(account, buddy);
    }

    // update name after checking against local alias and persisted name
    if (name != NULL && *name) {
        const char *local_alias = purple_buddy_get_alias(buddy);
        const char *server_alias = purple_blist_node_get_string(&buddy->node, "server_alias");
        if (local_alias == NULL) {
            // if no local alias exists, use the provided one
            purple_blist_alias_buddy(buddy, name);
        }
        if (!purple_strequal(local_alias, name) && !purple_strequal(server_alias, name)) {
            purple_serv_got_alias(purple_account_get_connection(account), identifier, name); // this sets buddy->server_alias, but it is not persisted
            purple_blist_node_set_string(&buddy->node, "server_alias", name); // explicitly persist the new name so there is no name-change reported after a restart
        }
    }

    return buddy;
}

/*
 * This is called after a buddy has been added to the buddy list 
 * (i.e. by manual user interaction).
 */
void gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group) {
    PurpleAccount *account = purple_connection_get_account(pc);
    gowhatsapp_assume_buddy_away(account, buddy);
    gowhatsapp_subscribe_presence_updates(account, buddy);
}

/*
 * Calls a function once on each buddy.
 */
void gowhatsapp_for_all_buddies(PurpleAccount *account, void(*func)(PurpleAccount *, PurpleBuddy *)) {
    g_return_if_fail(account != NULL);
    GSList *buddies = purple_find_buddies(account, NULL);
    while (buddies != NULL) {
        func(account, buddies->data);
        buddies = g_slist_delete_link(buddies, buddies);
    }
}

// Group chat related functions

/*
 * Add group chat to blist. Updates existing group chat if found. 
 * Only changes blist if fetch contacts is set.
 */
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(PurpleAccount *account, const char *remoteJid, const char *topic) {
    PurpleChat *chat = purple_blist_find_chat(account, remoteJid);

    if (chat == NULL) {
        GHashTable *comp = g_hash_table_new_full(g_str_hash, g_str_equal, NULL, g_free); // MEMCHECK: purple_chat_new takes ownership
        g_hash_table_insert(comp, "name", g_strdup(remoteJid)); // MEMCHECK: g_strdup'ed string released by GHashTable's value_destroy_func g_free (see above)
        chat = purple_blist_chat_new(account, remoteJid, comp); // MEMCHECK: blist takes ownership // TODO: double check if the middle parameter should be the topic instead
        PurpleGroup *group = gowhatsapp_get_purple_group();
        purple_blist_add_chat(chat, group, NULL);
    }

    if (topic != NULL) {
        purple_blist_alias_chat(chat, topic);
    }

    return chat;
}

/*
 * Find group chat in blist.
 * 
 * This reimplements the default behaviour of purple_blist_find_chat 
 * in libpurple/blist.c and could be removed from here.
 * Difference: purple_blist_find_chat returns NULL when account is not connected.
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

/*
 * Returns the alias of the contact or NULL.
 *
 * The alias is owned by the blist and must not be released.
 */
const char * gowhatsapp_blist_get_alias(PurpleAccount *account, const char *who) {
    PurpleBuddy *buddy = purple_blist_find_buddy(account, who);
    if (buddy == NULL) {
        return NULL;
    } else {
        return purple_buddy_get_alias(buddy);
    }
}