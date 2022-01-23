#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

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

    //WhatsappAccountData *wad = g_new0(WhatsappAccountData, 1);
    //purple_connection_set_protocol_data(pc, wad);
    
    purple_connection_set_state(pc, PURPLE_CONNECTION_CONNECTING);
    const char *credentials = purple_account_get_string(account, GOWHATSAPP_CREDENTIALS_KEY, NULL);
    if (credentials == NULL) {
        credentials = purple_account_get_password(account); // bitlbee stores credentials in password field
    }
    char *username = (char *)purple_account_get_username(account); // cgo does not suport const
    char *user_dir = (char *)purple_user_dir(); // cgo does not suport const
    gowhatsapp_go_login(account, user_dir, username, (char *)credentials); // cgo does not suport const
    
    gowhatsapp_receipts_init(pc);
}

void
gowhatsapp_close(PurpleConnection *pc)
{
    PurpleAccount * account = purple_connection_get_account(pc);
    gowhatsapp_go_close(account);
    
    //g_free(purple_connection_get_protocol_data(pc));
}

void
purple_account_set_credentials(PurpleAccount *account, char *credentials)
{
    // Pidgin stores the credentials in the account settings
    // in bitlbee, this has no effect
    purple_account_set_string(account, GOWHATSAPP_CREDENTIALS_KEY, credentials);
    
    // bitlbee stores credentials in password field
    // in Pidgin, these lines have no effect
    purple_account_set_password(account, credentials);
    purple_signal_emit(
        purple_accounts_get_handle(),
        "bitlbee-set-account-password",
        account,
        credentials
    );
}
