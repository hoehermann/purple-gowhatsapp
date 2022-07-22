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
    xfer->data = NULL; // not our memory to free any more
    return purple_xfer_get_size(xfer);
}

static
void xfer_release_blob(PurpleXfer * xfer) {
    g_free(xfer->data);
    xfer->data = NULL;
}

static
void xfer_download_attachment(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    g_return_if_fail(pc != NULL);
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleXfer * xfer = purple_xfer_new(account, PURPLE_XFER_RECEIVE, gwamsg->senderJid);
    //purple_xfer_ref(xfer);
    purple_xfer_set_filename(xfer, gwamsg->name);
    purple_xfer_set_size(xfer, gwamsg->blobsize);
    xfer->data = gwamsg->blob;
    
    purple_xfer_set_init_fnc(xfer, xfer_init_fnc);
    purple_xfer_set_start_fnc(xfer, xfer_start_fnc);
    purple_xfer_set_read_fnc(xfer, xfer_read_fnc);
    
    // be very sure to release the data no matter what
    purple_xfer_set_end_fnc(xfer, xfer_release_blob);
    purple_xfer_set_request_denied_fnc(xfer, xfer_release_blob);
    purple_xfer_set_cancel_recv_fnc(xfer, xfer_release_blob);
    
    purple_xfer_request(xfer);
}

void gowhatsapp_handle_attachment(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    const gboolean is_image = gwamsg->subtype == gowhatsapp_attachment_type_image;
    const gboolean is_sticker = gwamsg->subtype == gowhatsapp_attachment_type_sticker;
    const gboolean inline_images = purple_account_get_bool(gwamsg->account, GOWHATSAPP_INLINE_IMAGES_OPTION, TRUE);
    const gboolean inline_stickers = purple_account_get_bool(gwamsg->account, GOWHATSAPP_INLINE_STICKERS_OPTION, TRUE);
    const gboolean inline_this = (is_image && inline_images) || (is_sticker && inline_stickers);
    int img_id = -1;
    if (inline_this && pixbuf_is_loadable_image_mimetype(gwamsg->text)) {
        img_id = purple_imgstore_add_with_id(gwamsg->blob, gwamsg->blobsize, NULL);
        if (img_id >= 0) {
            // strip information from mis-used fields so they are not wrongly interpreted by gowhatsapp_display_text_message
            g_free(gwamsg->name); gwamsg->name = NULL; 
            g_free(gwamsg->text); gwamsg->text = NULL; 
            gwamsg->text = g_strdup_printf("<img id=\"%u\"/>", img_id);
            gowhatsapp_display_text_message(pc, gwamsg, PURPLE_MESSAGE_IMAGES);
        }
    }
    if (img_id < 0) {
        xfer_download_attachment(pc, gwamsg);
    }
}
