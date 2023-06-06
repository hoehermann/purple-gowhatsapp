#ifndef CONSTANTS_H
#define CONSTANTS_H

/*
 * Problem: CGO reads object files to determine the type of constants.
 * It does not understand preprocessor defines.
 * Constants cannot be defined in the header since they must be defined exactly once.
 * So they are declared as symbols to be resolved externally.
 * Then constants.c defines them and the external linker puts it all together.
 * This way, both cgo and gcc can access the values.
 */
#ifndef CHARCONSTANT
#define CHARCONSTANT(key, value) extern const char * key
#endif

/*
 * String keys for user-configurable settings.
 */
CHARCONSTANT(GOWHATSAPP_FETCH_CONTACTS_OPTION, "fetch-contacts");
CHARCONSTANT(GOWHATSAPP_FAKE_ONLINE_OPTION, "fake-online");
CHARCONSTANT(GOWHATSAPP_QRCODE_SIZE_OPTION, "qrcode-size");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_OPTION, "send-receipt");
CHARCONSTANT(GOWHATSAPP_GET_ICONS_OPTION, "get-icons");
CHARCONSTANT(GOWHATSAPP_EXPIRATION_OPTION, "message-expiration");
CHARCONSTANT(GOWHATSAPP_INLINE_IMAGES_OPTION, "inline-images");
CHARCONSTANT(GOWHATSAPP_INLINE_STICKERS_OPTION, "inline-stickers");
CHARCONSTANT(GOWHATSAPP_GROUP_IS_FILE_ORIGIN_OPTION, "group-is-file-origin");
CHARCONSTANT(GOWHATSAPP_AUTO_JOIN_CHAT_OPTION, "autojoin-chats");
CHARCONSTANT(GOWHATSAPP_BRIDGE_COMPATIBILITY_OPTION, "bridge-compatibility");
CHARCONSTANT(GOWHATSAPP_ECHO_OPTION, "echo-sent-messages");
CHARCONSTANT(GOWHATSAPP_MESSAGE_CACHE_SIZE_OPTION, "message-cache-size");
CHARCONSTANT(GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, "ignore-status-broadcast");
CHARCONSTANT(GOWHATSAPP_FETCH_HISTORY_OPTION, "fetch-history");
CHARCONSTANT(GOWHATSAPP_EMBED_MAX_FILE_SIZE_OPTION, "embed-max-file-size");
CHARCONSTANT(GOWHATSAPP_TRUSTED_URL_REGEX_OPTION, "trusted-url-regex");
CHARCONSTANT(GOWHATSAPP_TRUSTED_URL_REGEX_DEFAULT, "");
CHARCONSTANT(GOWHATSAPP_DATABASE_ADDRESS_OPTION, "database-address");
CHARCONSTANT(GOWHATSAPP_DATABASE_ADDRESS_DEFAULT, "file:$purple_user_dir/whatsmeow.db?_foreign_keys=on&_busy_timeout=3000");
CHARCONSTANT(GOWHATSAPP_ATTACHMENT_MESSAGE_OPTION, "attachment-message");
CHARCONSTANT(GOWHATSAPP_ATTACHMENT_MESSAGE_DEFAULT, "Preparing to store \"%s\" sent by %s...");

/*
 * Choices for GOWHATSAPP_SEND_RECEIPT_OPTION.
 */
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY, "immediately");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_INTERACT, "on-interact");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_ANSWER, "on-answer");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_NEVER, "never");

/*
 * Choices for GOWHATSAPP_ECHO_OPTION.
 */
CHARCONSTANT(GOWHATSAPP_ECHO_CHOICE_ON_SUCCESS, "on-success");
CHARCONSTANT(GOWHATSAPP_ECHO_CHOICE_INTERNAL, "internal");
CHARCONSTANT(GOWHATSAPP_ECHO_CHOICE_IMMEDIATELY, "immediately");
CHARCONSTANT(GOWHATSAPP_ECHO_CHOICE_NEVER, "never");

/*
 * Keys for account data.
 */
CHARCONSTANT(GOWHATSAPP_CREDENTIALS_KEY, "credentials");
CHARCONSTANT(GOWHATSAPP_PRESENCE_OVERRIDE_KEY, "presence-override");
#endif
