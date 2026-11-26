#pragma once
// The C side of the Facebook Messenger protocol (prpl-gometa, built with -tags gometa / -DGOMETA=ON).
// All identifiers are gometa_-prefixed so they do not collide with the WhatsApp (gowhatsapp_) code
// in the same shared object and Go package.
#include <purple.h>
#include <time.h>

#define GOMETA_PLUGIN_ID      "prpl-gometa"
#define GOMETA_PLUGIN_NAME    "Facebook (E2EE)"
#define GOMETA_PLUGIN_SUMMARY "Facebook Messenger, end-to-end encrypted chats included"
#define GOMETA_AUTHOR         "Herman van Hazendonk <github.com@herrie.org>"

enum gometa_message_type {
    gometa_message_type_error,        // text = error; fatal = connection-fatal
    gometa_message_type_connected,
    gometa_message_type_disconnected,
    gometa_message_type_text,         // a message: conv, who, name, text (markup), id, isGroup, isOutgoing
    gometa_message_type_buddy,        // add/update a contact: who=fbid, name=display name
    gometa_message_type_chat,         // register a group chat: conv=threadKey, name=thread name
    gometa_message_type_presence,     // buddy presence: who=fbid, isOutgoing=1 -> available else offline
    gometa_message_type_system,       // a line nobody sent (a group's news): conv, text (markup), isGroup
    gometa_message_type_members,      // who is in a group besides us: conv, text = "\x1f"-joined fbids
    gometa_message_type_typing,       // a 1:1 chat partner typing: who, isOutgoing=1 -> typing else stopped
    gometa_message_type_attachment,   // a received file: as text, plus path and mimetype; text = caption
};

typedef struct gometa_message {
    PurpleAccount *account;
    char *who;   // for text: sender fbid; for buddy: contact fbid
    char *conv;  // for text: conversation/thread key (1:1 = other user, group = thread); for chat: thread key
    char *name;  // display name (sender name, contact name, or group name)
    char *text;  // message body / error text
    char *id;    // message id (optional)
    char *path;     // attachment: the local file
    char *mimetype; // attachment: its type (optional)
    time_t timestamp;
    char msgtype;
    char isGroup;
    char isOutgoing;
    char fatal;
} gometa_message_t;

// Go -> purple; the C side owns and free()s the strings and handles the message on the glib main thread.
extern void gometa_process_message(gometa_message_t msg);

// Interactive login prompt (2FA code), asked on the main thread. The answer comes back via the Go
// export gometa_go_submit_input(account, value).
extern void gometa_request_input(PurpleAccount *account, const char *prompt);

// Account settings (used to cache the login session). Returned string is malloc'd;
// the Go caller frees it. Returns NULL if unset.
extern char *gometa_get_setting(PurpleAccount *account, const char *key);
extern void gometa_set_setting(PurpleAccount *account, const char *key, const char *value);

// The buddy-list group Facebook contacts and chats are put in.
extern PurpleGroup *gometa_blist_group(void);
