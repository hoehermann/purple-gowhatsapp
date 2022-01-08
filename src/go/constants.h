#ifndef CONSTANTS_H
#define CONSTANTS_H

/*
 * Problem: CGO reads object files to determine the type of constants.
 * It does not understand preprocessor defines.
 * Constants cannot be defined in the header since they must be defined exactly once.
 * So they ar declared as symbols to be resolved externally.
 * Then constants.c defines them and the external linker puts it all together.
 * This way, both cgo and gcc can access the values.
 */
#ifndef CHARCONSTANT
#define CHARCONSTANT(key, value) extern const char * key
#endif

#define GOWHATSAPP_NAME "whatsmeow"  // name to refer to this plug-in (in logs)
#define GOWHATSAPP_PRPL_ID "prpl-hehoe-whatsmeow"

/*
 * String keys for user-configurable settings.
 */
CHARCONSTANT(GOWHATSAPP_RESTORE_SESSION_OPTION, "restore-session");
CHARCONSTANT(GOWHATSAPP_FETCH_CONTACTS_OPTION, "fetch-contacts");
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
CHARCONSTANT(GOWHATSAPP_MAX_HISTORY_SECONDS_OPTION, "max-history-seconds");

/*
 * String key for administrative data.
 */
CHARCONSTANT(GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, "last-new-messages-timestamp");
CHARCONSTANT(GOWHATSAPP_RECEIVED_MESSAGES_ID_KEY, "received-messages-ids");

#endif
