#include "bridge.h"
#include "purple_compat.h"

#if !(GLIB_CHECK_VERSION(2, 67, 3))
#define g_memdup2 g_memdup
#endif

#define GOWHATSAPP_STATUS_STR_ONLINE   "online"
#define GOWHATSAPP_STATUS_STR_OFFLINE  "offline"
#define GOWHATSAPP_STATUS_STR_MOBILE   "mobile"
#define GOWHATSAPP_STATUS_STR_AWAY     "away"

typedef struct {
} WhatsappAccountData;

// login
void gowhatsapp_login(PurpleAccount *account);
void gowhatsapp_close(PurpleConnection *pc);

// qrcode
void gowhatsapp_handle_qrcode(PurpleConnection *pc, const char *challenge, const char *terminal, void *image_data, size_t image_data_len);
void gowhatsapp_close_qrcode(PurpleAccount *account);

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
void gowhatsapp_join_chat(PurpleConnection *pc, GHashTable *data);
char *gowhatsapp_get_chat_name(GHashTable *components);
PurpleRoomlist *gowhatsapp_roomlist_get_list(PurpleConnection *pc);
void gowhatsapp_set_chat_topic(PurpleConnection *pc, int id, const char *topic);
gchar *gowhatsapp_roomlist_serialize(PurpleRoomlistRoom *room);

// blist
void gowhatsapp_ensure_buddy_in_blist(PurpleAccount *account, char *remoteJid, char *display_name);
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(PurpleAccount *account, const char *remoteJid, const char *topic);
PurpleGroup * gowhatsapp_get_purple_group();
void gowhatsapp_assume_all_buddies_online(PurpleAccount *account);
PurpleChat * gowhatsapp_find_blist_chat(PurpleAccount *account, const char *jid);
void gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group);

// send_message
int gowhatsapp_send_im(PurpleConnection *pc, const gchar *who, const gchar *message, PurpleMessageFlags flags);
int gowhatsapp_send_chat(PurpleConnection *pc, int id, const gchar *message, PurpleMessageFlags flags);

// handle_attachment
void gowhatsapp_handle_attachment(PurpleConnection *pc, gowhatsapp_message_t *gwamsg);

// send_file
PurpleXfer * gowhatsapp_new_xfer(PurpleConnection *pc, const char *who);
void gowhatsapp_send_file(PurpleConnection *pc, const gchar *who, const gchar *filename);
