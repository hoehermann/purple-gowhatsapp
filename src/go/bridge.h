#ifndef _BRIDGE_H_
#define _BRIDGE_H_

#include <stdint.h> // for int64_t
#include <time.h> // for time_t

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_connected,
    gowhatsapp_message_type_typing,
    gowhatsapp_message_type_typing_stopped
};

// Structure to communicate go → purple.
// This holds all data for incoming messages, error messages, login data, etc.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    char *username; /// username identifying the account
    char *id; /// message id
    char *remoteJid; /// conversation identifier (may be a single contact or a group)
    char *senderJid; /// message author's identifier (useful in group chats)
    char *text; /// the message payload (interpretation depends on type)
    char *quote; /// quote (in case of replies)
    char *alias; /// remote users name (chosen by them)
    void *blob; /// binary payload (used for inlining images)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char msgtype; /// message type – see above
    char isGroup; /// this is a group chat message
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
};
typedef struct gowhatsapp_message gowhatsapp_message_t;
// TODO: remove ID (no longer used)

#endif
