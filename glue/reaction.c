#include "gowhatsapp.h"
#include "constants.h"

/*
 * An incoming reaction: sender put an emoji onto the message named by its id,
 * or took it back again (empty emoji).
 *
 * libpurple has no native concept of reactions, so this emits the
 * GOWHATSAPP_SIGNAL_REACTION signal for front-ends and plug-ins that can
 * attach the emoji to the message it names. The classic textual rendering
 * ("reacted with … to …") lives in the Go part and stays the default, see the
 * reaction-display account option.
 */
void
gowhatsapp_handle_reaction(PurpleConnection *pc, gowhatsapp_message_t *gwamsg) {
    gchar *timestamp = g_strdup_printf("%ld", (long)gwamsg->timestamp);
    GHashTable *details = g_hash_table_new(g_str_hash, g_str_equal); // MEMCHECK: values are borrowed, receivers must copy
    g_hash_table_insert(details, "chat", gwamsg->remoteJid);
    g_hash_table_insert(details, "sender", gwamsg->senderJid);
    g_hash_table_insert(details, "id", gwamsg->messageId);
    g_hash_table_insert(details, "emoji", gwamsg->text);
    g_hash_table_insert(details, "isGroup", gwamsg->isGroup ? "1" : "0");
    g_hash_table_insert(details, "timestamp", timestamp);
    purple_signal_emit(purple_connection_get_prpl(pc), GOWHATSAPP_SIGNAL_REACTION, pc, details);
    g_hash_table_destroy(details);
    g_free(timestamp);
}
