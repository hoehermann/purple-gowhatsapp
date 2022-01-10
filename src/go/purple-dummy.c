#include <stdint.h>
#include "bridge.h"

/*
 * This module implements dummy bodies for C functions to be called by cgo.
 * cgo uses the module to determine the parameter types, but it does not actually link to it.
 * This module MUST NOT be linked with the main target!
 */

void gowhatsapp_process_message_bridge(gowhatsapp_message_t gwamsg)
{
    (void) gwamsg;
}

int
gowhatsapp_account_exists(PurpleAccount *account)
{
    (void) account;
    return 0;
}

int 
purple_account_get_int(PurpleAccount *account, const char *name, int default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}

const char * 
purple_account_get_string(PurpleAccount *account, const char *name, const char *default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}

void
purple_account_set_username(PurpleAccount *account, const char *username)
{
    (void) account;
    (void) username;
}

void
purple_account_set_password(PurpleAccount *account, const char *password)
{
    (void) account;
    (void) password;
}

/*
const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account) {
    (void) account;
    return (void *)0xDEADBEEF;
};
*/
