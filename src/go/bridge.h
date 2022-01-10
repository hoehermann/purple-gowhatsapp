#ifndef _BRIDGE_H_
#define _BRIDGE_H_

#include <stdint.h> // for int64_t
#include <time.h> // for time_t

// for querying current settings
// these signatures are redefinitions taken from purple.h
// CGO needs to have them re-declared as external
#ifndef _PURPLE_ACCOUNT_H_
#define _PURPLE_ACCOUNT_H_
struct _PurpleAccount;
typedef struct _PurpleAccount PurpleAccount;
extern int gowhatsapp_account_exists(PurpleAccount *account);
extern int purple_account_get_int(PurpleAccount *account, const char *name, int default_value);
extern const char * purple_account_get_string(PurpleAccount *account, const char *name, const char *default_value);
extern void purple_account_set_password(PurpleAccount *account, const char *password);
#endif

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_connected,
    gowhatsapp_message_type_disconnected,
    gowhatsapp_message_type_name,
    gowhatsapp_message_type_typing,
    gowhatsapp_message_type_typing_stopped,
    gowhatsapp_message_type_presence,
    gowhatsapp_message_type_attachment,
    gowhatsapp_message_type_log
};

// Structure to communicate go → purple.
// This holds all data for incoming messages, error messages, login data, etc.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    PurpleAccount *account; /// pointer identifying the account
    char *remoteJid; /// conversation identifier (may be a single contact or a group)
    char *senderJid; /// message author's identifier (useful in group chats)
    char *text; /// the message payload (interpretation depends on type)
    char *name; /// remote user's name (chosen by them) or filename (in case of attachment)
    void *blob; /// binary payload (used for inlining images)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char msgtype; /// message type – see above
    char isGroup; /// this is a group chat message
    char level; /// loglevel, error severity, or online-state
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
};
typedef struct gowhatsapp_message gowhatsapp_message_t;

// for feeding messages from go into purple
extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);

#endif
