#include "gowhatsapp.h"
#include "constants.h"

static
PurpleConversation *gowhatsapp_have_conversation(char *username, PurpleAccount *account) {
    gowhatsapp_ensure_buddy_in_blist(account, username, NULL);

    PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, username, account);
    if (conv == NULL) {
        conv = purple_conversation_new(PURPLE_CONV_TYPE_IM, account, username); // MEMCHECK: caller takes ownership
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
        gboolean bridge = purple_account_get_bool(gwamsg->account, GOWHATSAPP_BRIDGE_COMPATIBILITY_OPTION, FALSE);
        if (bridge) {
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
        PurpleConvChat *chat = gowhatsapp_enter_group_chat(pc, gwamsg->remoteJid, NULL);
        if (chat != NULL) {
            // participants in group chats have their senderJid supplied
            const char *who = gwamsg->senderJid;
            if (gwamsg->fromMe) {
                who = purple_account_get_username(gwamsg->account);
            }
            purple_conv_chat_write(chat, who, gwamsg->text, flags, gwamsg->timestamp);
        }
    } else {
        if (gwamsg->fromMe) {
            PurpleConversation *conv = gowhatsapp_have_conversation(gwamsg->remoteJid, gwamsg->account);
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
