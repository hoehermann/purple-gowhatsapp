#include "gowhatsapp.h"
#include "constants.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_send_presence

/*
 * WhatsApp informed us that a contact is now online or offline.
 * Time last seen is provided optionally.
 */
void
gowhatsapp_handle_presence(PurpleAccount *account, char *remoteJid, char online, time_t last_seen)
{
    char *status = GOWHATSAPP_STATUS_STR_AVAILABLE;
    if (online == 0) {
        if (purple_account_get_bool(account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
            status = GOWHATSAPP_STATUS_STR_AWAY;
        } else {
            status = GOWHATSAPP_STATUS_STR_OFFLINE;
        }
    }
    purple_prpl_got_user_status(account, remoteJid, status, NULL);
    
    if (last_seen != 0) {
        PurpleBuddy *buddy = purple_blist_find_buddy(account, remoteJid);
        if (buddy){
            // TODO: narrowing time_t to int – year 2038 problem incoming
            // last_seen is a well known int in purple – cannot convert to string here
            purple_blist_node_set_int(&buddy->node, "last_seen", last_seen);
        }
    }
}

/*
 * A profile picture has been downloaded.
 * Set icon and store date for caching.
 */
void
gowhatsapp_handle_profile_picture(gowhatsapp_message_t *gwamsg)
{
    purple_buddy_icons_set_for_user(gwamsg->account, gwamsg->remoteJid, gwamsg->blob, gwamsg->blobsize, NULL);
    PurpleBuddy *buddy = purple_blist_find_buddy(gwamsg->account, gwamsg->remoteJid);
    // no g_free(gwamsg->blob) here – purple takes ownership
    purple_blist_node_set_string(&buddy->node, "picture_id", gwamsg->senderJid);
    purple_blist_node_set_string(&buddy->node, "picture_date", gwamsg->text);
    // TODO: use purple_buddy_icons_set_account_icon_timestamp instead of saving the time string
}

void 
gowhatsapp_tooltip_text(PurpleBuddy *buddy, PurpleNotifyUserInfo *info, gboolean full){
    int last_seen = purple_blist_node_get_int(&buddy->node, "last_seen");
    if (last_seen != 0) {
        time_t t = last_seen;
        const size_t bufsize = 100;
        char buf[bufsize];
        strftime(buf, bufsize, "%c", gmtime(&t));
        purple_notify_user_info_add_pair(info, "Last seen", buf);
    }
    const char *picture_id = purple_blist_node_get_string(&buddy->node, "picture_id");
    if (picture_id != NULL) {
        purple_notify_user_info_add_pair(info, "Picture ID", picture_id);
    }
    const char *picture_date = purple_blist_node_get_string(&buddy->node, "picture_date");
    if (picture_date != NULL) {
        purple_notify_user_info_add_pair(info, "Picture date", picture_date);
    }
    const char *server_alias = purple_blist_node_get_string(&buddy->node, "server_alias");
    if (server_alias != NULL) {
        purple_notify_user_info_add_pair(info, "Pushname", server_alias);
    }
}

/*
 * Subscribe for presence updates for all contacts currently in the buddy list.
 * 
 * NOTE: Receiving contact presence updates may only be sent by WhatsApp when being "available".
 */
void
gowhatsapp_subscribe_all_presence_updates(PurpleAccount *account)
{
    g_return_if_fail(account != NULL);
    GSList *buddies = purple_find_buddies(account, NULL);
    while (buddies != NULL) {
        gowhatsapp_subscribe_presence_updates(account, buddies->data);
        buddies = g_slist_delete_link(buddies, buddies);
    }
}

/*
 * Set own presence.
 * 
 * NOTE: Remote contact's presence updates may only be sent by WhatsApp when being "available" oneself.
 */
void
gowhatsapp_set_presence(PurpleAccount *account, PurpleStatus *status) {
    const char *status_id = purple_status_get_id(status);
    
    const char *presence_override = purple_account_get_string(account, GOWHATSAPP_PRESENCE_OVERRIDE_KEY, NULL);
    if (presence_override != NULL) {
        status_id = presence_override;
    }

    gowhatsapp_go_send_presence(account, (char *)status_id); // cgo does not support const
    // (re-)subscribe for presence updates (will check status_id again)
    gowhatsapp_subscribe_all_presence_updates(account);
}

/*
 * Subscribe for presence updates for a specific buddy.
 * 
 * NOTE: Remote contact's presence updates may only be sent by WhatsApp when being "available" oneself.
 */
void
gowhatsapp_subscribe_presence_updates(PurpleAccount *account, PurpleBuddy *buddy)
{
    const PurpleStatus *status = purple_account_get_active_status(account);
    const char *status_id = purple_status_get_id(status);
    
    const char *presence_override = purple_account_get_string(account, GOWHATSAPP_PRESENCE_OVERRIDE_KEY, NULL);
    if (presence_override != NULL) {
        status_id = presence_override;
    }
    
    if (purple_strequal(status_id, GOWHATSAPP_STATUS_STR_AVAILABLE)) {
        // NOTE: WhatsApp requires you to be available to receive presence updates
        // subscribing for presence updates might implicitly set own presence to available
        gowhatsapp_go_subscribe_presence(account, buddy->name);
    }
}
