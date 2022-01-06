#include <purple.h>

#define purple_connection_error purple_connection_error_reason
#define PURPLE_CONNECTION_CONNECTING PURPLE_CONNECTING
#define PURPLE_CONNECTION_CONNECTED PURPLE_CONNECTED
#define PurpleIMConversation PurpleConvIm
#define purple_blist_find_group purple_find_group
#define purple_blist_find_buddy purple_find_buddy
#define purple_chat_conversation_add_user purple_conv_chat_add_user
#define purple_conversations_find_chat_with_account(id, account) \
	PURPLE_CONV_CHAT(purple_find_conversation_with_account(PURPLE_CONV_TYPE_CHAT, id, account))
#define purple_conversations_find_im_with_account(name, account) \
	PURPLE_CONV_IM(purple_find_conversation_with_account(PURPLE_CONV_TYPE_IM, name, account))
#define purple_im_conversation_new(account, from) PURPLE_CONV_IM(purple_conversation_new(PURPLE_CONV_TYPE_IM, account, from))
#define PURPLE_CONVERSATION(chatorim) ((chatorim) == NULL ? NULL : (chatorim)->conv)
#define purple_serv_got_im serv_got_im
