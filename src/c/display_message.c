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

static void
gowhatsapp_display_group_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg, PurpleMessageFlags flags) {
    if (gwamsg->fromMe) {
        PurpleConvChat *chat = gowhatsapp_find_group_chat(
            gwamsg->remoteJid, NULL, NULL, pc
        );
        if (chat != NULL) {
            // display message sent from own account (but other device) here
            purple_conv_chat_write(chat, gwamsg->remoteJid, gwamsg->text, flags, gwamsg->timestamp
            );
        }
    } else {
        // don't create chat if not joined
        PurpleConvChat *chat = gowhatsapp_find_group_chat(
            gwamsg->remoteJid, gwamsg->senderJid, NULL, pc
        );
        if (chat != NULL) {
            // participants in group chats have their senderJid supplied
            purple_conv_chat_write(chat, gwamsg->senderJid, gwamsg->text, flags, gwamsg->timestamp
            );
        }
    }
}

void
gowhatsapp_display_text_message(PurpleConnection *pc, gowhatsapp_message_t *gwamsg)
{
    PurpleAccount *account = purple_connection_get_account(pc);
    PurpleMessageFlags flags = 0;
    gboolean check_message_date = purple_account_get_bool(account, GOWHATSAPP_TIMESTAMP_FILTERING_OPTION, FALSE);
    int message_is_new =
            gowhatsapp_append_message_id_if_not_exists(account, gwamsg->id) &&
            (!check_message_date || gowhatsapp_message_is_new_enough(account, gwamsg->timestamp));
    if (message_is_new) {
        if (gwamsg->fromMe) {
            // special handling of messages sent by self incoming from remote, addressing issue #32
            // copied from EionRobb/purple-discord/blob/master/libdiscord.c
            flags |= PURPLE_MESSAGE_SEND | PURPLE_MESSAGE_REMOTE_SEND | PURPLE_MESSAGE_DELAYED;
        } else {
            flags |= PURPLE_MESSAGE_RECV;
        }
        if (gowhatsapp_remotejid_is_group_chat(gwamsg->remoteJid)) {
            gowhatsapp_display_group_message(pc, gwamsg, flags);
        } else {
            if (gwamsg->fromMe) {
                PurpleConversation *conv = gowhatsapp_find_conversation(gwamsg->remoteJid, account);
                // display message sent from own account (but other device) here
                purple_conversation_write(conv, gwamsg->remoteJid, gwamsg->text, flags, gwamsg->timestamp);
            } else {
                // messages sometimes arrive before buddy has been
                // created... this method will be missing a display
                // name, but i don't think i ever saw one of them anyway
                gowhatsapp_ensure_buddy_in_blist(account, gwamsg->remoteJid, gwamsg->remoteJid);
                // normal mode: direct incoming message
                purple_serv_got_im(pc, gwamsg->remoteJid, gwamsg->text, flags, gwamsg->timestamp);
            }
        }
    }
}
