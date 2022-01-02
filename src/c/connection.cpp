#include "gowhatsapp.hpp"
#include "purple-go-whatsapp.h"

void
gowhatsapp::close(PurpleConnection *pc)
{
    gowhatsapp_go_close(uintptr_t(pc));
}

void
gowhatsapp::login(PurpleAccount *account)
{
    PurpleConnection *pc = purple_account_get_connection(account);
    gowhatsapp_go_login(uintptr_t(pc));
}
