#include <purple.h>

namespace gowhatsapp {
const char * list_icon(PurpleAccount *account, PurpleBuddy *buddy);
void close(PurpleConnection *pc);
void login(PurpleAccount *account);
}