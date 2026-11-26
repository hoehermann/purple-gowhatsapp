//go:build gometa

// gometabridge.c — the Go->purple boundary of the Facebook protocol. Like the WhatsApp bridge
// (bridge.c), each message is copied and handled on the glib main thread (purple_timeout_add).
#include "gometabridge.h"
#include <stdlib.h> // free/malloc
#include <string.h> // strdup
#include <time.h>   // time

// The cgo exports this file calls. libwhatsmeow.h, which declares them, does not exist yet when cgo
// compiles this file. The signatures must match the //export functions.
extern void gometa_go_submit_input(PurpleAccount *account, char *value);
extern void gometa_go_chat_joined(PurpleAccount *account, char *chat);

#if !GLIB_CHECK_VERSION(2, 68, 0)
#define g_memdup2 g_memdup
#endif

// The name of the chat a 2FA prompt is shown in when the UI has no request dialogs.
#define GOMETA_LOGIN_CHAT "Facebook"

// ---- account settings (cache the login session) ----
char *gometa_get_setting(PurpleAccount *account, const char *key) {
    const char *v = purple_account_get_string(account, key, NULL);
    return (v && *v) ? strdup(v) : NULL;
}
void gometa_set_setting(PurpleAccount *account, const char *key, const char *value) {
    purple_account_set_string(account, key, value ? value : "");
}

// ---- interactive login prompt (2FA) ----
struct gometa_req { PurpleAccount *account; char *prompt; };

// PurpleRequestInputCb: libpurple calls these as void (*)(void *, const char *)
static void gometa_input_ok(void *account, const char *value) {
    gometa_go_submit_input((PurpleAccount *)account, (char *)(value ? value : ""));
}

static void gometa_input_cancel(void *account, const char *value) {
    (void)value;
    gometa_go_submit_input((PurpleAccount *)account, (char *)"");
}

static gboolean gometa_do_request(gpointer data) {
    struct gometa_req *r = (struct gometa_req *)data;
    PurpleConnection *gc = purple_account_get_connection(r->account);
    if (gc != NULL) {
        if (purple_request_get_ui_ops() != NULL) {
            purple_request_input(gc, "Facebook", "Facebook login code", r->prompt, NULL, FALSE, FALSE, NULL,
                "_OK", G_CALLBACK(gometa_input_ok), "_Cancel", G_CALLBACK(gometa_input_cancel),
                r->account, NULL, NULL, r->account);
        } else {
            // A UI without request dialogs (a bridge, say): ask in a chat; the reply there is the
            // code (gometa_go_send_message).
            gchar *text = g_strdup_printf("Facebook needs your login code. Reply to this chat with it.\n%s", r->prompt);
            gchar *markup = purple_markup_escape_text(text, -1);
            serv_got_im(gc, GOMETA_LOGIN_CHAT, markup, PURPLE_MESSAGE_RECV, time(NULL));
            g_free(markup);
            g_free(text);
        }
    } else {
        gometa_go_submit_input(r->account, (char *)"");
    }
    free(r->prompt);
    g_free(r);
    return FALSE;
}

void gometa_request_input(PurpleAccount *account, const char *prompt) {
    struct gometa_req *r = g_new0(struct gometa_req, 1);
    r->account = account;
    r->prompt = strdup(prompt ? prompt : "");
    purple_timeout_add(0, gometa_do_request, r);
}

static void gometa_free_strings(gometa_message_t *m) {
    free(m->who);
    free(m->conv);
    free(m->name);
    free(m->text);
    free(m->id);
    free(m->path);
    free(m->mimetype);
}

// The buddy-list group all Facebook contacts/chats live under.
PurpleGroup *gometa_blist_group(void) {
    PurpleGroup *g = purple_find_group("Facebook");
    if (!g) { g = purple_group_new("Facebook"); purple_blist_add_group(g, NULL); }
    return g;
}

static void gometa_ensure_buddy(PurpleAccount *account, const char *id, const char *name) {
    if (!id || !*id) return;
    PurpleBuddy *b = purple_find_buddy(account, id);
    if (!b) {
        b = purple_buddy_new(account, id, name);
        purple_blist_add_buddy(b, NULL, gometa_blist_group(), NULL);
    }
    if (name && *name) {
        serv_got_alias(purple_account_get_connection(account), id, name);
        purple_blist_node_set_string(&b->node, "server_alias", name);
    }
}

// The buddy-list chat of a thread, by its "id" component: a group renamed is no longer found by its
// old name.
static PurpleChat *gometa_find_chat_by_id(PurpleAccount *account, const char *threadKey) {
    for (PurpleBlistNode *node = purple_blist_get_root(); node != NULL; node = purple_blist_node_next(node, TRUE)) {
        if (!PURPLE_BLIST_NODE_IS_CHAT(node)) continue;
        PurpleChat *chat = (PurpleChat *)node;
        if (purple_chat_get_account(chat) != account) continue;
        GHashTable *comp = purple_chat_get_components(chat);
        const char *id = comp ? (const char *)g_hash_table_lookup(comp, "id") : NULL;
        if (id != NULL && strcmp(id, threadKey) == 0) return chat;
    }
    return NULL;
}

// Register a group thread as a blist chat (component "id" = thread key, used by find_blist_chat).
static void gometa_ensure_chat(PurpleAccount *account, const char *threadKey, const char *name) {
    if (!threadKey || !*threadKey) return;
    PurpleChat *chat = gometa_find_chat_by_id(account, threadKey);
    if (!chat) {
        GHashTable *comp = g_hash_table_new_full(g_str_hash, g_str_equal, NULL, g_free);
        g_hash_table_insert(comp, (gpointer)"id", g_strdup(threadKey));
        chat = purple_chat_new(account, name && *name ? name : threadKey, comp);
        purple_blist_add_chat(chat, gometa_blist_group(), NULL);
    } else if (name && *name) {
        purple_blist_alias_chat(chat, name);
        // an open chat's title is the buddy list's name for it, which may just have changed
        PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_CHAT, threadKey, account);
        if (conv != NULL) purple_conversation_autoset_title(conv);
    }
}

// Open a group's chat conversation, as a message in it does, and have Go put its members on it
// when it is new.
static void gometa_open_group(PurpleConnection *pc, const char *threadKey) {
    int chat_id = (int)g_str_hash(threadKey);
    if (purple_find_chat(pc, chat_id) == NULL) {
        serv_got_joined_chat(pc, chat_id, threadKey);
        gometa_go_chat_joined(purple_connection_get_account(pc), (char *)threadKey);
    }
}

// Who is in a group besides us, put on its chat conversation when that is open. Each is a buddy
// already (the participant rows add them), so the chat shows their names.
static void gometa_set_members(PurpleAccount *account, const char *threadKey, const char *joined) {
    PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_CHAT, threadKey, account);
    PurpleConvChat *chat = conv ? purple_conversation_get_chat_data(conv) : NULL;
    if (chat == NULL) return;
    purple_conv_chat_clear_users(chat);
    if (joined == NULL || *joined == '\0') return;
    gchar **ids = g_strsplit(joined, "\x1f", -1);
    for (gchar **id = ids; *id != NULL; id++) {
        if (**id != '\0') purple_conv_chat_add_user(chat, *id, NULL, PURPLE_CBFLAGS_NONE, FALSE);
    }
    g_strfreev(ids);
}

// Write a message (markup) into a 1:1 or group chat. Our own messages from another device are
// written as sent (PURPLE_MESSAGE_REMOTE_SEND), as the WhatsApp protocol does (glue/display_message.c).
static void gometa_write(PurpleConnection *pc, PurpleAccount *account, const gometa_message_t *m,
                         const char *text, PurpleMessageFlags extra, time_t ts) {
    const PurpleMessageFlags dir = m->isOutgoing
        ? (PurpleMessageFlags)(PURPLE_MESSAGE_SEND | PURPLE_MESSAGE_REMOTE_SEND)
        : PURPLE_MESSAGE_RECV;
    const PurpleMessageFlags flags = (PurpleMessageFlags)(dir | extra);
    if (m->isGroup) {
        gometa_open_group(pc, m->conv);
        const char *from = m->isOutgoing ? purple_account_get_username(account)
                                         : (m->who && *m->who ? m->who : "unknown");
        serv_got_chat_in(pc, (int)g_str_hash(m->conv), from, flags, text, ts);
    } else if (m->isOutgoing) {
        // serv_got_im would mark it received
        PurpleConversation *conv = purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, m->conv, account);
        if (!conv) conv = purple_conversation_new(PURPLE_CONV_TYPE_IM, account, m->conv);
        purple_conv_im_write(purple_conversation_get_im_data(conv), m->conv, text, flags, ts);
    } else {
        serv_got_im(pc, m->conv, text, flags, ts);
    }
}

// A line nobody sent - a group's news, which Facebook words itself - kept in the chat as a system
// line, not logged, as the WhatsApp protocol writes one.
static void gometa_system_line(PurpleConnection *pc, PurpleAccount *account, const char *conv,
                               int isGroup, const char *text, time_t ts) {
    const PurpleMessageFlags flags = (PurpleMessageFlags)(PURPLE_MESSAGE_RECV | PURPLE_MESSAGE_SYSTEM | PURPLE_MESSAGE_NO_LOG);
    if (isGroup) {
        gometa_open_group(pc, conv);
        serv_got_chat_in(pc, (int)g_str_hash(conv), "system", flags, text, ts);
    } else {
        if (purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, conv, account) == NULL) {
            purple_conversation_new(PURPLE_CONV_TYPE_IM, account, conv);
        }
        serv_got_im(pc, conv, text, flags, ts);
    }
}

// A received file: a picture inline, as the WhatsApp protocol shows one (glue/handle_attachment.c),
// anything else as a link to the file. The caption (markup) comes after it.
static void gometa_attachment(PurpleConnection *pc, PurpleAccount *account, const gometa_message_t *m, time_t ts) {
    GString *body = g_string_new(NULL);
    PurpleMessageFlags extra = 0;
    int img_id = 0;
    if (m->mimetype != NULL && g_str_has_prefix(m->mimetype, "image/")) {
        gchar *data = NULL;
        gsize len = 0;
        if (g_file_get_contents(m->path, &data, &len, NULL)) {
            img_id = purple_imgstore_add_with_id(data, len, NULL); // takes data
            if (img_id > 0) {
                g_string_append_printf(body, "<img id=\"%d\"/>", img_id);
                extra = PURPLE_MESSAGE_IMAGES;
            }
        }
    }
    if (img_id <= 0) {
        gchar *uri = g_filename_to_uri(m->path, NULL, NULL);
        gchar *base = g_path_get_basename(m->path);
        gchar *label = purple_markup_escape_text(base, -1);
        if (uri != NULL) {
            gchar *href = g_markup_escape_text(uri, -1);
            g_string_append_printf(body, "<a href=\"%s\">%s</a>", href, label);
            g_free(href);
        } else {
            g_string_append(body, label);
        }
        g_free(label);
        g_free(base);
        g_free(uri);
    }
    if (m->text && *m->text) {
        g_string_append(body, "<br>");
        g_string_append(body, m->text);
    }
    gometa_write(pc, account, m, body->str, extra, ts);
    if (img_id > 0) purple_imgstore_unref_by_id(img_id);
    g_string_free(body, TRUE);
}

static gboolean gometa_dispatch(gpointer data) {
    gometa_message_t *m = (gometa_message_t *)data;
    // the account may have been deleted since Go sent this
    PurpleConnection *pc = g_list_find(purple_accounts_get_all(), m->account) != NULL
        ? purple_account_get_connection(m->account) : NULL;
    if (pc != NULL) {
        time_t ts = m->timestamp ? m->timestamp : time(NULL);
        switch (m->msgtype) {
            case gometa_message_type_connected:
                purple_connection_set_state(pc, PURPLE_CONNECTED);
                break;
            case gometa_message_type_disconnected:
                purple_connection_set_state(pc, PURPLE_DISCONNECTED);
                break;
            case gometa_message_type_error:
                if (m->fatal) {
                    // A fatal error means "stop, this won't succeed on its own" (e.g. login/2FA
                    // failed, session expired). Report it with a reason libpurple considers fatal
                    // (AUTHENTICATION_FAILED) so the account is taken offline and does not
                    // reconnect: a NETWORK_ERROR would retry the failed login every minute, which
                    // can get the Facebook account flagged.
                    purple_connection_error_reason(pc, PURPLE_CONNECTION_ERROR_AUTHENTICATION_FAILED,
                        m->text ? m->text : "Unknown error");
                } else {
                    purple_debug_error(GOMETA_PLUGIN_ID, "%s\n", m->text ? m->text : "error");
                }
                break;
            case gometa_message_type_buddy:
                gometa_ensure_buddy(m->account, m->who, m->name);
                break;
            case gometa_message_type_chat:
                gometa_ensure_chat(m->account, m->conv, m->name);
                break;
            case gometa_message_type_presence:
                if (m->who && *m->who) {
                    purple_prpl_got_user_status(m->account, m->who,
                        m->isOutgoing ? "available" : "offline", NULL);
                }
                break;
            case gometa_message_type_typing:
                if (m->who && *m->who) {
                    if (m->isOutgoing) {
                        serv_got_typing(pc, m->who, 0, PURPLE_TYPING);
                    } else {
                        serv_got_typing_stopped(pc, m->who);
                    }
                }
                break;
            case gometa_message_type_text:
                if (m->text && m->conv && *m->conv) {
                    gometa_write(pc, m->account, m, m->text, 0, ts);
                }
                break;
            case gometa_message_type_attachment:
                if (m->path && *m->path && m->conv && *m->conv) {
                    gometa_attachment(pc, m->account, m, ts);
                }
                break;
            case gometa_message_type_members:
                if (m->conv && *m->conv) {
                    gometa_set_members(m->account, m->conv, m->text);
                }
                break;
            case gometa_message_type_system:
                if (m->conv && *m->conv && m->text && *m->text) {
                    gometa_system_line(pc, m->account, m->conv, m->isGroup, m->text, ts);
                }
                break;
            default:
                break;
        }
    }
    gometa_free_strings(m);
    g_free(m);
    return FALSE;
}

void gometa_process_message(gometa_message_t msg) {
    gometa_message_t *heap = g_memdup2(&msg, sizeof msg);
    purple_timeout_add(0, gometa_dispatch, heap);
}
