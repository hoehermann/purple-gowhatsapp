#ifdef CONSTANTS_H
#error "constants.h may only be included once."
#endif
#define CONSTANTS_H

#ifdef CGO
#define CHARCONSTANT(key, value) extern const char * key
#else
#define CHARCONSTANT(key, value) const char * key = value
#endif

/*
 * This defines string constants in such a way both cgo and gcc can access them.
 */

/*
 * String keys for user-configurable settings.
 */
CHARCONSTANT(GOWHATSAPP_RESTORE_SESSION_OPTION, "restore-session");
CHARCONSTANT(GOWHATSAPP_FETCH_CONTACTS_OPTION, "pull-contacts"); // TODO: rename to "fetch-contacts"
CHARCONSTANT(GOWHATSAPP_MARK_READ_OPTION, "mark-read");
CHARCONSTANT(GOWHATSAPP_GET_ICONS_OPTION, "get-icons");
CHARCONSTANT(GOWHATSAPP_PLAIN_TEXT_LOGIN, "plain-text-login");
CHARCONSTANT(GOWHATSAPP_FAKE_ONLINE_OPTION, "fake-online");
CHARCONSTANT(GOWHATSAPP_MESSAGE_ID_STORE_SIZE_OPTION, "message-id-store-size");
CHARCONSTANT(GOWHATSAPP_TIMESTAMP_FILTERING_OPTION, "message-timestamp-filter");
CHARCONSTANT(GOWHATSAPP_SYSTEM_MESSAGES_ARE_ORDINARY_MESSAGES_OPTION, "system-messages-are-ordinary-messages");
CHARCONSTANT(GOWHATSAPP_DOWNLOAD_ATTACHMENTS_OPTION, "download-attachments");
CHARCONSTANT(GOWHATSAPP_DOWNLOAD_TRY_ONLY_ONCE_OPTION, "download-try-only-once");
CHARCONSTANT(GOWHATSAPP_INLINE_IMAGES_OPTION, "inline-images");

/*
 * String key for administrative data.
 */
CHARCONSTANT(GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, "last-new-messages-timestamp");

/*
 * String keys for log-in data.
 */
CHARCONSTANT(GOWHATSAPP_SESSION_CLIENDID_KEY, "clientid");
CHARCONSTANT(GOWHATSAPP_SESSION_CLIENTTOKEN_KEY, "clientToken");
CHARCONSTANT(GOWHATSAPP_SESSION_SERVERTOKEN_KEY, "serverToken");
CHARCONSTANT(GOWHATSAPP_SESSION_ENCKEY_KEY, "encKey");
CHARCONSTANT(GOWHATSAPP_SESSION_MACKEY_KEY, "macKey");
CHARCONSTANT(GOWHATSAPP_SESSION_WID_KEY, "wid");
