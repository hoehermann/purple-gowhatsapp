#include "gowhatsapp.h"
#include "purple-go-whatsapp.h" // for gowhatsapp_go_login

void
gowhatsapp_login(PurpleAccount *account)
{
    PurpleConnection *pc = purple_account_get_connection(account);
    
    // this protocol does not support anything special right now
    PurpleConnectionFlags pc_flags;
    pc_flags = purple_connection_get_flags(pc);
    pc_flags |= PURPLE_CONNECTION_NO_IMAGES;
    pc_flags |= PURPLE_CONNECTION_NO_FONTSIZE;
    pc_flags |= PURPLE_CONNECTION_NO_BGCOLOR;
    purple_connection_set_flags(pc, pc_flags);

    purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTING);
    
    WhatsappProtocolData *wpd = g_new0(WhatsappProtocolData, 1); // MEMCHECK: released in gowhatsapp_close
    purple_connection_set_protocol_data(pc, wpd);
    
    char *proxy_address = NULL;
    PurpleProxyInfo *proxy_info = purple_proxy_get_setup(account);
    if (proxy_info != NULL && purple_proxy_info_get_type(proxy_info) != PURPLE_PROXY_NONE) {
        if (purple_proxy_info_get_type(proxy_info) != PURPLE_PROXY_SOCKS5) {
            purple_connection_error(pc, PURPLE_CONNECTION_ERROR_OTHER_ERROR, "socks5 is the only supported proxy scheme.");
            return;
        }
        // TODO: find out if there is an opposite to purple_url_parse
        // or forward proxy_info into go and use client.SetProxy
        const char *proxy_username = purple_proxy_info_get_username(proxy_info);
        const char *proxy_password = purple_proxy_info_get_password(proxy_info);
        const char *proxy_host = purple_proxy_info_get_host(proxy_info);
        int proxy_port = purple_proxy_info_get_port(proxy_info);
        GString * proxy_string = g_string_new(proxy_host); // MEMCHECK: proxy_address takes ownership
        if (proxy_username && proxy_username[0]) {
            proxy_string = g_string_prepend_c(proxy_string, '@');
            if (proxy_password && proxy_password[0]) {
                proxy_string = g_string_prepend(proxy_string, proxy_password);
                proxy_string = g_string_prepend_c(proxy_string, ':');
            }
            proxy_string = g_string_prepend(proxy_string, proxy_username);
        }
        proxy_string = g_string_append_c(proxy_string, ':');
        g_string_append_printf(proxy_string, "%d", proxy_port);
        proxy_string = g_string_prepend(proxy_string, "socks5://");
        proxy_address = g_string_free(proxy_string, FALSE); // MEMCHECK: free'd here
        purple_debug_info(GOWHATSAPP_NAME, "Using proxy address %s.\n", proxy_address);
    } else {
        purple_debug_info(GOWHATSAPP_NAME, "No proxy set in purple. The go runtime might pick up the https_proxy environment variable regardless.\n");
    }
    
    const char *credentials = purple_account_get_string(account, GOWHATSAPP_CREDENTIALS_KEY, NULL);
    if (credentials == NULL) {
        credentials = purple_account_get_password(account); // bitlbee stores credentials in password field
    }
    char *username = (char *)purple_account_get_username(account); // cgo does not suport const
    char *user_dir = (char *)purple_user_dir(); // cgo does not suport const
    gowhatsapp_go_login(account, user_dir, username, (char *)credentials, proxy_address); // cgo does not suport const
    g_free(proxy_address);
    
    gowhatsapp_receipts_init(pc);
}

void
gowhatsapp_close(PurpleConnection *pc)
{
    PurpleAccount * account = purple_connection_get_account(pc);
    gowhatsapp_go_close(account);
    
    WhatsappProtocolData *wpd = (WhatsappProtocolData *)purple_connection_get_protocol_data(pc);
    purple_connection_set_protocol_data(pc, NULL);
    g_free(wpd);
}

void
gowhatsapp_store_credentials(PurpleAccount *account, char *credentials)
{
    // Pidgin stores the credentials in the account settings
    // since commit ee89203, spectrum supports this out of the box
    // in bitlbee, this has no effect
    // TODO: ask spectrum maintainer if storing in password woukd okay, too
    // or do not store credentials at all (just use the username for look-up)
    purple_account_set_string(account, GOWHATSAPP_CREDENTIALS_KEY, credentials);
    
    // bitlbee stores credentials in password field
    purple_account_set_password(account, credentials);
    purple_signal_emit(
        purple_accounts_get_handle(),
        "bitlbee-set-account-password",
        account,
        credentials
    );
}
