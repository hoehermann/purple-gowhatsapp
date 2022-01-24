#ifndef CONSTANTS_H
#define CONSTANTS_H

#define GOWHATSAPP_NAME "whatsmeow"  // name to refer to this plug-in (in logs)
#define GOWHATSAPP_PRPL_ID "prpl-hehoe-whatsmeow"

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

/*
 * String keys for user-configurable settings.
 */
CHARCONSTANT(GOWHATSAPP_FETCH_CONTACTS_OPTION, "fetch-contacts");
CHARCONSTANT(GOWHATSAPP_FAKE_ONLINE_OPTION, "fake-online");
CHARCONSTANT(GOWHATSAPP_QRCODE_SIZE_OPTION, "qrcode-size");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_OPTION, "send-receipt");
CHARCONSTANT(GOWHATSAPP_GET_ICONS_OPTION, "get-icons");
CHARCONSTANT(GOWHATSAPP_SPECTRUM_COMPATIBILITY_OPTION, "spectrum-compatibility");
CHARCONSTANT(GOWHATSAPP_DATABASE_ADDRESS_OPTION, "database-address");

CHARCONSTANT(GOWHATSAPP_DATABASE_ADDRESS_DEFAULT, "file:$purple_user_dir/whatsmeow.db?_foreign_keys=on&_busy_timeout=3000");
/*
 * Choices for GOWHATSAPP_SEND_RECEIPT_OPTION.
 */
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY, "immediately");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_INTERACT, "on-interact");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_ANSWER, "on-answer");
CHARCONSTANT(GOWHATSAPP_SEND_RECEIPT_CHOICE_NEVER, "never");

/*
 * Keys for account data.
 */
CHARCONSTANT(GOWHATSAPP_CREDENTIALS_KEY, "credentials");
#endif
