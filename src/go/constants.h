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

#endif
