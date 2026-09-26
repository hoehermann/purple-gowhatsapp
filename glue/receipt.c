#include "gowhatsapp.h"
#include "libwhatsmeow.h"

static gulong conversation_updated_signal = 0;

/*
 * This handler is called every time a conversation is updated in any possible way.
 * This is implemented to work with Pidgin specifically.
 * 
 * Inspired by https://github.com/EionRobb/skype4pidgin/blob/master/skypeweb/skypeweb_messages.c
 * and adapted in regard to pidgin/gtkconv.c.
 */
static void conversation_updated(PurpleConversation *conv, PurpleConversationUpdateType type) {
    if (type == PURPLE_CONVERSATION_UPDATE_UNSEEN) {
        // the conversation has been updated and is currently not the active window
        int unseen_count = GPOINTER_TO_INT(purple_conversation_get_data(conv, "unseen-count")); // specific to Pidgin
        int previous_unseen_count = GPOINTER_TO_INT(purple_conversation_get_data(conv, "previous-unseen-count")); // specific to this plug-in
        if (0 == unseen_count && previous_unseen_count > 0) {
            // in the past, there have been unseen events, now there are none
            // this probably means the conversation has been shown to the user
            // → the conversation shall be marked as "read"
            PurpleAccount *account = purple_conversation_get_account(conv);
            char *who = (char *)purple_conversation_get_name(conv); // cgo does not suport const
            gowhatsapp_go_mark_read_conversation(account, who); // this does all other checks (e.g. connection state)
        }
        previous_unseen_count = unseen_count;
        purple_conversation_set_data(conv, "previous-unseen-count", GINT_TO_POINTER(previous_unseen_count));
    }
}

/*
 * This registers a call-back: This prpl shall be notified every time a conversation is updated.
 */
/*
 * An incoming receipt: one station of one message's life (sent, delivered,
 * read, read_self – see gowhatsapp_receipt_type in bridge.h).
 *
 * libpurple has no native concept of receipts, so this is offered two ways:
 * the GOWHATSAPP_SIGNAL_RECEIPT signal always fires, with a string-to-string
 * hash table of the details, for front-ends and plug-ins that can draw ticks
 * themselves; and behind the receipt-display account option (off by default)
 * the read receipt is shown as a system message in the conversation window.
 */
void
gowhatsapp_handle_receipt(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    static const char *receipt_type_names[] = { FOREACH_RECEIPT_TYPE(GENERATE_STRING) };
    if (gwamsg->subtype < 0 || gwamsg->subtype >= gowhatsapp_receipt_type_max) {
        purple_debug_info(GOWHATSAPP_NAME, "received invalid receipt type %d.\n", gwamsg->subtype);
        return;
    }

    {
        gchar *timestamp = g_strdup_printf("%ld", (long)gwamsg->timestamp);
        GHashTable *details = g_hash_table_new(g_str_hash, g_str_equal); // MEMCHECK: values are borrowed, receivers must copy
        g_hash_table_insert(details, "chat", gwamsg->remoteJid);
        g_hash_table_insert(details, "sender", gwamsg->senderJid);
        g_hash_table_insert(details, "id", gwamsg->messageId);
        g_hash_table_insert(details, "type", (gpointer)receipt_type_names[(int)gwamsg->subtype]);
        g_hash_table_insert(details, "isGroup", gwamsg->isGroup ? "1" : "0");
        g_hash_table_insert(details, "timestamp", timestamp);
        purple_signal_emit(purple_connection_get_prpl(pc), GOWHATSAPP_SIGNAL_RECEIPT, pc, details);
        g_hash_table_destroy(details);
        g_free(timestamp);
    }

    if (gwamsg->subtype == gowhatsapp_receipt_type_read) {
        const char *setting = purple_account_get_string(gwamsg->account, GOWHATSAPP_RECEIPT_DISPLAY_OPTION, GOWHATSAPP_RECEIPT_DISPLAY_CHOICE_NONE);
        if (purple_strequal(setting, GOWHATSAPP_RECEIPT_DISPLAY_CHOICE_TEXT)) {
            PurpleConversation *conv = purple_find_conversation_with_account(
                gwamsg->isGroup ? PURPLE_CONV_TYPE_CHAT : PURPLE_CONV_TYPE_IM,
                gwamsg->remoteJid, gwamsg->account);
            if (conv != NULL) {
                purple_conversation_write(conv, NULL, "✓✓ Your message has been read.",
                                          PURPLE_MESSAGE_SYSTEM | PURPLE_MESSAGE_NO_LOG, time(NULL));
            }
        }
    }
}

void
gowhatsapp_receipts_init(PurpleConnection *pc) {
    if (!conversation_updated_signal) {
        conversation_updated_signal = purple_signal_connect(
            purple_conversations_get_handle(), 
            "conversation-updated", 
            purple_connection_get_protocol(pc), 
            PURPLE_CALLBACK(conversation_updated), 
            NULL
        );
    }
}
