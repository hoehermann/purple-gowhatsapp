#pragma once

#include <purple.h> // for PurpleAccount, PurpleXfer
#include <time.h> // for time_t
#include <stdint.h> // for uint64_t and uintptr_t

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
    char *messageId; /// message ID
    char *text; /// the message payload (interpretation depends on type)
    char *pairing_code; /// 6-character pairing code
    char *pairing_qrdata; /// the pairing QR-code raw data
    char *pairing_qrterminal; /// graphical QR for printing on a terminal
    char *name; /// remote user's name (chosen by them)
    void *blob; /// binary payload (used for image of pairing QR code PNG and profile pictures)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char **participants; /// list of participants (for group chats)
    char msgtype; /// message type – see above
    char subtype; /// loglevel, error severity, attachment type or online-state
    char isGroup; /// this is a group chat message
    char isOutgoing; /// this is an outgoing message (echo sent from this instance, indicating success on sending – this is *not* ifFromMe)
    // everything related to attachments:
    char *filename;
    char *extension; /// including the dot
    char *mimetype;
    char *hash_hex;
    uint64_t filesize;
    uintptr_t download_handle; // a reference to the structure needed for the actual download
};
typedef struct gowhatsapp_message gowhatsapp_message_t;

// for feeding messages from go into purple
extern void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg);

// for processing the messages in purple
extern gboolean process_message_bridge(gpointer data);

// for checking if the account exists before trying to read settings from it
extern int gowhatsapp_account_exists(PurpleAccount *account);

// for releasing the memory of a message struct
void gowhatsapp_free_message(gowhatsapp_message_t *gwamsg);

// for looking up alias locally
extern const char * gowhatsapp_blist_get_alias(PurpleAccount *account, const char *who);