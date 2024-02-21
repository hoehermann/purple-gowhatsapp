#include "gowhatsapp.h"
#include "constants.h"
#include "pixbuf.h"

static
void xfer_init_fnc(PurpleXfer *xfer) {
    purple_xfer_start(xfer, -1, NULL, 0); // invokes start_fnc
}

static
void xfer_start_fnc(PurpleXfer * xfer) {
    // TODO: it would be nice to invoke message.Download() just now, not beforehand
    purple_xfer_prpl_ready(xfer); // invokes do_transfer which invokes read_fnc
}

static
gssize xfer_read_fnc(guchar **buffer, PurpleXfer * xfer) {
    // entire attachment already is in memory.
    // just forward the pointer to the destination buffer.
    *buffer = xfer->data;
    xfer->data = NULL; // MEMCHECK: not our memory to free any more
    return purple_xfer_get_size(xfer);
}

static
void xfer_ack_fnc(PurpleXfer * xfer, const guchar * buffer, size_t bytes_read) {
    // This is called after each time xfer_read_fnc returned a positive value.
    // We only do one read, so the transfer is complete now.
    #if PURPLE_VERSION_CHECK(2,14,10)
    purple_xfer_set_completed(xfer, TRUE);
    #endif
}

static
void xfer_release_blob(PurpleXfer * xfer) {
    g_free(xfer->data);
    xfer->data = NULL;
}

static void gowhatsapp_xfer_announce(const gowhatsapp_message_t *gwamsg) {
    const char * template = purple_account_get_string(gwamsg->account, GOWHATSAPP_ATTACHMENT_MESSAGE_OPTION, GOWHATSAPP_ATTACHMENT_MESSAGE_DEFAULT);
    if (template != NULL && !purple_strequal(template, "")) {

        // resolve jid for displaying name
        const char * alias = gwamsg->senderJid; // use the jid by default
        PurpleBuddy * buddy = purple_find_buddy(gwamsg->account, gwamsg->senderJid);
        if (buddy != NULL) {
            alias = purple_buddy_get_contact_alias(buddy);
        }
        
        gowhatsapp_message_t lgwamsg = *gwamsg; // create a local copy of the message struct
        lgwamsg.text = NULL; // MEMCHECK: this pointer is a copy. the original will be released by caller
        lgwamsg.text = g_strdup_printf(template, lgwamsg.name, alias); // MEMCHECK: is released here
        
        #if (GLIB_CHECK_VERSION(2, 68, 0))
        // prepare the templated message
        GString * message = g_string_new(lgwamsg.text);
        // free the templated message
        g_free(lgwamsg.text);
        // replace placeholders with actual data
        g_string_replace(message, "$filename", lgwamsg.name, 0);
        g_string_replace(message, "$sender", alias, 0);
        // get data from filled template
        lgwamsg.text = g_string_free(message, FALSE);
        #else
        #pragma message "Note: GLib version 2.68 needed for g_string_replace attachment message template variables $filename and $sender."
        #endif
        
        // unset lgwamsg.name so gowhatsapp_display_text_message does not mistake it for a pushname
        lgwamsg.name = NULL; // MEMCHECK: this pointer is a copy. the original will be released by caller
        gowhatsapp_display_text_message(purple_account_get_connection(lgwamsg.account), &lgwamsg, PURPLE_MESSAGE_SYSTEM);
        g_free(lgwamsg.text);
    }
}

static
void xfer_download_attachment(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    g_return_if_fail(pc != NULL);
    PurpleAccount *account = purple_connection_get_account(pc);
    
    // an attachment should not have any text
    if (gwamsg->text != NULL && *gwamsg->text != 0) {
        // output a warning if it does
        purple_debug_warning(GOWHATSAPP_NAME, "gwamsg->text set in xfer_download_attachment.\n");
    }
    
    gowhatsapp_xfer_announce(gwamsg);
    
    const char * sender = gwamsg->senderJid; // by default, the group chat participant is the sender
    if (purple_account_get_bool(gwamsg->account, GOWHATSAPP_GROUP_IS_FILE_ORIGIN_OPTION, TRUE)) {
        sender = gwamsg->remoteJid; // set sender to the group chat
    }
    
    PurpleXfer * xfer = purple_xfer_new(account, PURPLE_XFER_RECEIVE, sender);
    purple_xfer_set_filename(xfer, gwamsg->name);
    purple_xfer_set_size(xfer, gwamsg->blobsize);
    xfer->data = gwamsg->blob;
    
    purple_xfer_set_init_fnc(xfer, xfer_init_fnc);
    purple_xfer_set_start_fnc(xfer, xfer_start_fnc);
    purple_xfer_set_read_fnc(xfer, xfer_read_fnc);
    purple_xfer_set_ack_fnc(xfer, xfer_ack_fnc);
    
    // be very sure to release the data no matter what
    purple_xfer_set_end_fnc(xfer, xfer_release_blob);
    purple_xfer_set_request_denied_fnc(xfer, xfer_release_blob);
    purple_xfer_set_cancel_recv_fnc(xfer, xfer_release_blob);
    
    purple_xfer_request(xfer);
    // MEMCHECK NOTE: purple_xfer_unref calls purple_xfer_destroy which MAY call purple_xfer_cancel_local if (purple_xfer_get_status(xfer) == PURPLE_XFER_STATUS_STARTED) which calls cancel_recv and cancel_local
}

void gowhatsapp_handle_attachment(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    const gboolean is_image = gwamsg->subtype == gowhatsapp_attachment_type_image;
    const gboolean is_sticker = gwamsg->subtype == gowhatsapp_attachment_type_sticker;
    const gchar * handle_images = purple_account_get_string(gwamsg->account, GOWHATSAPP_HANDLE_IMAGES_OPTION, GOWHATSAPP_IMAGES_CHOICE_INLINE);
    const gboolean inline_images = purple_strequal(handle_images, GOWHATSAPP_IMAGES_CHOICE_INLINE) || purple_strequal(handle_images, GOWHATSAPP_IMAGES_CHOICE_BOTH);
    const gboolean xfer_images = purple_strequal(handle_images, GOWHATSAPP_IMAGES_CHOICE_XFER) || purple_strequal(handle_images, GOWHATSAPP_IMAGES_CHOICE_BOTH);
    const gboolean inline_stickers = purple_account_get_bool(gwamsg->account, GOWHATSAPP_INLINE_STICKERS_OPTION, TRUE);
    const gboolean inline_this = (is_image && inline_images) || (is_sticker && inline_stickers);
    int img_id = 0;
    if (inline_this && pixbuf_is_loadable_image_mimetype(gwamsg->text)) {
        img_id = purple_imgstore_add_with_id(gwamsg->blob, gwamsg->blobsize, NULL); // MEMCHECK: released including gwamsg->blob by purple_imgstore_unref_by_id (see below)
        if (img_id > 0) {
            // at this point, the image data in gwamsg->blob is not our memory to free any more
            if (xfer_images) {
                // user wants us to offer the image also as xfer, that is why we need a copy of the blob
                // by overwriting the pointer, we lose the adress, but that is okay since PurpleStoredImage took ownership
                gwamsg->blob = g_memdup2(gwamsg->blob, gwamsg->blobsize);
            } else {
                gwamsg->blob = NULL; // MEMCHECK: see comment above
            }
            gchar * text = g_strdup_printf("<img id=\"%u\"/>", img_id); // MEMCHECK: released here
            gowhatsapp_display_message_common(pc, gwamsg->senderJid, gwamsg->remoteJid, text, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, NULL, PURPLE_MESSAGE_IMAGES);
            g_free(text);
            purple_imgstore_unref_by_id(img_id);
        }
    }
    if (gwamsg->blob != NULL) {
        xfer_download_attachment(pc, gwamsg); // MEMCHECK: xfer system takes ownership
    }
}
