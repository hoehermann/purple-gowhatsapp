#include "purple_compat.h"
#include "bridge.h"

#define GOWHATSAPP_NAME "whatsmeow"  // name to refer to this plug-in (in logs)
#define GOWHATSAPP_PRPL_ID "prpl-hehoe-whatsmeow"

#define GOWHATSAPP_STATUS_STR_AVAILABLE "available" // this must match whatsmeow's types.PresenceAvailable
#define GOWHATSAPP_STATUS_STR_AWAY      "unavailable" // this must match whatsmeow's types.PresenceUnavailable
#define GOWHATSAPP_STATUS_STR_OFFLINE   "offline"
#define GOWHATSAPP_STATUS_STR_MOBILE    "mobile"

// protocol data for one connection
typedef struct {
    // reference to roomlist which is currently being populated in asynchronous calls
    PurpleRoomlist *roomlist;
} WhatsappProtocolData;

// options
GList *gowhatsapp_add_account_options(GList *account_options);

// login
void gowhatsapp_login(PurpleAccount *account);
void gowhatsapp_close(PurpleConnection *pc);
void gowhatsapp_store_credentials(PurpleAccount *account, char *credentials);

// qrcode
void gowhatsapp_handle_qrcode(PurpleConnection *pc, gowhatsapp_message_t *gwamsg);
void gowhatsapp_close_qrcode(PurpleAccount *account);

// process_message
void gowhatsapp_process_message(gowhatsapp_message_t *gwamsg);

// display_message
void gowhatsapp_display_text_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg, PurpleMessageFlags flags);

// message_filtering
gboolean gowhatsapp_append_message_id_if_not_exists(PurpleAccount *account, char *message_id);
gboolean gowhatsapp_message_is_new_enough(PurpleAccount *account, const time_t ts);

// groups
PurpleConversation *gowhatsapp_enter_group_chat(PurpleConnection *pc, const char *remoteJid, char **participants);
void gowhatsapp_join_chat(PurpleConnection *pc, GHashTable *data);
char *gowhatsapp_get_chat_name(GHashTable *components);
PurpleRoomlist *gowhatsapp_roomlist_get_list(PurpleConnection *pc);
void gowhatsapp_set_chat_topic(PurpleConnection *pc, int id, const char *topic);
gchar *gowhatsapp_roomlist_serialize(PurpleRoomlistRoom *room);
GList * gowhatsapp_chat_info(PurpleConnection *pc);
GHashTable * gowhatsapp_chat_info_defaults(PurpleConnection *pc, const char *chat_name);
void gowhatsapp_chat_set_participants(PurpleConvChat *conv_chat, char **participants);
void gowhatsapp_roomlist_add_room(PurpleConnection *pc, char *remoteJid, char *name);
void gowhatsapp_handle_group(PurpleConnection *pc, gowhatsapp_message_t *gwamsg);
void gowhatsapp_free_name(PurpleConversation *conv);

// blist
void gowhatsapp_ensure_buddy_in_blist(PurpleAccount *account, char *remoteJid, char *display_name);
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(PurpleAccount *account, const char *remoteJid, const char *topic);
PurpleGroup * gowhatsapp_get_purple_group();
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
void gowhatsapp_chat_send_file(PurpleConnection *pc, int id, const char *filename);

// presence
void gowhatsapp_handle_presence(PurpleAccount *account, char *remoteJid, char available, time_t last_seen);
void gowhatsapp_tooltip_text(PurpleBuddy *buddy, PurpleNotifyUserInfo *info, gboolean full);
void gowhatsapp_handle_profile_picture(gowhatsapp_message_t *gwamsg);
void gowhatsapp_set_presence(PurpleAccount *account, PurpleStatus *status);
void gowhatsapp_subscribe_presence_updates(PurpleAccount *account, PurpleBuddy *buddy);

// receipts
void gowhatsapp_receipts_init(PurpleConnection *pc);

// commands
enum gowhatsapp_command {
    GOWHATSAPP_COMMAND_NONE = 0,
    GOWHATSAPP_COMMAND_VERSIONS,
    GOWHATSAPP_COMMAND_CONTACTS,
    GOWHATSAPP_COMMAND_PARTICIPANTS,
    GOWHATSAPP_COMMAND_PRESENCE,
    GOWHATSAPP_COMMAND_LOGOUT
};
enum gowhatsapp_command is_command(const char *message);
int execute_command(PurpleConnection *pc, const gchar *message, const gchar *who, PurpleConversation *conv);
