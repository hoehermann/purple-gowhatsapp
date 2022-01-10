#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

static gulong conversation_updated_signal = 0;

// from https://github.com/EionRobb/skype4pidgin/blob/master/skypeweb/skypeweb_messages.c
static void
conversation_updated(PurpleConversation *conv, PurpleConversationUpdateType type)
{
    if (type == PURPLE_CONVERSATION_UPDATE_UNSEEN) {
        // TODO: find out why this is fired way too often (e.g. typing… notifications from remote)
        PurpleAccount *account = purple_conversation_get_account(conv);
        char *who = (char *)purple_conversation_get_name(conv); // cgo does not suport const
        gowhatsapp_go_mark_read_conversation(account, who); // this does all other checks (e.g. connection state)
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
