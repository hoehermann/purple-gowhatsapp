#include "gowhatsapp.h"
#include "constants.h"

static void
null_cb(PurpleAccount *account, PurpleRequestFields *fields) {
}

static void
dismiss_cb(PurpleAccount *account, PurpleRequestFields *fields) {
    PurpleConnection *pc = purple_account_get_connection(account);
    purple_connection_error(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, "QR code was dismissed.");
}

void
gowhatsapp_close_qrcode(PurpleAccount *account)
{
    purple_request_close_with_handle(account); // close all currently open request fields, if any
}

static void
gowhatsapp_display_qrcode(PurpleAccount *account, const char * challenge, void * image_data, size_t image_data_len)
{
    g_return_if_fail(account != NULL);

    PurpleRequestFields *fields = purple_request_fields_new();
    PurpleRequestFieldGroup *group = purple_request_field_group_new(NULL);
    purple_request_fields_add_group(fields, group);

    PurpleRequestField *string_field = purple_request_field_string_new("qr_string", "QR Code Data", challenge, FALSE);
    purple_request_field_group_add_field(group, string_field);
    PurpleRequestField *image_field = purple_request_field_image_new("qr_image", "QR Code Image", image_data, image_data_len);
    purple_request_field_group_add_field(group, image_field);

    const char *username = purple_account_get_username(account);
    char *secondary = g_strdup_printf("WhatsApp account %s (multi-device mode must be enabled)", username); // MEMCHECK: released here

    gowhatsapp_close_qrcode(account);
    purple_request_fields(
        account, /*handle*/
        "Logon QR Code", /*title*/
        "Please scan this QR code with your phone", /*primary*/
        secondary, /*secondary*/
        fields, /*fields*/
        "OK", G_CALLBACK(null_cb), /*OK*/
        "Dismiss", G_CALLBACK(dismiss_cb), /*Cancel*/
        NULL, /*account*/
        username, /*username*/
        NULL, /*conversation*/
        account /*data*/
    );
    
    g_free(secondary);
}

void
gowhatsapp_handle_qrcode(PurpleConnection *pc, gowhatsapp_message_t *gwamsg)
{
    PurpleRequestUiOps *ui_ops = purple_request_get_ui_ops();
    if (!ui_ops->request_fields || gwamsg->blobsize <= 0) {
        // The UI hasn't implemented the func we want, just output as a message instead
        PurpleMessageFlags flags = PURPLE_MESSAGE_RECV;
        gchar *msg_out;
        int img_id = -1;
        if (gwamsg->blobsize > 0) {
            img_id = purple_imgstore_add_with_id(gwamsg->blob, gwamsg->blobsize, NULL); // MEMCHECK: imgstore does NOT make a copy
        }
        if (img_id >= 0) {
            gwamsg->blob = NULL; // MEMCHECK: not our memory to free any more
            msg_out = g_strdup_printf( // MEMCHECK: msg_out released here (see below)
                "%s<br /><img id=\"%u\" alt=\"%s\"/><br />%s", 
                "Please scan this QR code with your phone and WhatsApp multi-device mode enabled:", img_id, gwamsg->text, gwamsg->name
            );
            flags |= PURPLE_MESSAGE_IMAGES;
        } else {
            msg_out = g_strdup_printf( // MEMCHECK: msg_out released here (see below)
                "%s<br />%s<br />%s", 
                "Please scan this QR code with your phone and WhatsApp multi-device mode enabled:", gwamsg->text, gwamsg->name
            );
        }
        purple_serv_got_im(pc, "Logon QR Code", msg_out, flags, time(NULL));
        g_free(msg_out);
    } else {
        PurpleAccount *account = purple_connection_get_account(pc);
        gowhatsapp_display_qrcode(account, gwamsg->text, gwamsg->blob, gwamsg->blobsize);
    }
    g_free(gwamsg->blob);
}
