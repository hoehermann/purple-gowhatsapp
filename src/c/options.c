#include "gowhatsapp.h"
#include "constants.h"

// from https://github.com/ars3niy/tdlib-purple/blob/master/tdlib-purple.cpp
static GList * add_choice(GList *choices, const char *description, const char *value)
{
    PurpleKeyValuePair *kvp = g_new0(PurpleKeyValuePair, 1); // MEMCHECK: caller takes ownership
    kvp->key = g_strdup(description); // MEMCHECK: according to valgrind, this leaks
    kvp->value = g_strdup(value); // MEMCHECK: according to valgrind, this leaks
    return g_list_append(choices, kvp);
}

GList *
gowhatsapp_add_account_options(GList *account_options)
{
    PurpleAccountOption *option; // MEMCHECK: caller takes ownership
    
    option = purple_account_option_string_new( // MEMCHECK: account_options takes ownership
        "Database address",
        GOWHATSAPP_DATABASE_ADDRESS_OPTION,
        GOWHATSAPP_DATABASE_ADDRESS_DEFAULT
        );
    account_options = g_list_append(account_options, option);
    
    {
        GList *choices = NULL;
        choices = add_choice(choices, "Immediately", GOWHATSAPP_SEND_RECEIPT_CHOICE_IMMEDIATELY);
        choices = add_choice(choices, "When interacting with conversation", GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_INTERACT);
        choices = add_choice(choices, "When sending a reply", GOWHATSAPP_SEND_RECEIPT_CHOICE_ON_ANSWER);
        choices = add_choice(choices, "Never", GOWHATSAPP_SEND_RECEIPT_CHOICE_NEVER);
        option = purple_account_option_list_new( // MEMCHECK: account_options takes ownership
            "Send receipts",
            GOWHATSAPP_SEND_RECEIPT_OPTION,
            choices
        );
        account_options = g_list_append(account_options, option);
    }
    
    {
        GList *choices = NULL;
        choices = add_choice(choices, "Internal", GOWHATSAPP_ECHO_CHOICE_INTERNAL);
        choices = add_choice(choices, "On success", GOWHATSAPP_ECHO_CHOICE_ON_SUCCESS);
        choices = add_choice(choices, "Immediately", GOWHATSAPP_ECHO_CHOICE_IMMEDIATELY);
        choices = add_choice(choices, "Never", GOWHATSAPP_ECHO_CHOICE_NEVER);
        option = purple_account_option_list_new( // MEMCHECK: account_options takes ownership
            "Echo sent messages",
            GOWHATSAPP_ECHO_OPTION,
            choices
        );
        account_options = g_list_append(account_options, option);
    }
    
    option = purple_account_option_int_new(
                "Message duration in days (0 to disable expiration)",
                GOWHATSAPP_EXPIRATION_OPTION,
                0
                );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_int_new(
                "Number of messages to cache",
                GOWHATSAPP_MESSAGE_CACHE_SIZE_OPTION,
                100
                );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_int_new( // MEMCHECK: account_options takes ownership
        "QR code size (pixels)",
        GOWHATSAPP_QRCODE_SIZE_OPTION,
        256
        );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_int_new(
        "Maximum linked file-size (MB)",
        GOWHATSAPP_EMBED_MAX_FILE_SIZE_OPTION,
        0
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_string_new( // MEMCHECK: account_options takes ownership
        "Incoming file message",
        GOWHATSAPP_ATTACHMENT_MESSAGE_OPTION,
        GOWHATSAPP_ATTACHMENT_MESSAGE_DEFAULT
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_string_new( // MEMCHECK: account_options takes ownership
        "Linked file trusted URL regex",
        GOWHATSAPP_TRUSTED_URL_REGEX_OPTION,
        GOWHATSAPP_TRUSTED_URL_REGEX_DEFAULT
        );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Inline images",
        GOWHATSAPP_INLINE_IMAGES_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Inline stickers (needs webp loader)",
        GOWHATSAPP_INLINE_STICKERS_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Treat group as the origin of files",
        GOWHATSAPP_GROUP_IS_FILE_ORIGIN_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Display offline contacts as away",
        GOWHATSAPP_FAKE_ONLINE_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Automatically add contacts",
        GOWHATSAPP_FETCH_CONTACTS_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Download user profile pictures",
        GOWHATSAPP_GET_ICONS_OPTION,
        FALSE
        );
    account_options = g_list_append(account_options, option);

    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Fetch conversation history (experimental)",
        GOWHATSAPP_FETCH_HISTORY_OPTION,
        FALSE
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Automatically join all chats",
        GOWHATSAPP_AUTO_JOIN_CHAT_OPTION,
        FALSE
        );
    account_options = g_list_append(account_options, option);    
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Ignore status broadcasts",
        GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION,
        TRUE
        );
    account_options = g_list_append(account_options, option);
    
    option = purple_account_option_bool_new( // MEMCHECK: account_options takes ownership
        "Protocol bridge compatibility mode",
        GOWHATSAPP_BRIDGE_COMPATIBILITY_OPTION,
        FALSE
        );
    account_options = g_list_append(account_options, option);

    return account_options;
}
