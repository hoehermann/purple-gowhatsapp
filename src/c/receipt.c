#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

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
