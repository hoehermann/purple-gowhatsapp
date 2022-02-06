#include "gowhatsapp.h"
#include "constants.h"

static
PurpleConversation *gowhatsapp_find_conversation(char *username, PurpleAccount *account) {
    gowhatsapp_ensure_buddy_in_blist(account, username, NULL);

    PurpleIMConversation *imconv = purple_conversations_find_im_with_account(username, account);
    if (imconv == NULL) {
        imconv = purple_im_conversation_new(account, username);
    }
    PurpleConversation *conv = PURPLE_CONVERSATION(imconv);
    if (conv == NULL) {
        imconv = purple_conversations_find_im_with_account(username, account);
        conv = PURPLE_CONVERSATION(imconv);
    }
    return conv;
}

void
gowhatsapp_display_text_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg, PurpleMessageFlags flags)
{
    g_return_if_fail(pc != NULL);
    
    if (flags & PURPLE_MESSAGE_SYSTEM) {
        if (gwamsg->senderJid == NULL) {
            gwamsg->senderJid = g_strdup("system"); // g_strdup needed since senderJid is freed by caller
        }
        gboolean spectrum = purple_account_get_bool(gwamsg->account, GOWHATSAPP_SPECTRUM_COMPATIBILITY_OPTION, FALSE);
        if (spectrum) {
            // spectrum ignores system messages: strip the system flag
            flags &= ~PURPLE_MESSAGE_SYSTEM;
        } else {
            // normal Procedure: keep system flag, do not log message
            flags |= PURPLE_MESSAGE_NO_LOG;
        }
    }

    if (gwamsg->fromMe) {
        // special handling of messages sent by self incoming from remote, addressing issue #32
        // copied from EionRobb/purple-discord/blob/master/libdiscord.c
        flags |= PURPLE_MESSAGE_SEND | PURPLE_MESSAGE_REMOTE_SEND | PURPLE_MESSAGE_DELAYED;
    } else {
        flags |= PURPLE_MESSAGE_RECV;
    }
    if (gwamsg->isGroup) {
        PurpleConvChat *chat = gowhatsapp_enter_group_chat(pc, gwamsg->remoteJid);
        if (chat != NULL) {
            // participants in group chats have their senderJid supplied
            // TODO: test with fromMe
            purple_conv_chat_write(chat, gwamsg->senderJid, gwamsg->text, flags, gwamsg->timestamp);
        }
    } else {
        if (gwamsg->fromMe) {
            PurpleConversation *conv = gowhatsapp_find_conversation(gwamsg->remoteJid, gwamsg->account);
            // display message sent from own account (but other device) here
            purple_conversation_write(conv, gwamsg->remoteJid, gwamsg->text, flags, gwamsg->timestamp);
        } else {
            // messages sometimes arrive before buddy has been
            // created... this method will be missing a display
            // name, but i don't think i ever saw one of them anyway
            gowhatsapp_ensure_buddy_in_blist(gwamsg->account, gwamsg->remoteJid, gwamsg->name);
            // normal mode: direct incoming message
            purple_serv_got_im(pc, gwamsg->remoteJid, gwamsg->text, flags, gwamsg->timestamp);
        }
    }
}
