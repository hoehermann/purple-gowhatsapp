#include "gowhatsapp.h"
#include "constants.h"

void gowhatsapp_display_text_message(
    PurpleAccount *account, 
    const gchar * senderJid,
    const gchar * remoteJid,
    const gchar * text,
    const time_t timestamp,
    const gboolean isGroup,
    const gboolean isOutgoing,
    const gchar * name,
    PurpleMessageFlags flags,
    const gchar * messageId,
    const gboolean escape
) {
    g_return_if_fail(account != NULL);
    
    PurpleConnection * connection = purple_account_get_connection(account);
    
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

    // WhatsApp is a plain-text protocol, but Pidgin expects HTML
    gchar * escaped_text = NULL;
    if (escape) { // sometimes, text is already escaped
        gchar * html = purple_markup_escape_text(text, -1); // converts to HTML except the line breakes
        escaped_text = purple_strdup_withhtml(html); // converts newline characters to HTML br tags
        g_free(html);
    } else {
        escaped_text = g_strdup(text); // MEMCHECK: released here (see below)
    }

    // add message ID to visible text
    // for https://github.com/Juliaria08 in https://github.com/hoehermann/purple-gowhatsapp/issues/206
    gchar * text_with_id = NULL;
    if (purple_account_get_bool(account, GOWHATSAPP_DISPLAY_MESSAGE_ID_OPTION, FALSE) && messageId != NULL) {
        text_with_id = g_strdup_printf("%s <span lang=\"id\">%s</span>", escaped_text, messageId); // MEMCHECK: released here (see below)
    } else {
        text_with_id = g_strdup(escaped_text); // MEMCHECK: released here (see below)
    }

    g_free(escaped_text);
    
    if (isGroup) {
        gowhatsapp_enter_group_chat(connection, remoteJid, NULL);
        purple_serv_got_chat_in(connection, g_str_hash(remoteJid), senderJid, flags, text_with_id, timestamp);
    } else {
        if (flags & PURPLE_MESSAGE_SEND) {
            // display message sent from own account (other device as well as local echo)
            // cannot use purple_serv_got_im since it sets the flag PURPLE_MESSAGE_RECV
            PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, remoteJid, account);
            if (conv == NULL) {
                conv = purple_conversation_new(PURPLE_CONV_TYPE_IM, account, remoteJid); // MEMCHECK: caller takes ownership
            }
            purple_conv_im_write(purple_conversation_get_im_data(conv), remoteJid, text_with_id, flags, timestamp);
        } else {
            if (purple_account_get_bool(account, GOWHATSAPP_UPDATE_BUDDY_ON_MESSAGE_OPTION, TRUE)) {
                gowhatsapp_ensure_buddy_in_blist(account, remoteJid, name);
            }
            purple_serv_got_im(connection, remoteJid, text_with_id, flags, timestamp);
        }
    }
    
    g_free(text_with_id);
}
