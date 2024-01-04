#include "gowhatsapp.h"
#include "constants.h"

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

    if (purple_strequal(purple_account_get_username(gwamsg->account), gwamsg->senderJid)) {
        flags |= PURPLE_MESSAGE_SEND;
        // Note: For outgoing messages (no matter if local echo or sent by other device),
        // PURPLE_MESSAGE_SEND must be set due to how purple_conversation_write is implemented
        if (!gwamsg->isOutgoing) {
            // special handling of messages sent by self incoming from remote, addressing issue #32
            // adjusted for Spectrum, see issue #130
            flags |= PURPLE_MESSAGE_REMOTE_SEND;
        }
    } else {
        flags |= PURPLE_MESSAGE_RECV;
    }
    
    // WhatsApp is a plain-text protocol, but Pidgin expects HTML
    gchar * text = purple_markup_escape_text(gwamsg->text, -1);
    if (gwamsg->isGroup) {
        PurpleConversation *conv = gowhatsapp_enter_group_chat(pc, gwamsg->remoteJid, NULL);
        if (conv != NULL) {
            purple_serv_got_chat_in(pc, g_str_hash(gwamsg->remoteJid), gwamsg->senderJid, flags, text, gwamsg->timestamp);
        }
    } else {
        if (flags & PURPLE_MESSAGE_SEND) {
            // display message sent from own account (other device as well as local echo)
            // cannot use purple_serv_got_im since it sets the flag PURPLE_MESSAGE_RECV
            PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, gwamsg->remoteJid, gwamsg->account);
            if (conv == NULL) {
                conv = purple_conversation_new(PURPLE_CONV_TYPE_IM, gwamsg->account, gwamsg->remoteJid); // MEMCHECK: caller takes ownership
            }
            purple_conv_im_write(purple_conversation_get_im_data(conv), gwamsg->remoteJid, text, flags, gwamsg->timestamp);
        } else {
            // messages sometimes arrive before buddy has been created
            // a buddy created here may be missing a display name,
            // but i don't think i ever saw one of them anyway
            gowhatsapp_ensure_buddy_in_blist(gwamsg->account, gwamsg->remoteJid, gwamsg->name);
            purple_serv_got_im(pc, gwamsg->remoteJid, text, flags, gwamsg->timestamp);
        }
    }
    g_free(text);
}
