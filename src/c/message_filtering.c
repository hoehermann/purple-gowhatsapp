#include "gowhatsapp.h"
#include "constants.h"

/*
 * Checks whether message is old. Compares with previous session's last
 * timestamp and account option for age. Updates last session timestamp
 * in account.
 */
gboolean
gowhatsapp_message_is_new_enough(PurpleAccount *account, const time_t ts) {
    int max_history_seconds = purple_account_get_int(
        account, GOWHATSAPP_MAX_HISTORY_SECONDS_OPTION, 0
    );

    time_t cur_raw_time;
    time(&cur_raw_time);

    time_t message_age = cur_raw_time - ts;

    gboolean new_by_config = (
        max_history_seconds == 0 || message_age <= max_history_seconds
    );

    int previous_sessions_last_messages_timestamp = purple_account_get_int(account, GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, 0);
    if (ts < previous_sessions_last_messages_timestamp) {
        return FALSE;
    } else {
        purple_account_set_int(account, GOWHATSAPP_PREVIOUS_SESSION_TIMESTAMP_KEY, ts); // TODO: find out if this narrowing of time_t aka. long int to int imposes a problem
        return new_by_config;
    }
}

/*
 * Builds a persistent list of IDs of messages already received.
 * 
 * @return Message ID is new.
 */
gboolean
gowhatsapp_append_message_id_if_not_exists(PurpleAccount *account, char *message_id)
{
    if (message_id == NULL || message_id[0] == 0) {
        // always display system messages (they have no ID)
        return TRUE;
    }
    const gchar *received_messages_ids_str = purple_account_get_string(account, GOWHATSAPP_RECEIVED_MESSAGES_ID_KEY, "");
    if (strstr(received_messages_ids_str, message_id) != NULL) {
        purple_debug_info(
            "gowhatsapp", "Suppressed message (already received).\n"
        );
        return FALSE;
    } else {
        // count ids currently in store
        static const char GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR = ',';
        const unsigned int GOWHATSAPP_MESSAGEIDSTORE_SIZE = purple_account_get_int(account, GOWHATSAPP_MESSAGE_ID_STORE_SIZE_OPTION, 1000);
        unsigned int occurrences = 0;
        char *offset = (char *)received_messages_ids_str + strlen(received_messages_ids_str);
        for (; offset > received_messages_ids_str; offset--) {
            if (*offset == GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR) {
                occurrences++;
                if (occurrences >= GOWHATSAPP_MESSAGEIDSTORE_SIZE) {
                    // store is full, stop counting
                    break;
                }
            }
        }
        // append new ID
        gchar *new_received_messages_ids_str = g_strdup_printf("%s%c%s", offset, GOWHATSAPP_MESSAGEIDSTORE_SEPARATOR, message_id);
        purple_account_set_string(account, GOWHATSAPP_RECEIVED_MESSAGES_ID_KEY, new_received_messages_ids_str);
        g_free(new_received_messages_ids_str);
        return TRUE;
    }
}
