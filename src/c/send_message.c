#include "gowhatsapp.h"
#include "purple-go-whatsapp.h"

int
gowhatsapp_send_im(PurpleConnection *pc, const gchar *who, const gchar *message, PurpleMessageFlags flags)
{
    // strip html similar to these reasons: https://github.com/majn/telegram-purple/issues/12 and https://github.com/majn/telegram-purple/commit/fffe7519d7269cf4e5029a65086897c77f5283ac
    char *msg = purple_markup_strip_html(message);
    PurpleAccount *account = purple_connection_get_account(pc);
    char *username = (char *)purple_account_get_username(account); // cgo does not suport const
    char *w = (char *)who; // cgo does not suport const
    return gowhatsapp_go_send_message(username, w, msg);
}
