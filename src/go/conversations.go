package main

import (
	//"encoding/json"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"time"
)

/*
 * Handle a conversation (history of a chat).
 *
 * Many checks are done silently since WhatsApp servers should never supply an invalid data.
 */
func (handler *Handler) handle_historical_conversations(conversations []*waProto.Conversation) {
	for _, conversation := range conversations {
		//s, _ := json.MarshalIndent(conversation, "", "  ")
		//handler.log.Infof("Received Conversation:\n\n%s\n\n", s)
		if conversation != nil && conversation.Id != nil {
			// conversation.Name may be present for identifying the Group title
			chat, err := types.ParseJID(*conversation.Id)
			if err == nil {
				for _, historysyncmessage := range conversation.Messages {
					webmessageinfo := historysyncmessage.Message
					if webmessageinfo != nil && webmessageinfo.Key != nil && webmessageinfo.Key.FromMe != nil && webmessageinfo.Key.Id != nil && webmessageinfo.MessageTimestamp != nil && webmessageinfo.Message != nil {
						sender := chat // assume a direct conversation by default
						if webmessageinfo.Participant != nil {
							// in case a participant is given, use them for the sender
							sender, err = types.ParseJID(*webmessageinfo.Participant)
						}
						if err == nil {
							//chat, err := types.ParseJID(webmessageinfo.Key.RemoteJid) // already taken care of in outer loop
							isGroup := chat.Server == types.GroupServer || chat.Server == types.BroadcastServer // taken from whatsmeow's message.go
							source := types.MessageSource{
								Chat:     chat,
								IsGroup:  isGroup,
								IsFromMe: *webmessageinfo.Key.FromMe,
								Sender:   sender}
							timestamp := time.Unix(int64(*webmessageinfo.MessageTimestamp), 0)
							is_historical := true
							handler.handle_message(webmessageinfo.Message, *webmessageinfo.Key.Id, source, nil, timestamp, is_historical)
						}
					}
				}
			}
		}
	}
}
