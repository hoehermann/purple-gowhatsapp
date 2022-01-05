#include "gowhatsapp.h"
void
gowhatsapp_login(PurpleAccount *account)
{   
    purple_connection_set_state(purple_account_get_connection(account), PURPLE_CONNECTING);
    char * username = (char *)purple_account_get_username(account); // cgo does not suport const
    gowhatsapp_go_login(username);
}

void
gowhatsapp_close(PurpleConnection *pc)
{
    PurpleAccount * account = purple_connection_get_account(pc);
    char * username = (char *)purple_account_get_username(account); // cgo does not suport const
    gowhatsapp_go_close(username);
}
