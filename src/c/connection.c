#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

void
gowhatsapp_login(PurpleAccount *account)
{   
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
