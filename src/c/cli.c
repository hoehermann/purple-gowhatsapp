#include "bridge.h"
#include "purple-go-whatsapp.h"
#include <stdio.h>
#include <glib.h>

static const char *gowhatsapp_message_type_string[] = {
    FOREACH_MESSAGE_TYPE(GENERATE_STRING)
};

void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg_)
{
    gowhatsapp_message_t * gwamsg = &gwamsg_;
    printf(
        "recieved %s (subtype %d) remote %s (isGroup %d) sender %s (alias %s, fromMe %d) sent %ld:%c%s%s",
        gowhatsapp_message_type_string[gwamsg->msgtype],
        gwamsg->subtype,
        gwamsg->remoteJid,
        gwamsg->isGroup,
        gwamsg->senderJid,
        gwamsg->name,
        gwamsg->fromMe,
        gwamsg->timestamp,
        gwamsg->msgtype == gowhatsapp_message_type_log ? '\n' : ' ',
        gwamsg->text,
        gwamsg->msgtype == gowhatsapp_message_type_log ? "" : "\n"
    );
    
    g_free(gwamsg->remoteJid);
    g_free(gwamsg->senderJid);
    g_free(gwamsg->text);
    g_free(gwamsg->name);
    g_free(gwamsg->blob);
    for(char **iter = gwamsg->participants; iter != NULL && *iter != NULL; iter++) {
        g_free(*iter);
    }
    g_free(gwamsg->participants);
}

int
gowhatsapp_account_exists(PurpleAccount *account)
{
    (void) account;
    return 1;
}

int 
purple_account_get_int(PurpleAccount *account, const char *name, int default_value)
{
    (void) account;
    (void) name;
    return default_value;
}

int 
purple_account_get_bool(PurpleAccount *account, const char *name, int default_value)
{
    (void) account;
    (void) name;
    return default_value;
}

const char * 
purple_account_get_string(PurpleAccount *account, const char *name, const char *default_value)
{
    (void) account;
    (void) name;
    return default_value;
}

void
purple_account_set_credentials(PurpleAccount *account, char *credentials)
{
    (void) account;
    printf("recieved credentials %s\n", credentials);
}


int main(int argc, char **argv) {
    const int argcount = 4;
    if (argc != argcount) {
        printf("Must have exactly %d arguments: user_dir, username, credentials\n", argcount-1);
        return 1;
    }
    char *user_dir = argv[1];
    char *username = argv[2];
    char *credentials = argv[3];
    void *account = (void *)1;
    gowhatsapp_go_login(account, user_dir, username, credentials);
    printf("Hit enter to terminate.\n");
    getchar();
    gowhatsapp_go_close(account);
}
