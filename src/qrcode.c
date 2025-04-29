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
gowhatsapp_display_qrcode(PurpleAccount *account, const char *pairing_code, const char *qr_data, void * image_data, size_t image_data_len)
{
    g_return_if_fail(account != NULL);

    PurpleRequestFields *fields = purple_request_fields_new();
    PurpleRequestFieldGroup *group = purple_request_field_group_new(NULL);
    purple_request_fields_add_group(fields, group);

    {
        PurpleRequestField *string_code = purple_request_field_string_new("pairing_code", "Pairing Code", pairing_code, FALSE);
        purple_request_field_group_add_field(group, string_code);
    }
    {
        PurpleRequestField *string_field = purple_request_field_string_new("qr_data", "QR Code Data", qr_data, FALSE);
        purple_request_field_group_add_field(group, string_field);
    }
    {
        PurpleRequestField *image_field = purple_request_field_image_new("qr_image", "QR Code Image", image_data, image_data_len);
        purple_request_field_group_add_field(group, image_field);
    }

    const char *username = purple_account_get_username(account);
    char *secondary = g_strdup_printf("WhatsApp account %s", username); // MEMCHECK: released here

    gowhatsapp_close_qrcode(account);
    purple_request_fields(
        account, /*handle*/
        "Logon QR Code", /*title*/
        "Please enter pairing code or scan the QR code", /*primary*/
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
    if (!ui_ops || !ui_ops->request_fields || gwamsg->blobsize <= 0) {
        // The UI hasn't implemented the func we want, just output as a message instead
        PurpleMessageFlags flags = PURPLE_MESSAGE_RECV;
        int img_id = 0;
        if (gwamsg->blobsize > 0) {
            img_id = purple_imgstore_add_with_id(gwamsg->blob, gwamsg->blobsize, NULL); // MEMCHECK: released including gwamsg->blob by purple_imgstore_unref_by_id (see below)
        }
        gchar *msg_img = NULL;
        if (img_id > 0) {
            gwamsg->blob = NULL; // MEMCHECK: not our memory to free any more
            msg_img = g_strdup_printf("<img id=\"%u\"/>", img_id); // MEMCHECK: released here (see below)
            flags |= PURPLE_MESSAGE_IMAGES;
        } else {
            // NOTE: This turns the newlines into br-tags. Front-ends should know what they are doing.
            gchar * qrterminal_html = purple_markup_escape_text(gwamsg->pairing_qrterminal, -1); // MEMCHECK: released here (see below)
            msg_img = g_strdup_printf("Your UI does not handle images. The next lines emulate the QR code with text characters. If viewed with a mono-spaced font, scanning may succeed. In case you see the raw HTML (with br-tags), you need to convert them into newlines first.<br/>%s", qrterminal_html); // MEMCHECK: released here (see below)
            g_free(qrterminal_html);
        }
        gchar *msg_out = g_strdup_printf(
            "Please enter pairing code %s or scan the QR code with your phone.<br/>%s<br/>In case the QR code above does not work, this is the challenge data. Use the QR code generator of your choice to turn it into an image:<br/>%s",
            gwamsg->pairing_code,
            msg_img,
            gwamsg->pairing_qrdata
        ); // MEMCHECK: released here (see below)
        g_free(msg_img);
        const gchar *who = "Logon QR Code";
        purple_serv_got_im(pc, who, msg_out, flags, time(NULL));
        g_free(msg_out);
        if (img_id > 0) {
            purple_imgstore_unref_by_id(img_id);
        }
    } else {
        PurpleAccount *account = purple_connection_get_account(pc);
        gowhatsapp_display_qrcode(account, gwamsg->pairing_code, gwamsg->pairing_qrdata, gwamsg->blob, gwamsg->blobsize);
    }
    g_free(gwamsg->blob);
}
