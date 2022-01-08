#include <stdint.h>

/*
 * This module implements dummy bodies for C functions to be called by cgo.
 * cgo uses the module to determine the parameter types, but it does not actually link to it.
 * This module MUST NOT be linked with the main target!
 */

void gowhatsapp_process_message_bridge(void *gwamsg)
{
    (void) gwamsg;
}

void *
gowhatsapp_get_account(char *username)
{
    (void) username;
    return 0;
}

int 
purple_account_get_int(void *account, const char *name, int default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}

/*
const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account) {
    (void) account;
    return (void *)0xDEADBEEF;
};
*/
