#include <stdint.h>
#include "constants.h"

void gowhatsapp_process_message_bridge(uintptr_t pc, void *gwamsg)
{
	(void) pc;
	(void) gwamsg;
}

void *
gowhatsapp_get_account(void *pc)
{
    return (void *)0xDEADBEEF;
}

const char *purple_account_get_string(void *account, const char *name, const char *default_value)
{
    return "Dummy implementation";
}

int 
gowhatsapp_account_get_bool(void *account, const char *name, int default_value)
{
    return 0xDEADBEEF;
}
