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
    // NOTE: I am not using g_string_replace here since the GLib shipped with win32 Pidgin is ancient
    char *template1 = purple_strreplace(purple_utf8_strftime(template, localtime(&timestamp)), "$hash", hash);
    char *template2 = purple_strreplace(template1, "$direction", direction);
    char *template3 = purple_strreplace(template2, "$extension", extension);
    char *template4 = purple_strreplace(template3, "$remote", remote);
    char *template5 = purple_strreplace(template4, "$sender", sender);
    char *template6 = purple_strreplace(template5, "$messageid", messageid);
    char *template7 = purple_strreplace(template6, "$filename", filename);
    g_free(template1);
    g_free(template2);
    g_free(template3);
    g_free(template4);
    g_free(template5);
    g_free(template6);
    return template7;
}

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
