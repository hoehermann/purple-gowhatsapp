#include <stdint.h>
#include "constants.h"

/*
 * This module implements dummy bodies for C functions to be called by cgo.
 * cgo uses the module to determine the parameter types, but it does not actually link to it.
 * This module MUST NOT be linked with the main target libgowhatsapp.so.
 */

void gowhatsapp_process_message_bridge(void *gwamsg)
{
	(void) gwamsg;
}

void *
gowhatsapp_get_account(void *pc)
{
    (void) pc;
    return (void *)0xDEADBEEF;
}

const char * 
gowhatsapp_account_get_string(void *account, const char *name, const char *default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}

int 
gowhatsapp_account_get_bool(void *account, const char *name, int default_value)
{
    (void) account;
    (void) name;
    (void) default_value;
    return 0;
}
