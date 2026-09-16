#include "gowhatsapp.h"
#include "constants.h"
#include "libwhatsmeow.h"

static gulong conversation_created_signal = 0;
static int requests_suppressed = 0;

/*
 * Displaying a message brings its conversation into being if it did not exist,
 * which would otherwise read as "the user just opened this" and pull a heap of
 * old messages in underneath the new one. Requests are held during a display
 * for that reason, and it is the same guard that keeps the history we fetch
 * from asking for more history.
 */
void
gowhatsapp_history_suppress_requests(gboolean suppress) {
    if (suppress) {
        requests_suppressed++;
    } else if (requests_suppressed > 0) {
        requests_suppressed--;
    }
}

/*
 * A message the primary device sent back in answer to a history request.
 *
 * It travels the ordinary display path so it carries its message id like any
 * other message, but marked PURPLE_MESSAGE_DELAYED: a front-end can then show
 * it as the old news it is rather than announcing it.
 */
void
gowhatsapp_display_history_message(gowhatsapp_message_t *gwamsg) {
    gowhatsapp_display_text_message(
        gwamsg->account, gwamsg->senderJid, gwamsg->remoteJid, gwamsg->text,
        gwamsg->timestamp, gwamsg->isGroup, FALSE, gwamsg->name,
        PURPLE_MESSAGE_DELAYED, gwamsg->messageId, TRUE
    );
}

/*
 * A conversation just came into being, either because the user opened it or
 * because a message arrived. Ask the primary device for what came before.
 *
 * The Go part holds off requests it has answered recently, so a conversation
 * being re-created (and the messages fetched here creating it again) does not
 * turn into a loop.
 */
static void conversation_created(PurpleConversation *conv, gpointer data) {
    PurpleAccount *account = purple_conversation_get_account(conv);
    if (requests_suppressed) {
        // a message brought this conversation into being, not the user
        return;
    }
    if (!purple_strequal(purple_account_get_protocol_id(account), GOWHATSAPP_PRPL_ID)) {
        return;
    }
    int count = purple_account_get_int(account, GOWHATSAPP_FETCH_HISTORY_OPTION, 0);
    if (count <= 0) {
        return;
    }
    PurpleConnection *pc = purple_account_get_connection(account);
    WhatsappProtocolData *wpd = pc ? (WhatsappProtocolData *)purple_connection_get_protocol_data(pc) : NULL;
    if (wpd == NULL || wpd->history_requests_from == 0 || time(NULL) < wpd->history_requests_from) {
        // still settling in after connecting, where conversations appear by themselves
        return;
    }
    char *who = (char *)purple_conversation_get_name(conv); // cgo does not suport const
    gowhatsapp_go_request_history(account, who, count); // this does all other checks (e.g. connection state)
}

/*
 * This registers a call-back: This prpl shall be notified every time a
 * conversation comes into being.
 */
void
gowhatsapp_history_init(PurpleConnection *pc) {
    if (!conversation_created_signal) {
        conversation_created_signal = purple_signal_connect(
            purple_conversations_get_handle(),
            "conversation-created",
            purple_connection_get_protocol(pc),
            PURPLE_CALLBACK(conversation_created),
            NULL
        );
    }
}
