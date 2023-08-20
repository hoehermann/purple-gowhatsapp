#include <stdint.h>
#include "../c/bridge.h"
#include "../c/opusreader.h"

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

int 
purple_account_get_bool(PurpleAccount *account, const char *name, int default_value)
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
purple_account_set_credentials(PurpleAccount *account, char *credentials)
{
    (void) account;
    (void) credentials;
}

/*
const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account) {
    (void) account;
    return (void *)0xDEADBEEF;
};
*/

struct opusfile_info opusfile_get_info(void *data, size_t size) {
  struct opusfile_info info = {.length_seconds = -1}; // use negative length to indicate error
  return info;
}
