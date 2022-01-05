#include <purple.h>
#include "purple-go-whatsapp.h"

void gowhatsapp_login(PurpleAccount *account);
void gowhatsapp_close(PurpleConnection *pc);
void gowhatsapp_display_qrcode(PurpleConnection *pc, const char * qr_code_raw, void * image_data, size_t image_data_len);
void gowhatsapp_process_message(PurpleAccount *account, gowhatsapp_message_t *gwamsg);
