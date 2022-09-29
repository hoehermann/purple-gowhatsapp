#ifndef _BRIDGE_H_
#define _BRIDGE_H_

#include <stdint.h> // for int64_t
#include <time.h> // for time_t
#include <stdlib.h> // for calloc

// for querying current settings
// these signatures are redefinitions taken from purple.h
// CGO needs to have them re-declared as external
#ifndef _PURPLE_ACCOUNT_H_
#define _PURPLE_ACCOUNT_H_
struct _PurpleAccount;
typedef struct _PurpleAccount PurpleAccount;
extern int gowhatsapp_account_exists(PurpleAccount *account);
extern int purple_account_get_int(PurpleAccount *account, const char *name, int default_value);
extern int purple_account_get_bool(PurpleAccount *account, const char *name, int default_value); // assumes gboolean is int
extern const char * purple_account_get_string(PurpleAccount *account, const char *name, const char *default_value);
#endif

// no real reason to do this, I just think it is cool
// https://stackoverflow.com/questions/9907160/how-to-convert-enum-names-to-string-in-c
#define GENERATE_STRING(STRING) #STRING,

#define FOREACH_MESSAGE_TYPE(MESSAGE_TYPE) \
    MESSAGE_TYPE(none) \
    MESSAGE_TYPE(error) \
    MESSAGE_TYPE(log) \
    MESSAGE_TYPE(login) \
    MESSAGE_TYPE(pairing_succeeded) \
    MESSAGE_TYPE(credentials) \
    MESSAGE_TYPE(connected) \
    MESSAGE_TYPE(disconnected) \
    MESSAGE_TYPE(system) \
    MESSAGE_TYPE(name) \
    MESSAGE_TYPE(presence) \
    MESSAGE_TYPE(typing) \
    MESSAGE_TYPE(typing_stopped) \
    MESSAGE_TYPE(text) \
    MESSAGE_TYPE(attachment) \
    MESSAGE_TYPE(profile_picture) \
    MESSAGE_TYPE(group) \
    MESSAGE_TYPE(max) \

#define GENERATE_MESSAGE_ENUM(ENUM) gowhatsapp_message_type_##ENUM,

enum gowhatsapp_message_type {
    FOREACH_MESSAGE_TYPE(GENERATE_MESSAGE_ENUM)
};

#define FOREACH_ATTACHMENT_TYPE(ATTACHMENT_TYPE) \
    ATTACHMENT_TYPE(none) \
    ATTACHMENT_TYPE(image) \
    ATTACHMENT_TYPE(video) \
    ATTACHMENT_TYPE(audio) \
    ATTACHMENT_TYPE(document) \
    ATTACHMENT_TYPE(sticker) \
    ATTACHMENT_TYPE(max) \

#define GENERATE_ATTACHMENT_ENUM(ENUM) gowhatsapp_attachment_type_##ENUM,

enum gowhatsapp_attachment_type {
    FOREACH_ATTACHMENT_TYPE(GENERATE_ATTACHMENT_ENUM)
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
    char **participants; /// list of participants (for group chats)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char msgtype; /// message type – see above
    char subtype; /// loglevel, error severity, attachment type or online-state
    char isGroup; /// this is a group chat message
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
};
typedef struct gowhatsapp_message gowhatsapp_message_t;

// for feeding messages from go into purple
extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);
#endif
