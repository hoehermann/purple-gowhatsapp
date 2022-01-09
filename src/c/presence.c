#include "gowhatsapp.h"
#include "constants.h"

void
gowhatsapp_handle_presence(PurpleAccount *account, char *remoteJid, char online, time_t lastseen) {
    char *status = GOWHATSAPP_STATUS_STR_ONLINE;
    if (online == 0) {
        if (purple_account_get_bool(account, GOWHATSAPP_FAKE_ONLINE_OPTION, TRUE)) {
            status = GOWHATSAPP_STATUS_STR_AWAY;
        } else {
            status = GOWHATSAPP_STATUS_STR_OFFLINE;
        }
    }
    purple_prpl_got_user_status(account, remoteJid, status, NULL);
    
    if (lastseen != 0) {
        PurpleBuddy *buddy = purple_blist_find_buddy(account, remoteJid);
        if (buddy){
            purple_blist_node_set_int(&buddy->node, "last_seen", lastseen);
        }
    }
}
