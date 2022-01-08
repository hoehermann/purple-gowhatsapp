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
    char * username = (char *)purple_account_get_username(account); // cgo does not suport const
    gowhatsapp_go_login(username);
}

void
gowhatsapp_close(PurpleConnection *pc)
{
    PurpleAccount * account = purple_connection_get_account(pc);
    char * username = (char *)purple_account_get_username(account); // cgo does not suport const
    gowhatsapp_go_close(username);
    
    //g_free(purple_connection_get_protocol_data(pc));
}
