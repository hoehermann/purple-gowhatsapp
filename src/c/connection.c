#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

void
gowhatsapp_close(PurpleConnection *pc)
{
    gowhatsapp_go_close((uintptr_t)pc);
}

void
gowhatsapp_login(PurpleAccount *account)
{
    PurpleConnection *pc = purple_account_get_connection(account);
    gowhatsapp_go_login((uintptr_t)pc);
}
