#include "bridge.h"
#include "purple_compat.h"

#define GOWHATSAPP_STATUS_STR_ONLINE   "online"
#define GOWHATSAPP_STATUS_STR_OFFLINE  "offline"
#define GOWHATSAPP_STATUS_STR_MOBILE   "mobile"

// login
void gowhatsapp_login(PurpleAccount *account);
void gowhatsapp_close(PurpleConnection *pc);

// qrcode
void gowhatsapp_display_qrcode(PurpleConnection *pc, const char * qr_code_raw, void * image_data, size_t image_data_len);

// process_message
void gowhatsapp_process_message(PurpleAccount *account, gowhatsapp_message_t *gwamsg);

// display_message
void gowhatsapp_display_text_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg);

// message_filtering
gboolean gowhatsapp_append_message_id_if_not_exists(PurpleAccount *account, char *message_id);
gboolean gowhatsapp_message_is_new_enough(PurpleAccount *account, const time_t ts);

// groups
PurpleConvChat *gowhatsapp_find_group_chat(const char *remoteJid, const char *senderJid, const char *topic, PurpleConnection *pc);
int gowhatsapp_user_in_conv_chat(PurpleConvChat *conv_chat, const char *userJid);
int gowhatsapp_remotejid_is_group_chat(char *remoteJid);

// blist
void gowhatsapp_ensure_buddy_in_blist(PurpleAccount *account, char *remoteJid, char *display_name);
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(PurpleAccount *account, const char *remoteJid, const char *topic);
PurpleGroup * gowhatsapp_get_purple_group();
