#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

/////////////////////////////////////////////////////////////////////
//                                                                 //
//      WELCOME TO THE LAND OF ABANDONMENT OF TYPE AND SAFETY      //
//                        Wanderer, beware.                        //
//                                                                 //
/////////////////////////////////////////////////////////////////////

/*
 * This allows cgo to acces the PurpleAccount pointer having only an integer which actually is the PurpleConnection pointer.
 */
void *
gowhatsapp_get_account(uintptr_t pc)
{
    return purple_connection_get_account((PurpleConnection *)pc);
}

/*
 * This allows cgo to get a boolean from the current account settings.
 * The PurpleAccount pointer is untyped and the gboolean is implicitly promoted to int.
 */
int
gowhatsapp_account_get_bool(void *account, const char *name, int default_value)
{
    return purple_account_get_bool((const PurpleAccount *)account, name, default_value);
}

/*
 * This allows cgo to get a string from the current account settings.
 * The PurpleAccount pointer is untyped.
 */
const char * gowhatsapp_account_get_string(void *account, const char *name, const char *default_value)
{
    return purple_account_get_string((const PurpleAccount *)account, name, default_value);
}

/*
 * Session data is stored in the password and can be retrieved via this
 */
const char * gowhatsapp_account_get_password(void *account)
{
    return purple_account_get_password((const PurpleAccount *)account);
}

/*
 * This allows cgo to get a string from the current account settings.
 * The PurpleAccount pointer is untyped.
 */
const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account)
{
    PurpleProxyInfo *proxyInfo = purple_account_get_proxy_info((const PurpleAccount *)account);
    // TODO: find out whether proxyInfo == null means "use global" or "use none"
    if (!proxyInfo) {
        purple_debug_info("gowhatsapp", "Account has no proxy info.\n");
    }
    if (proxyInfo && proxyInfo->type == PURPLE_PROXY_USE_GLOBAL) {
        purple_debug_info("gowhatsapp", "Getting global proxy info…\n");
        proxyInfo = purple_global_proxy_get_info();
    }
    return proxyInfo;
}