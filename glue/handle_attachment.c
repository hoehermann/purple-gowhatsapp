#include "gowhatsapp.h"
#include "libwhatsmeow.h"
#include "constants.h"
#include "pixbuf.h"

static void gowhatsapp_display_image_inline(gowhatsapp_message_t *gwamsg, const char *local_file_path) {
    const gboolean is_image = gwamsg->subtype == gowhatsapp_attachment_type_image;
    const gboolean is_sticker = gwamsg->subtype == gowhatsapp_attachment_type_sticker;
    const gboolean inline_images = purple_account_get_bool(gwamsg->account, GOWHATSAPP_INLINE_IMAGES_OPTION, TRUE);
    if (inline_images && (is_image || is_sticker) && pixbuf_is_loadable_image_mimetype(gwamsg->mimetype)) {
        gchar *data = NULL;
	    size_t len;
	    GError *err = NULL;
	    if (g_file_get_contents(local_file_path, &data, &len, &err)) {
            int img_id = purple_imgstore_add_with_id(data, len, NULL); // MEMCHECK: released by purple_imgstore_unref_by_id (see below)
            if (img_id > 0) {
                // at this point, the image data in gwamsg->blob is not our memory to free any more
                gchar * text = g_strdup_printf("<img id=\"%u\"/>", img_id); // MEMCHECK: released here
                gowhatsapp_display_text_message(gwamsg->account, gwamsg->senderJid, gwamsg->remoteJid, text, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, NULL, PURPLE_MESSAGE_IMAGES, gwamsg->messageId, FALSE);
                g_free(text);
                purple_imgstore_unref_by_id(img_id);
            }
        }
    }
}

static void gowhatsapp_display_caption(gowhatsapp_message_t *gwamsg) {
    if (gwamsg->text && gwamsg->text[0]) {
        gowhatsapp_display_text_message(gwamsg->account, gwamsg->senderJid, gwamsg->remoteJid, gwamsg->text, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, gwamsg->name, 0, gwamsg->messageId, TRUE);
    }
}

// This is called after the user accepted the file transfer (and chose a destination)
static void xfer_init(PurpleXfer *xfer) {
    gowhatsapp_message_t * gwamsg = xfer->data;
    const char * local_file_name = purple_xfer_get_local_filename(xfer);
    PurpleAccount *account = purple_xfer_get_account(xfer);
    char *error = gowhatsapp_go_download_attachment(account, (char *)local_file_name, gwamsg->download_handle);
    if (error && error[0]) {
        purple_xfer_error(purple_xfer_get_type(xfer), account, xfer->who, error); 
        purple_xfer_cancel_local(xfer);
    } else {
        purple_xfer_set_bytes_sent(xfer, purple_xfer_get_size(xfer));
        purple_xfer_set_completed(xfer, TRUE);
        gowhatsapp_display_image_inline(gwamsg, local_file_name);
        gowhatsapp_display_caption(gwamsg);
    }
    g_free(error);
}

static void xfer_release(PurpleXfer * xfer) {
    purple_debug_info(GOWHATSAPP_NAME, "xfer_release(…) called.\n");
    if (xfer->data != NULL) {
        gowhatsapp_message_t * gwamsg = xfer->data;
        gowhatsapp_go_download_attachment(gwamsg->account, NULL, gwamsg->download_handle); // free the handle
        gowhatsapp_free_message(gwamsg);
        xfer->data = NULL;
    }
}

static gowhatsapp_message_t *duplicate_gowhatsapp_message(gowhatsapp_message_t *gwamsg) {
    // NOTE: this is tailored for the use in handle_attachment
    gowhatsapp_message_t *clone = g_new0(gowhatsapp_message_t, 1);
    clone->download_handle = gwamsg->download_handle;
    clone->account = gwamsg->account;
    clone->senderJid = g_strdup(gwamsg->senderJid);
    clone->remoteJid = g_strdup(gwamsg->remoteJid);
    clone->text = g_strdup(gwamsg->text);
    clone->timestamp = gwamsg->timestamp;
    clone->isGroup = gwamsg->isGroup;
    clone->isOutgoing = gwamsg->isOutgoing;
    clone->name = g_strdup(gwamsg->name);
    clone->subtype = gwamsg->subtype;
    clone->messageId = g_strdup(gwamsg->messageId);
    clone->mimetype = g_strdup(gwamsg->mimetype);
    return clone;
}

static void xfer_download_attachment(gowhatsapp_message_t *gwamsg) {    
    const char * sender = gwamsg->senderJid; // by default, the group chat participant is the sender
    if (purple_account_get_bool(gwamsg->account, GOWHATSAPP_GROUP_IS_FILE_ORIGIN_OPTION, TRUE)) {
        sender = gwamsg->remoteJid; // set sender to the group chat
    }
    
    PurpleXfer * xfer = purple_xfer_new(gwamsg->account, PURPLE_XFER_RECEIVE, sender);
    char *filename = g_strdup_printf("%s%s%s", gwamsg->hash_hex, gwamsg->filename, gwamsg->extension);
    purple_xfer_set_filename(xfer, filename);
    g_free(filename);
    purple_xfer_set_size(xfer, gwamsg->filesize);
    // NOTE: xfer->message cannot be used for the caption since in purple_xfer_ask_recv message is automatically written to the conversation of the sender, but purple_xfer_ask_recv does not consider the case where the sender is a chat. also purple_xfer_ask_recv disregards the message timestamp
    xfer->data = duplicate_gowhatsapp_message(gwamsg);
    
    purple_xfer_set_init_fnc(xfer, xfer_init);
    
    // be very sure to release the data no matter what
    purple_xfer_set_end_fnc(xfer, xfer_release);
    purple_xfer_set_request_denied_fnc(xfer, xfer_release);
    purple_xfer_set_cancel_recv_fnc(xfer, xfer_release);
    
    purple_xfer_request(xfer);
    // MEMCHECK NOTE: purple_xfer_unref calls purple_xfer_destroy which MAY call purple_xfer_cancel_local if (purple_xfer_get_status(xfer) == PURPLE_XFER_STATUS_STARTED) which calls cancel_recv and cancel_local
}

static void replace_placeholder(gpointer key, gpointer value, gpointer user_data) {
    char **text = user_data;
    // NOTE: I am not using g_string_replace here since the GLib shipped with win32 Pidgin is ancient
    char *replaced = purple_strreplace(*text, key, value);
    g_free(*text);
    *text = replaced;
}

char * gowhatsapp_attachment_fill_template(const char *template, time_t timestamp, const char *hash, const char *filename, const char *extension, const char *remote, const char *sender, const char *messageid, PurpleMessageFlags flags) {
    // in case of chats, remote and sender may be different
    // but in case of direct messages, they are the same
    // I do not want the sender to appear twice
    if (purple_strequal(remote, sender)) {
        sender = "";
    }
    const char *direction = "";
    if (flags & PURPLE_MESSAGE_RECV) {
        direction = "received";
    }
    if (flags & PURPLE_MESSAGE_SEND) {
        direction = "sent";
    }

    // this hash table does not release keys or values since everything is either static or not owned by this function
    GHashTable *replacements = g_hash_table_new_full(g_str_hash, g_str_equal, NULL, NULL);
    // casts necessary to remove const
    g_hash_table_insert(replacements, "$home", (char *)purple_home_dir());
    g_hash_table_insert(replacements, "$purple", (char *)purple_user_dir());
    g_hash_table_insert(replacements, "$hash", (char *)hash);
    g_hash_table_insert(replacements, "$direction", (char *)direction);
    g_hash_table_insert(replacements, "$remote", (char *)remote);
    g_hash_table_insert(replacements, "$sender", (char *)sender);
    g_hash_table_insert(replacements, "$messageid", (char *)messageid);
    g_hash_table_insert(replacements, "$extension", (char *)extension);
    g_hash_table_insert(replacements, "$filename", (char *)filename);

    char *replaced = g_strdup(purple_utf8_strftime(template, localtime(&timestamp)));
    g_hash_table_foreach(replacements, replace_placeholder, &replaced);
    return replaced;
}

#ifndef WIN32
#include <unistd.h> // for symlink

// This feature will not be implemented on win32, since creating directory junctions via reparse-points is insanely cumbersome:
// https://stackoverflow.com/questions/1400549/in-net-how-do-i-create-a-junction-in-ntfs-as-opposed-to-a-symlink

void create_symlinks_recurse(char *path, char *aliased_path) {
    //purple_debug_info(GOWHATSAPP_NAME, "create_symlinks_recurse(%s, %s)…\n", aliased_path, path);
    if (strlen(path) <= 1 || strlen(aliased_path) <= 1) {
        // we reached / or . – stop recursion
        return;
    }
    char *parent_directory = g_path_get_dirname(path);
    char *aliased_parent_directory = g_path_get_dirname(aliased_path);
    create_symlinks_recurse(parent_directory, aliased_parent_directory);
    g_free(parent_directory);
    g_free(aliased_parent_directory);
    if (symlink(path, aliased_path) == 0) {
        purple_debug_info(GOWHATSAPP_NAME, "Created symlink „%s“ → „%s“.\n", aliased_path, path);
        // TODO: return aliased path, show that in conversation window
    }
}

char * create_symlinks(PurpleAccount *account, const char *template, time_t timestamp, const char *hash, const char *filename, const char *extension, const char *remote, const char *sender, const char *messageid, PurpleMessageFlags flags) {
    const char *chat_alias = remote;
    const char *buddy_alias = sender;
    PurpleBuddy *buddy = purple_blist_find_buddy(account, sender);
    if (buddy) {
        const char *alias = purple_buddy_get_alias(buddy);
        // do not use alias if it is NULL, empty or containing directory separator (characters unfit for use in file-system are not checked or escaped)
        if (alias != NULL && *alias != 0 && strchr(alias, '/') == NULL) {
            buddy_alias = alias;
        }
    }
    PurpleChat *chat = purple_blist_find_chat(account, remote);
    if (chat) {
        const char *alias = purple_chat_get_name(chat);
        // do not use alias if it is NULL, empty or containing directory separator (characters unfit for use in file-system are not checked or escaped)
        if (alias != NULL && *alias != 0 && strchr(alias, '/') == NULL) {
            chat_alias = alias;
        }
    }
    if (purple_strequal(remote, sender)) {
        // chat is contact (direct message)
        chat_alias = buddy_alias;
    } else {
        // group chat
    }
    // TODO: always store files with their hash, then provide symlink with the filename?
    char *aliased_path = gowhatsapp_attachment_fill_template(template, timestamp, hash, filename, extension, chat_alias, buddy_alias, messageid, flags);
    char *path = gowhatsapp_attachment_fill_template(template, timestamp, hash, filename, extension, remote, sender, messageid, flags);
    create_symlinks_recurse(path, aliased_path);
    g_free(aliased_path);
    g_free(path);
}
#endif

void gowhatsapp_handle_attachment(gowhatsapp_message_t *gwamsg) {
    // TODO: mention in readme: for maintaining order of messages, do not use purple's xfer mechanism
    const char *local_path_template = purple_account_get_string(gwamsg->account, GOWHATSAPP_ATTACHMENT_PATH_TEMPLATE_OPTION, GOWHATSAPP_ATTACHMENT_PATH_TEMPLATE_DEFAULT);
    // auto-downloader
    if (local_path_template && local_path_template[0]) {
        // assume a contact sent this file
        PurpleMessageFlags flags = PURPLE_MESSAGE_RECV;
        if (purple_strequal(purple_account_get_username(gwamsg->account), gwamsg->senderJid)) {
            // we actually sent this file (from a different device)
            flags = PURPLE_MESSAGE_SEND | PURPLE_MESSAGE_REMOTE_SEND;
        }
        char *local_path = gowhatsapp_attachment_fill_template(local_path_template, gwamsg->timestamp, gwamsg->hash_hex, gwamsg->filename, gwamsg->extension, gwamsg->remoteJid, gwamsg->senderJid, gwamsg->messageId, flags);
        char *error = gowhatsapp_go_download_attachment(gwamsg->account, local_path, gwamsg->download_handle);
        if (error && error[0]) {
            gowhatsapp_display_text_message(gwamsg->account, gwamsg->senderJid, gwamsg->remoteJid, error, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, gwamsg->name, PURPLE_MESSAGE_ERROR, gwamsg->messageId, TRUE);
        } else {
            #ifndef WIN32
            create_symlinks(gwamsg->account, local_path_template, gwamsg->timestamp, gwamsg->hash_hex, gwamsg->filename, gwamsg->extension, gwamsg->remoteJid, gwamsg->senderJid, gwamsg->messageId, flags);
            #endif
            const char *url_template = purple_account_get_string(gwamsg->account, GOWHATSAPP_ATTACHMENT_URL_TEMPLATE_OPTION, GOWHATSAPP_ATTACHMENT_URL_TEMPLATE_DEFAULT);
            char *url = gowhatsapp_go_url_from_local_path(local_path);
            if (url_template && url_template[0]) {
                url = gowhatsapp_attachment_fill_template(url_template, gwamsg->timestamp, gwamsg->hash_hex, gwamsg->filename, gwamsg->extension, gwamsg->remoteJid, gwamsg->senderJid, gwamsg->messageId, flags);
            }
            gowhatsapp_display_text_message(gwamsg->account, gwamsg->senderJid, gwamsg->remoteJid, url, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, gwamsg->name, 0, gwamsg->messageId, TRUE);
            g_free(url);
            gowhatsapp_display_image_inline(gwamsg, local_path);
            gowhatsapp_display_caption(gwamsg);
        }
        g_free(error);
        g_free(local_path);
    } else {
        xfer_download_attachment(gwamsg);
    }
}
