#include "gowhatsapp.h"

static void
null_cb() {
}

void
gowhatsapp_display_qrcode(PurpleConnection *pc, const char * qr_code_raw, void * image_data, size_t image_data_len)
{
    g_return_if_fail(pc != NULL);
    PurpleAccount *account = purple_connection_get_account(pc);

    PurpleRequestFields *fields = purple_request_fields_new();
    PurpleRequestFieldGroup *group = purple_request_field_group_new(NULL);
    purple_request_fields_add_group(fields, group);

    PurpleRequestField *string_field = purple_request_field_string_new("qr_string", "QR Code Data", g_strdup(qr_code_raw), FALSE);
    purple_request_field_group_add_field(group, string_field);
    PurpleRequestField *image_field = purple_request_field_image_new("qr_image", "QR Code Image", image_data, image_data_len);
    purple_request_field_group_add_field(group, image_field);
    
    g_free(image_data); // purple_request_field_image_new maintains an internal copy

    const char *username = g_strdup(purple_account_get_username(account));
    const char *secondary = g_strdup_printf("WhatsApp account %s", username);

    purple_request_fields(
        pc, /*handle*/
        "Logon QR Code", /*title*/
        "Please scan this QR code with your phone", /*primary*/
        secondary, /*secondary*/
        fields, /*fields*/
        "OK", G_CALLBACK(null_cb), /*OK*/
        "Dismiss", G_CALLBACK(null_cb), /*Cancel*/
        NULL, /*account*/
        username, /*username*/
        NULL, /*conversation*/
        NULL /*data*/
    );
}
