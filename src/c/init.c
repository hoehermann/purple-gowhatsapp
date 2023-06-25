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
    GList *types = NULL; // MEMCHECK: caller takes ownership
    PurpleStatusType *status;

    status = purple_status_type_new_full( // MEMCHECK: types takes ownership
        PURPLE_STATUS_AVAILABLE, GOWHATSAPP_STATUS_STR_AVAILABLE, NULL, TRUE, TRUE, FALSE
    );
    types = g_list_append(types, status);

    status = purple_status_type_new_full( // MEMCHECK: types takes ownership
        PURPLE_STATUS_AWAY, GOWHATSAPP_STATUS_STR_AWAY, NULL, TRUE, TRUE, FALSE
    );
    types = g_list_prepend(types, status);

    status = purple_status_type_new_full( // MEMCHECK: types takes ownership
        PURPLE_STATUS_OFFLINE, GOWHATSAPP_STATUS_STR_OFFLINE, NULL, TRUE, TRUE, FALSE
    );
    types = g_list_append(types, status);

    status = purple_status_type_new_full( // MEMCHECK: types takes ownership
        PURPLE_STATUS_MOBILE, GOWHATSAPP_STATUS_STR_MOBILE, NULL, FALSE, FALSE, TRUE
    );
    types = g_list_prepend(types, status);
    
    return types;
}

static void
logout(PurplePluginAction* action) {
  PurpleConnection* pc = action->context;
  PurpleAccount *account = purple_connection_get_account(pc);
  gowhatsapp_go_logout(account);
}

static GList *
actions(PurplePlugin *plugin, gpointer context)
{
    GList* actions = NULL;
    {
        PurplePluginAction *act = purple_plugin_action_new("Logout", &logout);
        actions = g_list_append(actions, act);
    }
    return actions;
}

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

static PurplePluginProtocolInfo prpl_info = {
    .struct_size = sizeof(PurplePluginProtocolInfo), // must be set for PURPLE_PROTOCOL_PLUGIN_HAS_FUNC to work across versions
    .options = OPT_PROTO_NO_PASSWORD, // with this set, Pidgin will neither ask for a password and also won't store it. Yet storing a password is necessary for compatibility with bitlbee. See login.c for more information.
    // TODO: experiment with purple_account_set_remember_password and use password in Pidgin for consistency
    .list_icon = list_icon,
    .status_types = status_types, // this actually needs to exist, else the protocol cannot be set to "online"
    .set_status = gowhatsapp_set_presence,
    .login = gowhatsapp_login,
    .close = gowhatsapp_close,
    .send_im = gowhatsapp_send_im,
    // group-chat related functions
    .chat_info = gowhatsapp_chat_info,
    .chat_info_defaults = gowhatsapp_chat_info_defaults,
    .join_chat = gowhatsapp_join_chat,
    .get_chat_name = gowhatsapp_get_chat_name,
    .find_blist_chat = gowhatsapp_find_blist_chat,
    .chat_send = gowhatsapp_send_chat,
    .set_chat_topic = gowhatsapp_set_chat_topic,
    .roomlist_get_list = gowhatsapp_roomlist_get_list,
    //.roomlist_room_serialize = gowhatsapp_roomlist_serialize, // not necessary – we store the JID in room->name
    // managing buddies (contacts)
    .add_buddy = gowhatsapp_add_buddy,
    .tooltip_text = gowhatsapp_tooltip_text,
    // file transfer
    .new_xfer = gowhatsapp_new_xfer,
    .send_file = gowhatsapp_send_file,
    #if PURPLE_VERSION_CHECK(2,14,0)
    .chat_send_file = gowhatsapp_chat_send_file,
    #endif
};

static void plugin_init(PurplePlugin *plugin) {
    prpl_info.protocol_options = gowhatsapp_add_account_options(prpl_info.protocol_options);
}

static PurplePluginInfo info = {
    .magic = PURPLE_PLUGIN_MAGIC,
    .major_version = PURPLE_MAJOR_VERSION,
    .minor_version = PURPLE_MINOR_VERSION,
    .type = PURPLE_PLUGIN_PROTOCOL,
    .priority = PURPLE_PRIORITY_DEFAULT,
    .id = GOWHATSAPP_PRPL_ID,
    .name = "WhatsApp (whatsmeow)",
    .version = MAKE_STR(PLUGIN_VERSION),
    .summary = "",
    .description = "",
    .author = "Hermann Hoehne <hoehermann@gmx.de>",
    .homepage = "https://github.com/hoehermann/purple-gowhatsapp",
    .load = libpurple2_plugin_load,
    .unload = libpurple2_plugin_unload,
    .extra_info = &prpl_info,
    .actions = actions,
};

PURPLE_INIT_PLUGIN(gowhatsapp, plugin_init, info);
