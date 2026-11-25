/*
 * Registers the Facebook Messenger protocol (prpl-gometa) as a second protocol of this shared
 * object, which is built with it when configured with -DGOMETA=ON. Both protocols run on the one
 * Go runtime this library carries: a process can host only one Go c-archive, so a Go-based
 * Facebook plug-in of its own could not be loaded next to the WhatsApp one.
 *
 * plugin_init() in init.c calls gometa_register_prpl().
 */
#include <purple.h>
#include <stdlib.h>
#include <string.h>
#include "gometabridge.h" // GOMETA_PLUGIN_ID / _NAME / _SUMMARY / _AUTHOR
#include "libwhatsmeow.h" // the gometa_go_* cgo exports

#ifndef PLUGIN_VERSION
#error Must set PLUGIN_VERSION in build system
#endif
#define GM_STR(x) _GM_STR(x)
#define _GM_STR(x) #x

void gometa_register_prpl(void);

static PurplePlugin *gometa_plugin = NULL;

static const char *gometa_list_icon(PurpleAccount *account, PurpleBuddy *buddy) {
    (void)account;
    (void)buddy;
    return "facebook";
}

static GList *gometa_status_types(PurpleAccount *account) {
    (void)account;
    GList *types = NULL;
    types = g_list_append(types, purple_status_type_new_full(PURPLE_STATUS_AVAILABLE, NULL, NULL, TRUE, TRUE, FALSE));
    types = g_list_append(types, purple_status_type_new_full(PURPLE_STATUS_OFFLINE, NULL, NULL, TRUE, TRUE, FALSE));
    return types;
}

/*
 * Read receipts: a chat counts as read when the UI shows it, which Pidgin tells by clearing the
 * conversation's "unseen-count" (as glue/receipt.c reads it for WhatsApp).
 */
static void gometa_conversation_updated(PurpleConversation *conv, PurpleConvUpdateType type) {
    if (type != PURPLE_CONV_UPDATE_UNSEEN) return;
    PurpleAccount *account = purple_conversation_get_account(conv);
    if (account == NULL || g_strcmp0(purple_account_get_protocol_id(account), GOMETA_PLUGIN_ID) != 0) return;
    int unseen = GPOINTER_TO_INT(purple_conversation_get_data(conv, "unseen-count"));
    int previous = GPOINTER_TO_INT(purple_conversation_get_data(conv, "gometa-previous-unseen-count"));
    const char *name = purple_conversation_get_name(conv);
    if (unseen == 0 && previous > 0 && name != NULL && *name) {
        gometa_go_mark_read(account, (char *)name);
    }
    purple_conversation_set_data(conv, "gometa-previous-unseen-count", GINT_TO_POINTER(unseen));
}

/*
 * A deleted account's E2EE store (gometa-e2ee-<username>.db in the purple user directory) goes with
 * it. libpurple frees the account as soon as the signal returns, so this works synchronously.
 */
static void gometa_account_removed(PurpleAccount *account, gpointer data) {
    (void)data;
    if (g_strcmp0(purple_account_get_protocol_id(account), GOMETA_PLUGIN_ID) != 0) return;
    gometa_go_account_removed(account, (char *)purple_user_dir(), (char *)purple_account_get_username(account));
}

/*
 * The signals are connected at the first login, not at plug-in load: libpurple loads protocol
 * plug-ins before it registers the accounts' signals.
 */
static void gometa_connect_signals_once(void) {
    static gboolean connected = FALSE;
    if (connected) return;
    connected = TRUE;
    purple_signal_connect(purple_conversations_get_handle(), "conversation-updated",
            gometa_plugin, PURPLE_CALLBACK(gometa_conversation_updated), NULL);
    purple_signal_connect(purple_accounts_get_handle(), "account-removed",
            gometa_plugin, PURPLE_CALLBACK(gometa_account_removed), NULL);
}

/*
 * Our typing state in a 1:1 chat. PURPLE_TYPED, a pause, is told as not typing: Facebook has no
 * paused state.
 */
static unsigned int gometa_send_typing(PurpleConnection *pc, const char *who, PurpleTypingState state) {
    if (who != NULL && *who) {
        gometa_go_send_typing(purple_connection_get_account(pc), (char *)who, state == PURPLE_TYPING ? 1 : 0);
    }
    return 0;
}

static void gometa_login(PurpleAccount *account) {
    PurpleConnection *pc = purple_account_get_connection(account);
    purple_connection_set_state(pc, PURPLE_CONNECTING);
    gometa_connect_signals_once();
    const char *password = purple_account_get_password(account);
    gometa_go_login(account, (char *)purple_user_dir(), (char *)purple_account_get_username(account),
            (char *)(password ? password : ""), (char *)"");
}

static void gometa_close(PurpleConnection *pc) {
    gometa_go_close(purple_connection_get_account(pc));
}

static int gometa_send_im(PurpleConnection *pc, const char *who, const char *message, PurpleMessageFlags flags) {
    (void)flags;
    return gometa_go_send_message(purple_connection_get_account(pc), (char *)who, (char *)message);
}

// ---- group chats (registered with component "id" = thread key) ----
static GList *gometa_chat_info(PurpleConnection *pc) {
    (void)pc;
    struct proto_chat_entry *pce = g_new0(struct proto_chat_entry, 1);
    pce->label = "Thread ID";
    pce->identifier = "id";
    pce->required = TRUE;
    return g_list_append(NULL, pce);
}

static GHashTable *gometa_chat_info_defaults(PurpleConnection *pc, const char *chat_name) {
    (void)pc;
    GHashTable *defaults = g_hash_table_new_full(g_str_hash, g_str_equal, NULL, g_free);
    if (chat_name) g_hash_table_insert(defaults, (gpointer)"id", g_strdup(chat_name));
    return defaults;
}

static char *gometa_get_chat_name(GHashTable *components) {
    const char *id = (const char *)g_hash_table_lookup(components, "id");
    return g_strdup(id ? id : "");
}

// Like purple_blist_find_chat but works while still connecting; matches the "id" component.
static PurpleChat *gometa_find_blist_chat(PurpleAccount *account, const char *id) {
    for (PurpleBlistNode *node = purple_blist_get_root(); node != NULL; node = purple_blist_node_next(node, TRUE)) {
        if (PURPLE_BLIST_NODE_IS_CHAT(node)) {
            PurpleChat *chat = (PurpleChat *)node;
            if (purple_chat_get_account(chat) != account) continue;
            GHashTable *comp = purple_chat_get_components(chat);
            const char *cid = comp ? (const char *)g_hash_table_lookup(comp, "id") : NULL;
            if (cid && strcmp(cid, id) == 0) return chat;
        }
    }
    return NULL;
}

static void gometa_join_chat(PurpleConnection *pc, GHashTable *components) {
    const char *id = (const char *)g_hash_table_lookup(components, "id");
    if (!id) return;
    int chat_id = (int)g_str_hash(id);
    if (purple_find_chat(pc, chat_id) == NULL) {
        serv_got_joined_chat(pc, chat_id, id);
        gometa_go_chat_joined(purple_connection_get_account(pc), (char *)id); // Go puts the members on it
    }
}

/*
 * A group chat's conversation is named after its thread key (serv_got_joined_chat), which the
 * same send path as a 1:1 chat takes.
 */
static int gometa_chat_send(PurpleConnection *pc, int id, const char *message, PurpleMessageFlags flags) {
    PurpleConversation *conv = purple_find_chat(pc, id);
    if (conv == NULL) return -6; // ENXIO "no such address"
    const char *threadKey = purple_conversation_get_name(conv);
    if (threadKey == NULL || *threadKey == '\0') return -6;
    PurpleAccount *account = purple_connection_get_account(pc);
    int ret = gometa_go_send_message(account, (char *)threadKey, (char *)message);
    if (ret > 0) {
        // libpurple echoes a sent IM, but not a chat message (as in glue/send_message.c)
        purple_conv_chat_write(purple_conversation_get_chat_data(conv), purple_account_get_username(account),
                message, flags, time(NULL));
    }
    return ret;
}

// ---- sending files (PurpleXfer -> gometa_go_send_file, which uploads via messagix or whatsmeow) ----
static void gometa_xfer_send_init(PurpleXfer *xfer) {
    PurpleAccount *account = purple_xfer_get_account(xfer);
    const char *who = purple_xfer_get_remote_user(xfer);
    const char *filename = purple_xfer_get_local_filename(xfer);
    char *error = gometa_go_send_file(account, (char *)who, (char *)filename);
    if (error && error[0]) {
        purple_xfer_error(purple_xfer_get_type(xfer), account, who, error);
        purple_xfer_cancel_local(xfer);
    } else {
        purple_xfer_set_bytes_sent(xfer, purple_xfer_get_size(xfer));
        purple_xfer_set_completed(xfer, TRUE);
    }
    free(error); // allocated by C.CString
}

static PurpleXfer *gometa_new_xfer(PurpleConnection *pc, const char *who) {
    PurpleXfer *xfer = purple_xfer_new(purple_connection_get_account(pc), PURPLE_XFER_SEND, who);
    purple_xfer_set_init_fnc(xfer, gometa_xfer_send_init);
    return xfer;
}

static void gometa_send_file(PurpleConnection *pc, const char *who, const char *filename) {
    PurpleXfer *xfer = gometa_new_xfer(pc, who);
    if (filename && *filename) {
        purple_xfer_request_accepted(xfer, filename);
    } else {
        purple_xfer_request(xfer);
    }
}

#if PURPLE_VERSION_CHECK(2,14,0)
// Group file send: the chat conversation name is the thread key (see gometa_chat_send).
static void gometa_chat_send_file(PurpleConnection *pc, int id, const char *filename) {
    PurpleConversation *conv = purple_find_chat(pc, id);
    if (conv != NULL) {
        const char *who = purple_conversation_get_name(conv);
        if (who != NULL && *who) {
            gometa_send_file(pc, who, filename);
        }
    }
}
#endif

static PurplePluginProtocolInfo gometa_prpl_info = {
    .struct_size = sizeof(PurplePluginProtocolInfo),
    .options = OPT_PROTO_IM_IMAGE,
    .list_icon = gometa_list_icon,
    .status_types = gometa_status_types,
    .login = gometa_login,
    .close = gometa_close,
    .send_im = gometa_send_im,
    .send_typing = gometa_send_typing,
    .new_xfer = gometa_new_xfer,
    .send_file = gometa_send_file,
    #if PURPLE_VERSION_CHECK(2,14,0)
    .chat_send_file = gometa_chat_send_file,
    #endif
    .chat_info = gometa_chat_info,
    .chat_info_defaults = gometa_chat_info_defaults,
    .join_chat = gometa_join_chat,
    .get_chat_name = gometa_get_chat_name,
    .find_blist_chat = gometa_find_blist_chat,
    .chat_send = gometa_chat_send,
};

static PurplePluginInfo gometa_info = {
    .magic = PURPLE_PLUGIN_MAGIC,
    .major_version = PURPLE_MAJOR_VERSION,
    .minor_version = PURPLE_MINOR_VERSION,
    .type = PURPLE_PLUGIN_PROTOCOL,
    .priority = PURPLE_PRIORITY_DEFAULT,
    .id = GOMETA_PLUGIN_ID,
    .name = GOMETA_PLUGIN_NAME,
    .version = GM_STR(PLUGIN_VERSION),
    .summary = GOMETA_PLUGIN_SUMMARY,
    .description = GOMETA_PLUGIN_SUMMARY,
    .author = GOMETA_AUTHOR,
    .homepage = "https://github.com/hoehermann/purple-gowhatsapp",
    .extra_info = &gometa_prpl_info,
};

/*
 * A protocol plug-in with no file of its own. libpurple keeps no path or module handle for it,
 * which is safe: it saves only non-protocol plug-ins as loaded, and closes a module only when the
 * plug-in has a handle.
 */
void gometa_register_prpl(void) {
    if (gometa_plugin != NULL) return;
    PurplePlugin *p = purple_plugin_new(TRUE, NULL);
    p->info = &gometa_info;
    purple_plugin_load(p);
    if (purple_plugin_register(p)) {
        gometa_plugin = p;
        purple_debug_info(GOMETA_PLUGIN_ID, "registered\n");
    } else {
        purple_debug_error(GOMETA_PLUGIN_ID, "could not register the protocol\n");
    }
}
