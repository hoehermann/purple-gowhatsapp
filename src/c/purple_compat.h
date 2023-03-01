#include <purple.h>

#define PURPLE_XFER_TYPE_SEND PURPLE_XFER_SEND
#define PURPLE_IS_CHAT PURPLE_BLIST_NODE_IS_CHAT
#define purple_connection_error purple_connection_error_reason
#define PURPLE_CONNECTION_CONNECTING PURPLE_CONNECTING
#define PURPLE_CONNECTION_CONNECTED PURPLE_CONNECTED
#define PURPLE_CONNECTION_DISCONNECTED PURPLE_DISCONNECTED
#define PurpleIMConversation PurpleConvIm
#define purple_blist_find_group purple_find_group
#define purple_blist_find_buddy purple_find_buddy
#define purple_chat_conversation_add_user purple_conv_chat_add_user
#define purple_im_conversation_new(account, from) PURPLE_CONV_IM(purple_conversation_new(PURPLE_CONV_TYPE_IM, account, from))
#define PURPLE_CONVERSATION(chatorim) ((chatorim) == NULL ? NULL : (chatorim)->conv)
#define purple_serv_got_im serv_got_im
#define purple_connection_set_flags(pc, f) ((pc)->flags = (f))
#define purple_connection_get_flags(pc) ((pc)->flags)
#define purple_connection_get_protocol          purple_connection_get_prpl
#define PurpleConversationUpdateType       PurpleConvUpdateType
#define PURPLE_CONVERSATION_UPDATE_UNSEEN  PURPLE_CONV_UPDATE_UNSEEN
#define purple_blist_chat_new purple_chat_new
