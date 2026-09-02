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
    // when this connection was established
    time_t connected_at_timestamp;
    // not before this may a new conversation ask the phone for history, see history.c
    time_t history_requests_from;
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
void gowhatsapp_display_text_message(PurpleAccount *account, const gchar * senderJid, const gchar * remoteJid, const gchar * text, const time_t timestamp, const gboolean isGroup, const gboolean isOutgoing, const gchar * name, PurpleMessageFlags flags, const gchar * messageId, const gboolean escape);

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
char *gowhatsapp_get_cb_alias(PurpleConnection *gc, int id, const char *who);

// blist
PurpleBuddy * gowhatsapp_ensure_buddy_in_blist(PurpleAccount *account, const char *remoteJid, const char *display_name);
PurpleChat * gowhatsapp_ensure_group_chat_in_blist(PurpleAccount *account, const char *remoteJid, const char *topic);
PurpleGroup * gowhatsapp_get_purple_group();
PurpleChat * gowhatsapp_find_blist_chat(PurpleAccount *account, const char *jid);
void gowhatsapp_add_buddy(PurpleConnection *pc, PurpleBuddy *buddy, PurpleGroup *group);
void gowhatsapp_tooltip_text(PurpleBuddy *buddy, PurpleNotifyUserInfo *info, gboolean full);
void gowhatsapp_assume_buddy_away(PurpleAccount *account, PurpleBuddy *buddy);
void gowhatsapp_for_all_buddies(PurpleAccount *account, void(*func)(PurpleAccount *, PurpleBuddy *));

// send_message
int gowhatsapp_send_im(PurpleConnection *pc, const gchar *who, const gchar *message, PurpleMessageFlags flags);
unsigned int gowhatsapp_send_typing(PurpleConnection *pc, const gchar *who, PurpleTypingState state);
int gowhatsapp_send_chat(PurpleConnection *pc, int id, const gchar *message, PurpleMessageFlags flags);

// handle_attachment
void gowhatsapp_handle_attachment(gowhatsapp_message_t *gwamsg);
char * gowhatsapp_attachment_fill_template(const char *template, time_t timestamp, const char *hash, const char *filename, const char *extension, const char *remote, const char *sender, const char *title, const char *alias, const char *messageid, PurpleMessageFlags flags);

// send_file
PurpleXfer * gowhatsapp_new_xfer(PurpleConnection *pc, const char *who);
void gowhatsapp_send_file(PurpleConnection *pc, const gchar *who, const gchar *filename);
void gowhatsapp_chat_send_file(PurpleConnection *pc, int id, const char *filename);

// presence
void gowhatsapp_handle_presence(PurpleAccount *account, char *remoteJid, char available, time_t last_seen);
void gowhatsapp_set_presence(PurpleAccount *account, PurpleStatus *status);
void gowhatsapp_subscribe_presence_updates(PurpleAccount *account, PurpleBuddy *buddy);

// profile pictures
void gowhatsapp_request_profile_picture(PurpleAccount *account, PurpleBuddy *buddy);
void gowhatsapp_handle_profile_picture(gowhatsapp_message_t *gwamsg);

// receipts
void gowhatsapp_receipts_init(PurpleConnection *pc);

// history fetched from the primary device
void gowhatsapp_history_init(PurpleConnection *pc);
void gowhatsapp_display_history_message(gowhatsapp_message_t *gwamsg);
void gowhatsapp_history_suppress_requests(gboolean suppress);

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
