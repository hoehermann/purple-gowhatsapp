/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2016 hermann Höhne
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef __GNUC__
#include <unistd.h>
#endif
#include "purplegwa.h"

#ifdef ENABLE_NLS
// TODO: implement localisation
#else
#      define _(a) (a)
#      define N_(a) (a)
#endif

#include "purple_compat.h"

#define GOWHATSAPP_PLUGIN_ID "prpl-hehoe-gowhatsapp"
#ifndef GOWHATSAPP_PLUGIN_VERSION
#define GOWHATSAPP_PLUGIN_VERSION "0.0.1"
#endif
#define GOWHATSAPP_PLUGIN_WEBSITE "https://github.com/hoehermann/libpurple-gowhatsapp"

#define GOWHATSAPP_STATUS_STR_ONLINE   "online"
#define GOWHATSAPP_STATUS_STR_OFFLINE  "offline"
#define GOWHATSAPP_STATUS_STR_MOBILE   "mobile"

typedef struct {
    PurpleAccount *account;
    PurpleConnection *pc;
    
    int index;
    guint event_timer;
    GList *used_images;
} GoWhatsappAccount;
typedef struct gowhatsapp_message gowhatsapp_message_t;

static const char *
gowhatsapp_list_icon(PurpleAccount *account, PurpleBuddy *buddy)
{
    return "gowhatsapp";
}

void
gowhatsapp_assume_buddy_online(PurpleAccount *account, PurpleBuddy *buddy)
{
    purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_ONLINE, NULL);
    purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_MOBILE, NULL);
}

void
gowhatsapp_assume_all_buddies_online(GoWhatsappAccount *sa)
{
    GSList *buddies = purple_find_buddies(sa->account, NULL);
    while (buddies != NULL) {
        gowhatsapp_assume_buddy_online(sa->account, buddies->data);
        buddies = g_slist_delete_link(buddies, buddies);
    }
}


// Copied from p2tgl_imgstore_add_with_id, tgp_msg_photo_display, tgp_format_img
void gowhatsapp_image(PurpleConnection *pc, gchar *who, gchar *caption, void *data, size_t len, PurpleMessageFlags flags, time_t time) {
    int id = purple_imgstore_add_with_id(data, len, NULL);
    if (id <= 0) {
        purple_debug_info("gowhatsapp", "Cannot display picture, adding to imgstore failed.");
        return;
    }
    flags |= PURPLE_MESSAGE_IMAGES;
    if (caption == NULL) {
        caption = "";
    }
    char *content = g_strdup_printf("%s<img id=\"%u\">", caption, id);
    purple_serv_got_im(pc, who, content, flags, time);
    g_free(content);
}

gboolean
gowhatsapp_append_message_id_if_not_exists(PurpleAccount *account, char *message_id)
{
    const gchar RECEIVED_MESSAGES_ID_KEY[] = "receivedmessagesids";
    const gchar *received_messages_ids_str = purple_account_get_string(account, RECEIVED_MESSAGES_ID_KEY, "");
    if (strstr(received_messages_ids_str, message_id) != NULL) {
        purple_debug_info(
            "gowhatsapp", "Suppressed message (already received).\n"
        );
        return FALSE;
    } else {
        gchar *sep = received_messages_ids_str[0] == 0 ? "" : ",";
        // TODO: count entries, prune
        gchar *new_received_messages_ids_str = g_strdup_printf("%s%s%s", received_messages_ids_str, sep, message_id);
        //g_free(received_messages_ids_str); // TODO: check if this is necessary
        purple_account_set_string(account, RECEIVED_MESSAGES_ID_KEY, new_received_messages_ids_str);
        purple_debug_info(
            "gowhatsapp", "received_messages_ids_str now contains %s\n",
            purple_account_get_string(account, RECEIVED_MESSAGES_ID_KEY, "")
        );
        return TRUE;
    }
}

// Polling technique copied from https://github.com/EionRobb/pidgin-opensteamworks/blob/master/libsteamworks.cpp .
gboolean
gowhatsapp_eventloop(gpointer userdata)
{
    PurpleConnection *pc = (PurpleConnection *) userdata;
    GoWhatsappAccount *sa = purple_connection_get_protocol_data(pc);

    PurpleMessageFlags flags = PURPLE_MESSAGE_RECV;
    gowhatsapp_message_t empty = {};
    for (
        gowhatsapp_message_t gwamsg = gowhatsapp_go_getMessage(sa->index);
        memcmp(&gwamsg, &empty, sizeof(gowhatsapp_message_t));
        gwamsg = gowhatsapp_go_getMessage(sa->index)
    ) {
        purple_debug_info(
            "gowhatsapp", "Recieved message: %ld %ld %s %s\n",
            gwamsg.type,
            gwamsg.timestamp,
            gwamsg.id,
            gwamsg.remoteJid
        );
        if (!gwamsg.timestamp) {
            gwamsg.timestamp = time(NULL);
        }
        // to show a system message: flags |= PURPLE_MESSAGE_SYSTEM | PURPLE_MESSAGE_NO_LOG
        switch(gwamsg.type) {
            case gowhatsapp_message_type_error:
                purple_connection_error(pc, PURPLE_CONNECTION_ERROR_NETWORK_ERROR, gwamsg.text);
                break;
            default:
                purple_connection_set_state(pc, PURPLE_CONNECTED);
                if (gowhatsapp_append_message_id_if_not_exists(pc->account, gwamsg.id)) {
                    if (gwamsg.blob) {
                        gowhatsapp_image(pc, gwamsg.remoteJid, gwamsg.text, gwamsg.blob, gwamsg.blobsize, flags, gwamsg.timestamp);
                    } else if (gwamsg.text) {
                        purple_serv_got_im(pc, gwamsg.remoteJid, gwamsg.text, flags, gwamsg.timestamp);
                    }
                }
        }
        free(gwamsg.id);
        free(gwamsg.remoteJid);
        free(gwamsg.text);
        free(gwamsg.blob);
    }
    return TRUE;
}

void
gowhatsapp_login(PurpleAccount *account)
{
    PurpleConnection *pc = purple_account_get_connection(account);

    // this protocol does not support anything special right now
    PurpleConnectionFlags pc_flags;
    pc_flags = purple_connection_get_flags(pc);
    pc_flags |= PURPLE_CONNECTION_NO_IMAGES;
    pc_flags |= PURPLE_CONNECTION_NO_FONTSIZE;
    pc_flags |= PURPLE_CONNECTION_NO_NEWLINES;
    pc_flags |= PURPLE_CONNECTION_NO_BGCOLOR;
    purple_connection_set_flags(pc, pc_flags);

    GoWhatsappAccount *sa = g_new0(GoWhatsappAccount, 1);
    purple_connection_set_protocol_data(pc, sa);
    sa->account = account;
    sa->pc = pc;
    sa->index = gowhatsapp_go_login();
    sa->event_timer = purple_timeout_add_seconds(1, (GSourceFunc)gowhatsapp_eventloop, pc);
    purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTING);
}

static void
gowhatsapp_close(PurpleConnection *pc)
{
    purple_debug_info("gowhatsapp", "gowhatsapp_close()\n");
    GoWhatsappAccount *sa = purple_connection_get_protocol_data(pc);
    gowhatsapp_go_close(sa->index);
    purple_connection_set_state(pc, PURPLE_DISCONNECTED);
    purple_timeout_remove(sa->event_timer);
    g_free(sa);
}

static GList *
gowhatsapp_status_types(PurpleAccount *account)
{
    GList *types = NULL;
    PurpleStatusType *status;

    status = purple_status_type_new_full(PURPLE_STATUS_AVAILABLE, GOWHATSAPP_STATUS_STR_ONLINE, _("Online"), TRUE, FALSE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_OFFLINE, GOWHATSAPP_STATUS_STR_OFFLINE, _("Offline"), TRUE, TRUE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_MOBILE, GOWHATSAPP_STATUS_STR_MOBILE, NULL, FALSE, FALSE, TRUE);
    types = g_list_prepend(types, status);

    return types;
}

/*
static int
gowhatsapp_send_im(PurpleConnection *pc,
#if PURPLE_VERSION_CHECK(3, 0, 0)
                PurpleMessage *msg)
{
    const gchar *who = purple_message_get_recipient(msg);
    const gchar *message = purple_message_get_contents(msg);
#else
                const gchar *who, const gchar *message, PurpleMessageFlags flags)
{
#endif
    //GoWhatsappAccount *sa = purple_connection_get_protocol_data(pc);
    return 1;
}
*/

/*
static void
gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group
#if PURPLE_VERSION_CHECK(3, 0, 0)
                  ,
                  const char *message
#endif
                  )
{
    // does not actually do anything. buddy is added to pidgin's local list and is usable from there.
    GoWhatsappAccount *sa = purple_connection_get_protocol_data(pc);
    if (purple_account_get_bool(sa->account, "fake-online", TRUE)) {
        gowhatsapp_assume_buddy_online(sa->account, buddy);
    }
}
*/

static GList *
gowhatsapp_add_account_options(GList *account_options)
{
    PurpleAccountOption *option;

    option = purple_account_option_bool_new(
                _("Display all contacts as online"),
                "fake-online",
                TRUE
                );
    account_options = g_list_append(account_options, option);

    return account_options;
}

static GList *
gowhatsapp_actions(
#if !PURPLE_VERSION_CHECK(3, 0, 0)
  PurplePlugin *plugin, gpointer context
#else
  PurpleConnection *pc
#endif
  )
{
    GList *m = NULL;
    return m;
}

static gboolean
plugin_load(PurplePlugin *plugin, GError **error)
{
    return TRUE;
}

static gboolean
plugin_unload(PurplePlugin *plugin, GError **error)
{
    purple_signals_disconnect_by_handle(plugin);
    return TRUE;
}

/* Purple2 Plugin Load Functions */
#if !PURPLE_VERSION_CHECK(3, 0, 0)
static gboolean
libpurple2_plugin_load(PurplePlugin *plugin)
{
    return plugin_load(plugin, NULL);
}

static gboolean
libpurple2_plugin_unload(PurplePlugin *plugin)
{
    return plugin_unload(plugin, NULL);
}

static void
plugin_init(PurplePlugin *plugin)
{
    PurplePluginInfo *info;
    PurplePluginProtocolInfo *prpl_info = g_new0(PurplePluginProtocolInfo, 1);

    info = plugin->info;

    if (info == NULL) {
        plugin->info = info = g_new0(PurplePluginInfo, 1);
    }

    info->name = "Whatsapp (HTTP)";
    info->extra_info = prpl_info;
#if PURPLE_MINOR_VERSION >= 5
    //
#endif
#if PURPLE_MINOR_VERSION >= 8
    //
#endif

    prpl_info->options = OPT_PROTO_NO_PASSWORD; // OPT_PROTO_IM_IMAGE
    prpl_info->protocol_options = gowhatsapp_add_account_options(prpl_info->protocol_options);

    /*
    prpl_info->get_account_text_table = discord_get_account_text_table;
    prpl_info->list_emblem = discord_list_emblem;
    prpl_info->status_text = discord_status_text;
    prpl_info->tooltip_text = discord_tooltip_text;
    */
    prpl_info->list_icon = gowhatsapp_list_icon;
    /*
    prpl_info->set_status = discord_set_status;
    prpl_info->set_idle = discord_set_idle;
    */
    prpl_info->status_types = gowhatsapp_status_types; // this actually needs to exist, else the protocol cannot be set to "online"
    /*
    prpl_info->chat_info = discord_chat_info;
    prpl_info->chat_info_defaults = discord_chat_info_defaults;
    */
    prpl_info->login = gowhatsapp_login;
    prpl_info->close = gowhatsapp_close;
    //prpl_info->send_im = gowhatsapp_send_im;
    /*
    prpl_info->send_typing = discord_send_typing;
    prpl_info->join_chat = discord_join_chat;
    prpl_info->get_chat_name = discord_get_chat_name;
    prpl_info->find_blist_chat = discord_find_chat;
    prpl_info->chat_invite = discord_chat_invite;
    prpl_info->chat_send = discord_chat_send;
    prpl_info->set_chat_topic = discord_chat_set_topic;
    prpl_info->get_cb_real_name = discord_get_real_name;
    */
    //prpl_info->add_buddy = gowhatsapp_add_buddy;
    /*
    prpl_info->remove_buddy = discord_buddy_remove;
    prpl_info->group_buddy = discord_fake_group_buddy;
    prpl_info->rename_group = discord_fake_group_rename;
    prpl_info->get_info = discord_get_info;
    prpl_info->add_deny = discord_block_user;
    prpl_info->rem_deny = discord_unblock_user;

    prpl_info->roomlist_get_list = discord_roomlist_get_list;
    prpl_info->roomlist_room_serialize = discord_roomlist_serialize;
    */
}

static PurplePluginInfo info = {
    PURPLE_PLUGIN_MAGIC,
    /*    PURPLE_MAJOR_VERSION,
        PURPLE_MINOR_VERSION,
    */
    2, 1,
    PURPLE_PLUGIN_PROTOCOL,            /* type */
    NULL,                            /* ui_requirement */
    0,                                /* flags */
    NULL,                            /* dependencies */
    PURPLE_PRIORITY_DEFAULT,        /* priority */
    GOWHATSAPP_PLUGIN_ID,                /* id */
    "gowhatsapp",                        /* name */
    GOWHATSAPP_PLUGIN_VERSION,            /* version */
    "",                                /* summary */
    "",                                /* description */
    "Hermann Hoehne <hoehermann@gmx.de>", /* author */
    GOWHATSAPP_PLUGIN_WEBSITE,            /* homepage */
    libpurple2_plugin_load,            /* load */
    libpurple2_plugin_unload,        /* unload */
    NULL,                            /* destroy */
    NULL,                            /* ui_info */
    NULL,                            /* extra_info */
    NULL,                            /* prefs_info */
    gowhatsapp_actions,                /* actions */
    NULL,                            /* padding */
    NULL,
    NULL,
    NULL
};

PURPLE_INIT_PLUGIN(gowhatsapp, plugin_init, info);

#else
/* Purple 3 plugin load functions */
#perror Purple 3 not supported.
#endif
