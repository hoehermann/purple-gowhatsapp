/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2021 Hermann Höhne
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
 
 /*
  * Please note this is the fourth purple plugin I have ever written.
  * I still have no idea what I am doing.
  */

#include "gowhatsapp.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_init

#ifndef PLUGIN_VERSION
#error Must set PLUGIN_VERSION in build system
#endif
// https://github.com/LLNL/lbann/issues/117#issuecomment-334333286
#define MAKE_STR(x) _MAKE_STR(x)
#define _MAKE_STR(x) #x

static const char *
list_icon(PurpleAccount *account, PurpleBuddy *buddy)
{
    return "whatsapp";
}

static GList *
status_types(PurpleAccount *account)
{
    GList *types = NULL;
    PurpleStatusType *status;

    status = purple_status_type_new_full(PURPLE_STATUS_AVAILABLE, GOWHATSAPP_STATUS_STR_AVAILABLE, NULL, TRUE, TRUE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_AWAY, GOWHATSAPP_STATUS_STR_AWAY, NULL, TRUE, TRUE, FALSE);
    types = g_list_prepend(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_OFFLINE, GOWHATSAPP_STATUS_STR_OFFLINE, NULL, TRUE, TRUE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_MOBILE, GOWHATSAPP_STATUS_STR_MOBILE, NULL, FALSE, FALSE, TRUE);
    types = g_list_prepend(types, status);
    
    return types;
}

static GList *
actions(PurplePlugin *plugin, gpointer context)
{
    GList *m = NULL;
    return m;
}

/* Purple 2 Plugin Load Functions */
static gboolean
libpurple2_plugin_load(PurplePlugin *plugin)
{
    return TRUE;
}

static gboolean
libpurple2_plugin_unload(PurplePlugin *plugin)
{
    purple_signals_disconnect_by_handle(plugin);
    return TRUE;
}

static void
plugin_init(PurplePlugin *plugin)
{
    PurplePluginInfo *info;
    PurplePluginProtocolInfo *prpl_info = g_new0(PurplePluginProtocolInfo, 1); // TODO: find out why this leaks memory

    info = plugin->info;

    if (info == NULL) {
        plugin->info = info = g_new0(PurplePluginInfo, 1);
    }
    // base protocol information
    info->name = "WhatsApp (whatsmeow)";
    info->extra_info = prpl_info;
    prpl_info->options = OPT_PROTO_NO_PASSWORD; // with this set, Pidgin will neither ask for a password and also won't store it. Yet storing a password is necessary for compatibility with bitlbee. See login.c for more information.
    prpl_info->protocol_options = gowhatsapp_add_account_options(prpl_info->protocol_options);
    prpl_info->list_icon = list_icon;
    prpl_info->status_types = status_types; // this actually needs to exist, else the protocol cannot be set to "online"
    prpl_info->set_status = gowhatsapp_set_presence;
    prpl_info->login = gowhatsapp_login;
    prpl_info->close = gowhatsapp_close;
    prpl_info->send_im = gowhatsapp_send_im;
    // group-chat related functions
    prpl_info->chat_info = gowhatsapp_chat_info;
    prpl_info->join_chat = gowhatsapp_join_chat;
    prpl_info->get_chat_name = gowhatsapp_get_chat_name;
    prpl_info->find_blist_chat = gowhatsapp_find_blist_chat;
    prpl_info->chat_send = gowhatsapp_send_chat;
    prpl_info->set_chat_topic = gowhatsapp_set_chat_topic;
    prpl_info->roomlist_get_list = gowhatsapp_roomlist_get_list;
    prpl_info->roomlist_room_serialize = gowhatsapp_roomlist_serialize;
    // managing buddies (contacts)
    prpl_info->add_buddy = gowhatsapp_add_buddy;
    prpl_info->tooltip_text = gowhatsapp_tooltip_text;
    // file transfer
    prpl_info->new_xfer = gowhatsapp_new_xfer;
    prpl_info->send_file = gowhatsapp_send_file;
}

static PurplePluginInfo info = {
    PURPLE_PLUGIN_MAGIC,
    PURPLE_MAJOR_VERSION,
    PURPLE_MINOR_VERSION,
    PURPLE_PLUGIN_PROTOCOL, /* type */
    NULL, /* ui_requirement */
    0, /* flags */
    NULL, /* dependencies */
    PURPLE_PRIORITY_DEFAULT, /* priority */
    GOWHATSAPP_PRPL_ID, /* id */
    GOWHATSAPP_NAME, /* name */
    MAKE_STR(PLUGIN_VERSION), /* version */
    "", /* summary */
    "", /* description */
    "Hermann Hoehne <hoehermann@gmx.de>", /* author */
    "https://github.com/hoehermann/libpurple-gowhatsapp", /* homepage */
    libpurple2_plugin_load, /* load */
    libpurple2_plugin_unload, /* unload */
    NULL, /* destroy */
    NULL, /* ui_info */
    NULL, /* extra_info */
    NULL, /* prefs_info */
    actions, /* actions */
    NULL, /* padding */
    NULL,
    NULL,
    NULL
};

PURPLE_INIT_PLUGIN(gowhatsapp, plugin_init, info);
