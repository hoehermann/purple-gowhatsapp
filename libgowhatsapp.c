/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2019 Hermann Höhne
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
  * Please note this is the third purple plugin I have ever written.
  * I still have no idea what I am doing.
  */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef __GNUC__
#include <unistd.h>
#endif
#include <inttypes.h>
#include "purplegwa.h"

#ifdef ENABLE_NLS
// TODO: implement localisation
#include <glib/gi18n.h>
#else
#      define _(a) (a)
#      define N_(a) (a)
#endif

#include "purple_compat.h"

#define GOWHATSAPP_PLUGIN_ID "prpl-hehoe-gowhatsapp"
#ifndef GOWHATSAPP_PLUGIN_VERSION
#error Must set GOWHATSAPP_PLUGIN_VERSION in Makefile
#endif
#define GOWHATSAPP_PLUGIN_WEBSITE "https://github.com/hoehermann/libpurple-gowhatsapp"

#define GOWHATSAPP_STATUS_STR_ONLINE   "online"
#define GOWHATSAPP_STATUS_STR_OFFLINE  "offline"
#define GOWHATSAPP_STATUS_STR_MOBILE   "mobile"

/*
 * Holds all information related to this account (connection) instance.
 */
typedef struct {
    PurpleAccount *account;
    PurpleConnection *pc;

    GList *used_images; // for inline images

    time_t previous_sessions_last_messages_timestamp; // keeping track of last received message's date
} GoWhatsappAccount;
typedef struct gowhatsapp_message gowhatsapp_message_t;

static const char *
gowhatsapp_list_icon(PurpleAccount *account, PurpleBuddy *buddy)
{
    return "whatsapp";
}

void
gowhatsapp_assume_buddy_online(PurpleAccount *account, PurpleBuddy *buddy)
{
    purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_ONLINE, NULL);
    purple_prpl_got_user_status(account, buddy->name, GOWHATSAPP_STATUS_STR_MOBILE, NULL);
}

void
gowhatsapp_assume_all_buddies_online(GoWhatsappAccount *gwa)
{
    GSList *buddies = purple_find_buddies(gwa->account, NULL);
    while (buddies != NULL) {
        gowhatsapp_assume_buddy_online(gwa->account, buddies->data);
        buddies = g_slist_delete_link(buddies, buddies);
    }
}

/*
 * Displays an image in the conversation window.
 * Copied from p2tgl_imgstore_add_with_id, tgp_msg_photo_display, tgp_format_img.
 */
char * gowhatsapp_prepare_image_message(gchar *caption, void *data, size_t len, PurpleMessageFlags *flags) {
    const gpointer data_copy = g_memdup(data, len); // create a copy so freeing is consistent in caller, but imgstore may free the copy it is given
    int id = purple_imgstore_add_with_id(data_copy, len, NULL);
    if (id <= 0) {
        return g_strdup_printf("Cannot display picture with caption \"%s\", as adding to imgstore failed.", caption);
    }
    *flags |= PURPLE_MESSAGE_IMAGES;
    if (caption == NULL) {
        caption = "";
    }
    return g_strdup_printf("%s<img id=\"%u\">", caption, id);
}

/*
 * Checks whether message is old.
 */
gboolean
gowhatsapp_message_newer_than_last_session (GoWhatsappAccount *gwa, const time_t ts) {
    if (ts < gwa->previous_sessions_last_messages_timestamp) {
        return FALSE;
    } else {
        purple_account_set_int(gwa->account, GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, ts); // TODO: find out if this narrowing of time_t aka. long int to int imposes a problem
        return TRUE;
    }
}

/*
 * Builds a persistent list of IDs of messages already received.
 * 
 * @return Message ID is new.
 */
gboolean
gowhatsapp_append_message_id_if_not_exists(PurpleAccount *account, char *message_id)
{
    if (message_id == NULL || message_id[0] == 0) {
        // always display system messages (they have no ID)
        return TRUE;
    }
    static const gchar *RECEIVED_MESSAGES_ID_KEY = "receivedmessagesids";
    const gchar *received_messages_ids_str = purple_account_get_string(account, RECEIVED_MESSAGES_ID_KEY, "");
    if (strstr(received_messages_ids_str, message_id) != NULL) {
        purple_debug_info(
            "gowhatsapp", "Suppressed message (already received).\n"
        );
        return FALSE;
    } else {
        // count ids currently in store
        static const char GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR = ',';
        const unsigned int GOWHATSAPP_MESSAGEIDSTORE_SIZE = purple_account_get_int(account, GOWHATSAPP_MESSAGE_ID_STORE_SIZE_OPTION, 1000);
        unsigned int occurrences = 0;
        char *offset = (char *)received_messages_ids_str + strlen(received_messages_ids_str);
        for (; offset > received_messages_ids_str; offset--) {
            if (*offset == GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR) {
                occurrences++;
                if (occurrences >= GOWHATSAPP_MESSAGEIDSTORE_SIZE) {
                    // store is full, stop counting
                    break;
                }
            }
        }
        // append new ID
        gchar *new_received_messages_ids_str = g_strdup_printf("%s%c%s", offset, GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR, message_id);
        purple_account_set_string(account, RECEIVED_MESSAGES_ID_KEY, new_received_messages_ids_str);
        g_free(new_received_messages_ids_str);
        return TRUE;
    }
}

PurpleConversation *gowhatsapp_find_conversation(char *username, PurpleAccount *account) {
    PurpleIMConversation *imconv = purple_conversations_find_im_with_account(username, account);
    if (imconv == NULL) {
        imconv = purple_im_conversation_new(account, username);
    }
    PurpleConversation *conv = PURPLE_CONVERSATION(imconv);
    if (conv == NULL) {
        imconv = purple_conversations_find_im_with_account(username, account);
        conv = PURPLE_CONVERSATION(imconv);
    }
    return conv;
}

void
gowhatsapp_display_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg)
{
    GoWhatsappAccount *gwa = purple_connection_get_protocol_data(pc);
    PurpleMessageFlags flags = 0;
    if (gwamsg->system && !purple_account_get_bool(gwa->account, GOWHATSAPP_SYSTEM_MESSAGES_ARE_ORDINARY_MESSAGES_OPTION, FALSE)) {
        // spectrum2 swallows system messages – that is why these flags can be suppressed
        flags |= PURPLE_MESSAGE_SYSTEM | PURPLE_MESSAGE_NO_LOG;
    }
    gboolean check_message_date = purple_account_get_bool(gwa->account, GOWHATSAPP_TIMESTAMP_FILTERING_OPTION, FALSE);
    int message_is_new =
            gowhatsapp_append_message_id_if_not_exists(gwa->account, gwamsg->id) &&
            (!check_message_date || gowhatsapp_message_newer_than_last_session(gwa, gwamsg->timestamp));
    if (message_is_new) {
        char * content = NULL;
        if (gwamsg->blobsize) {
            content = gowhatsapp_prepare_image_message(gwamsg->text, gwamsg->blob, gwamsg->blobsize, &flags);
        } else if (gwamsg->text) {
            content = g_strdup(gwamsg->text);
        }
        if (gwamsg->fromMe) {
            // special handling of messages sent by self incoming from remote, addressing issue #32
            // copied from EionRobb/purple-discord/blob/master/libdiscord.c
            flags |= PURPLE_MESSAGE_SEND | PURPLE_MESSAGE_REMOTE_SEND | PURPLE_MESSAGE_DELAYED;
            PurpleConversation *conv = gowhatsapp_find_conversation(gwamsg->remoteJid, gwa->account);
            purple_conversation_write(conv, gwamsg->remoteJid, content, flags, gwamsg->timestamp);
        } else {
            flags |= PURPLE_MESSAGE_RECV;
            purple_serv_got_im(pc, gwamsg->remoteJid, content, flags, gwamsg->timestamp);
        }
        g_free(content);
    }
}

static void
gowhatsapp_null_cb() {
}

static void
gowhatsapp_display_qrcode(PurpleConnection *pc, const char * qr_code_raw, void * image_data, size_t image_data_len)
{
    GoWhatsappAccount *gwa = purple_connection_get_protocol_data(pc);

    PurpleRequestFields *fields = purple_request_fields_new();
    PurpleRequestFieldGroup *group = purple_request_field_group_new(NULL);
    purple_request_fields_add_group(fields, group);

    PurpleRequestField *string_field = purple_request_field_string_new("qr_string", _("QR Code Data"), g_strdup(qr_code_raw), FALSE);
    purple_request_field_group_add_field(group, string_field);
    PurpleRequestField *image_field = purple_request_field_image_new("qr_image", _("QR Code Image"), image_data, image_data_len);
    purple_request_field_group_add_field(group, image_field);

    const char *username = g_strdup(purple_account_get_username(gwa->account));
    const char *secondary = g_strdup_printf(_("WhatsApp account %s"), username);

    purple_request_fields(
        gwa->pc, /*handle*/
        _("Logon QR Code"), /*title*/
        _("Please scan this QR code with your phone"), /*primary*/
        secondary, /*secondary*/
        fields, /*fields*/
        _("OK"), G_CALLBACK(gowhatsapp_null_cb), /*OK*/
        _("Dismiss"), G_CALLBACK(gowhatsapp_null_cb), /*Cancel*/
        NULL, /*account*/
        username, /*username*/
        NULL, /*conversation*/
        NULL /*data*/
    );
}

/*
 * Interprets a message received from go-whatsapp. Handles login success and failure. Forwards errors.
 */
void
gowhatsapp_process_message(gowhatsapp_message_t *gwamsg)
{
    purple_debug_info(
        "gowhatsapp", "%p recieved message type %" PRId64 " at %ld id %s remote %s sender %s (fromMe %d, system %d): %s\n",
        (void *)gwamsg->connection,
        gwamsg->msgtype,
        gwamsg->timestamp,
        gwamsg->id,
        gwamsg->remoteJid,
        gwamsg->senderJid,
        gwamsg->fromMe,
        gwamsg->system,
        gwamsg->text
    );

    PurpleConnection *pc = (PurpleConnection *)gwamsg->connection;
    GoWhatsappAccount *gwa = purple_connection_get_protocol_data(pc);
    PurpleAccount *account = purple_connection_get_account(pc);

    if (!gwamsg->timestamp) {
        gwamsg->timestamp = time(NULL);
    }
    switch(gwamsg->msgtype) {
        case gowhatsapp_message_type_error:
            // purplegwa presents an error, handle it
            if (strstr(gwamsg->text, "401") || strstr(gwamsg->text, "419")) {
                // received error mentioning 401 – assume login failed and try again without stored session
                // TODO: find a better way to discriminate errors
                // "restoring failed: admin login responded with 419"
                purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTING);
                purple_connection_update_progress(gwa->pc, _("Connecting"), 0, 3);
                char *download_directory = g_strdup_printf("%s/gowhatsapp", purple_user_dir());
                gowhatsapp_go_login((uintptr_t)pc, FALSE, download_directory);
                g_free(download_directory);
                // alternatively just display purple_connection_error(pc, PURPLE_CONNECTION_ERROR_AUTHENTICATION_FAILED, gwamsg->text); and let the user handle the session reset
            } else {
                purple_connection_error(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, gwamsg->text);
            }
            break;
        
        case gowhatsapp_message_type_session:
            // purplegwa presents session data, store it
            purple_account_set_string(account, GOWHATSAPP_SESSION_CLIENDID_KEY, gwamsg->clientId);
            purple_account_set_string(account, GOWHATSAPP_SESSION_CLIENTTOKEN_KEY, gwamsg->clientToken);
            purple_account_set_string(account, GOWHATSAPP_SESSION_SERVERTOKEN_KEY, gwamsg->serverToken);
            purple_account_set_string(account, GOWHATSAPP_SESSION_ENCKEY_KEY, gwamsg->encKey_b64);
            purple_account_set_string(account, GOWHATSAPP_SESSION_MACKEY_KEY, gwamsg->macKey_b64);
            purple_account_set_string(account, GOWHATSAPP_SESSION_WID_KEY, gwamsg->wid);
            if (!PURPLE_CONNECTION_IS_CONNECTED(pc)) {
                // connection has now been established, show it in UI
                purple_connection_update_progress(pc, _("Received session data."), 2, 3);
                purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTED);
                if (purple_account_get_bool(account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
                    gowhatsapp_assume_all_buddies_online(gwa);
                }
            }
            if (purple_account_get_bool(account, GOWHATSAPP_PLAIN_TEXT_LOGIN, FALSE)) {
                const char *username = purple_account_get_username(gwa->account);
                gwamsg->text = g_strdup_printf(_("WhatsApp account %s logged in."), username);
                gowhatsapp_display_message(pc, gwamsg);
            }
            break;

        case gowhatsapp_message_type_login:
            purple_connection_update_progress(pc, _("Waiting for QR code scan"), 1, 3);
            if (purple_account_get_bool(account, GOWHATSAPP_PLAIN_TEXT_LOGIN, FALSE)) {
                char * text = gwamsg->text;
                const char *username = purple_account_get_username(gwa->account);
                gwamsg->text = g_strdup_printf(_("Login data for WhatsApp account %s: %s"), username, text);
                g_free(text);
                gowhatsapp_display_message(pc, gwamsg);
            } else {
                gowhatsapp_display_qrcode(pc, gwamsg->text, gwamsg->blob, gwamsg->blobsize);
            }
            break;

        default:
            gowhatsapp_display_message(pc, gwamsg);
    }

    g_free(gwamsg->id);
    g_free(gwamsg->remoteJid);
    g_free(gwamsg->senderJid);
    g_free(gwamsg->text);
    g_free(gwamsg->blob);
    g_free(gwamsg->clientId);
    g_free(gwamsg->clientToken);
    g_free(gwamsg->serverToken);
    g_free(gwamsg->encKey_b64);
    g_free(gwamsg->macKey_b64);
    g_free(gwamsg->wid);
    //g_free(gwamsg); // TODO: find out why this fails
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
    pc_flags |= PURPLE_CONNECTION_NO_NEWLINES; // TODO: find out how whatsapp represents newlines, use them
    pc_flags |= PURPLE_CONNECTION_NO_BGCOLOR;
    purple_connection_set_flags(pc, pc_flags);

    GoWhatsappAccount *gwa = g_new0(GoWhatsappAccount, 1);
    purple_connection_set_protocol_data(pc, gwa);
    gwa->account = account;
    gwa->pc = pc;

    // load persisted data into memory
    gwa->previous_sessions_last_messages_timestamp = purple_account_get_int(account, GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, 0);
    // this connecting is now considered logging in
    purple_connection_set_state(gwa->pc, PURPLE_CONNECTION_CONNECTING);
    gboolean restore_session = purple_account_get_bool(gwa->account, GOWHATSAPP_RESTORE_SESSION_OPTION, TRUE);
    // where to put downloaded files
    char *download_directory = g_strdup_printf("%s/gowhatsapp", purple_user_dir());
    gowhatsapp_go_login(
        (uintptr_t)pc, // abusing guaranteed-to-be-unique address as connection identifier
        restore_session,
        download_directory);
    g_free(download_directory);
    
}

static void
gowhatsapp_close(PurpleConnection *pc)
{
    GoWhatsappAccount *gwa = purple_connection_get_protocol_data(pc);
    gowhatsapp_go_close((uintptr_t)pc);
    g_free(gwa);
}

static GList *
gowhatsapp_status_types(PurpleAccount *account)
{
    GList *types = NULL;
    PurpleStatusType *status;

    status = purple_status_type_new_full(PURPLE_STATUS_AVAILABLE, GOWHATSAPP_STATUS_STR_ONLINE, _("Online"), TRUE, TRUE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_OFFLINE, GOWHATSAPP_STATUS_STR_OFFLINE, _("Offline"), TRUE, TRUE, FALSE);
    types = g_list_append(types, status);

    status = purple_status_type_new_full(PURPLE_STATUS_MOBILE, GOWHATSAPP_STATUS_STR_MOBILE, NULL, FALSE, FALSE, TRUE);
    types = g_list_prepend(types, status);

    return types;
}

static int
gowhatsapp_send_im(PurpleConnection *pc, const gchar *who, const gchar *message, PurpleMessageFlags flags)
{
    char *w = g_strdup(who); // dumbly removing const qualifier (cgo does not know them)
    char *m = purple_markup_strip_html(message); // related: https://github.com/majn/telegram-purple/issues/12 and https://github.com/majn/telegram-purple/commit/fffe7519d7269cf4e5029a65086897c77f5283ac
    char *msgid = gowhatsapp_go_sendMessage((intptr_t)pc, w, m);
    g_free(w);
    g_free(m);
    if (msgid) {
        gowhatsapp_append_message_id_if_not_exists(pc->account, msgid);
        g_free(msgid);
        return 1; // TODO: wait for server receipt before displaying message locally?
    } else {
        return -70; // was -ECOMM
    }
}

static void
gowhatsapp_free_xfer(PurpleXfer *xfer)
{
}

static void
gowhatsapp_xfer_send_init(PurpleXfer *xfer)
{
    PurpleConnection *pc = purple_account_get_connection(purple_xfer_get_account(xfer));
    const char *who = purple_xfer_get_remote_user(xfer);
    const char *filename = purple_xfer_get_local_filename(xfer);
    char *msgid = gowhatsapp_go_sendMedia((intptr_t)pc, (char *)who, (char *)filename);
    if (msgid) {
        purple_xfer_set_completed(xfer, TRUE);
        //gowhatsapp_append_message_id_if_not_exists(pc->account, msgid);
        g_free(msgid);
    } else {
        purple_xfer_error(purple_xfer_get_type(xfer), pc->account, who,
            _("Sending file failed.")
        ); // TODO: have better error handling: display error and remove xfer
    }
}

PurpleXfer *
gowhatsapp_new_xfer(PurpleConnection *pc, const char *who)
{
    GoWhatsappAccount *gwa = purple_connection_get_protocol_data(pc);
    PurpleXfer *xfer;
    //SkypeWebFileTransfer *swft;
    
    xfer = purple_xfer_new(gwa->account, PURPLE_XFER_TYPE_SEND, who);
    
    /*
    swft = g_new0(SkypeWebFileTransfer, 1);
    swft->sa = sa;
    swft->from = g_strdup(who);
    swft->xfer = xfer;
    purple_xfer_set_protocol_data(xfer, swft);
    * */
    
    purple_xfer_set_init_fnc(xfer, gowhatsapp_xfer_send_init);
    purple_xfer_set_request_denied_fnc(xfer, gowhatsapp_free_xfer);
    purple_xfer_set_cancel_send_fnc(xfer, gowhatsapp_free_xfer);
    
    return xfer;
}

void
gowhatsapp_send_file(PurpleConnection *pc, const gchar *who, const gchar *filename)
{
    PurpleXfer *xfer = gowhatsapp_new_xfer(pc, who);

    if (filename && *filename) {
        purple_xfer_request_accepted(xfer, filename);
    } else {
        purple_xfer_request(xfer);
    }
}

static void
gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group
#if PURPLE_VERSION_CHECK(3, 0, 0)
    , const char *message
#endif
    )
{
    // does not actually do anything. buddy is added to pidgin's local list and is usable from there.
    GoWhatsappAccount *sa = purple_connection_get_protocol_data(pc);
    if (purple_account_get_bool(sa->account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
        gowhatsapp_assume_buddy_online(sa->account, buddy);
    }
}

static GList *
gowhatsapp_add_account_options(GList *account_options)
{
    PurpleAccountOption *option;

    option = purple_account_option_bool_new(
                _("Display all contacts as online"),
                GOWHATSAPP_FAKE_ONLINE_OPTION,
                TRUE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Use stored credentials for login"),
                GOWHATSAPP_RESTORE_SESSION_OPTION,
                TRUE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Plain text login"),
                GOWHATSAPP_PLAIN_TEXT_LOGIN,
                FALSE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_int_new(
                _("Number of received messages to remember as already shown"),
                GOWHATSAPP_MESSAGE_ID_STORE_SIZE_OPTION,
                1000
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Do not show messages older than previous session"),
                GOWHATSAPP_TIMESTAMP_FILTERING_OPTION,
                FALSE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Treat system messages like normal messages (spectrum2 compatibility)"),
                GOWHATSAPP_SYSTEM_MESSAGES_ARE_ORDINARY_MESSAGES_OPTION,
                FALSE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Download files from media (image, audio, video, document) messages"),
                GOWHATSAPP_DOWNLOAD_ATTACHMENTS_OPTION,
                FALSE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Try to download once only"),
                GOWHATSAPP_DOWNLOAD_TRY_ONLY_ONCE_OPTION,
                TRUE
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new(
                _("Display images in conversation window after download"),
                GOWHATSAPP_INLINE_IMAGES_OPTION,
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
    PurplePluginProtocolInfo *prpl_info = g_new0(PurplePluginProtocolInfo, 1); // TODO: this leaks

    info = plugin->info;

    if (info == NULL) {
        plugin->info = info = g_new0(PurplePluginInfo, 1);
    }

    info->name = "Whatsapp (HTTP)";
    info->extra_info = prpl_info;
    prpl_info->options = OPT_PROTO_NO_PASSWORD; // add OPT_PROTO_IM_IMAGE?
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
    prpl_info->send_im = gowhatsapp_send_im;
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
    prpl_info->add_buddy = gowhatsapp_add_buddy;
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
    prpl_info->new_xfer = gowhatsapp_new_xfer;
    prpl_info->send_file = gowhatsapp_send_file;
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
#error Purple 3 not supported.
#endif

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * This allows cgo to acces the PurpleAccount pointer having only an integer which actually is the PurpleConnection pointer.
 */
void *
gowhatsapp_get_account(uintptr_t pc)
{
    return purple_connection_get_account((PurpleConnection *)pc);
}

/*
 * This allows cgo to get a boolean from the current account settings.
 * The PurpleAccount pointer is untyped and the gboolean is implicitly promoted to int.
 */
int
gowhatsapp_account_get_bool(void *account, const char *name, int default_value)
{
    return purple_account_get_bool((const PurpleAccount *)account, name, default_value);
}

/*
 * This allows cgo to get a string from the current account settings.
 * The PurpleAccount pointer is untyped.
 */
const char * gowhatsapp_account_get_string(void *account, const char *name, const char *default_value)
{
    return purple_account_get_string((const PurpleAccount *)account, name, default_value);
}

/*
 * Handler for a message received by go-whatsapp.
 * Called inside of the GTK eventloop.
 *
 * @return Whether to execute again. Always FALSE.
 */
gboolean
gowhatsapp_process_message_bridge_mainthread(gpointer data)
{
    gowhatsapp_process_message((gowhatsapp_message_t *)data);
    return FALSE;
}

/*
 * Handler for a message received by go-whatsapp.
 * Called by go-whatsapp (outside of the GTK eventloop).
 */
void
gowhatsapp_process_message_bridge(void *gwamsg)
{
    purple_timeout_add(0, gowhatsapp_process_message_bridge_mainthread, gwamsg); // yes, this is indeed neccessary – we checked
}
