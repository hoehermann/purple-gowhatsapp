#include <stdint.h>
//#include "constants.h"
//#include "proxy.h"

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
purple_connection_get_account(void *pc)
{
    (void) pc;
    return (void *)0xDEADBEEF;
}

const char *
purple_account_get_password(void *account)
{
    (void) account;
    return 0;
}

const char * 
purple_account_get_string(void *account, const char *name, const char *default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}

// this works as long as gboolean is gint is int
int 
purple_account_get_bool(void *account, const char *name, int default_value)
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