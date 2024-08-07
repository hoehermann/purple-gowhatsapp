#include "gowhatsapp.h"
#include "constants.h"

void gowhatsapp_display_text_message(PurpleConnection *connection, gowhatsapp_message_t *gwamsg, PurpleMessageFlags flags) {
    g_return_if_fail(connection != NULL);
    // WhatsApp is a plain-text protocol, but Pidgin expects HTML
    gchar * text = purple_markup_escape_text(gwamsg->text, -1);
    PurpleAccount *account = purple_connection_get_account(connection);
    if (purple_account_get_bool(account, GOWHATSAPP_DISPLAY_MESSAGE_ID_OPTION, FALSE)) {
        // for https://github.com/Juliaria08
        gchar * text_with_id = g_strdup_printf("%s <span lang=\"id\">%s</span>", text, gwamsg->messageId);
        g_free(text);
        text = text_with_id;
    }
    gowhatsapp_display_message_common(connection, gwamsg->senderJid, gwamsg->remoteJid, text, gwamsg->timestamp, gwamsg->isGroup, gwamsg->isOutgoing, gwamsg->name, flags);
    g_free(text);
}

void gowhatsapp_display_message_common(
    PurpleConnection *pc, 
    const gchar * senderJid,
    const gchar * remoteJid,
    const gchar * text,
    const time_t timestamp,
    const gboolean isGroup,
    const gboolean isOutgoing,
    const gchar * name,
    PurpleMessageFlags flags
) {
    g_return_if_fail(pc != NULL);
    
    PurpleAccount * account = purple_connection_get_account(pc);
    
    if (flags & PURPLE_MESSAGE_SYSTEM) {
        if (senderJid == NULL) {
            senderJid = g_strdup("system"); // g_strdup needed since senderJid is freed by caller
        }
        gboolean bridge = purple_account_get_bool(account, GOWHATSAPP_BRIDGE_COMPATIBILITY_OPTION, FALSE);
        if (bridge) {
            // spectrum ignores system messages: strip the system flag
            flags &= ~PURPLE_MESSAGE_SYSTEM;
        } else {
            // normal Procedure: keep system flag, do not log message
            flags |= PURPLE_MESSAGE_NO_LOG;
        }
    }

    if (purple_strequal(purple_account_get_username(account), senderJid)) {
        flags |= PURPLE_MESSAGE_SEND;
        // Note: For outgoing messages (no matter if local echo or sent by other device),
        // PURPLE_MESSAGE_SEND must be set due to how purple_conversation_write is implemented
        if (!isOutgoing) {
            // special handling of messages sent by self incoming from remote, addressing issue #32
            // adjusted for Spectrum, see issue #130
            flags |= PURPLE_MESSAGE_REMOTE_SEND;
        }
    } else {
        flags |= PURPLE_MESSAGE_RECV;
    }
    
    if (isGroup) {
        gowhatsapp_enter_group_chat(pc, remoteJid, NULL);
        purple_serv_got_chat_in(pc, g_str_hash(remoteJid), senderJid, flags, text, timestamp);
    } else {
        if (flags & PURPLE_MESSAGE_SEND) {
            // display message sent from own account (other device as well as local echo)
            // cannot use purple_serv_got_im since it sets the flag PURPLE_MESSAGE_RECV
            PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, remoteJid, account);
            if (conv == NULL) {
                conv = purple_conversation_new(PURPLE_CONV_TYPE_IM, account, remoteJid); // MEMCHECK: caller takes ownership
            }
            purple_conv_im_write(purple_conversation_get_im_data(conv), remoteJid, text, flags, timestamp);
        } else {
            // messages sometimes arrive before buddy has been created
            // a buddy created here may be missing a display name,
            // but i don't think i ever saw one of them anyway
            gowhatsapp_ensure_buddy_in_blist(account, remoteJid, name);
            purple_serv_got_im(pc, remoteJid, text, flags, timestamp);
        }
    }
}
