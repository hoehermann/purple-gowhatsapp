package main

import (
	"fmt"
	"go.mau.fi/whatsmeow/types/events"
	"mime"
)

func (handler *Handler) handle_message(evt *events.Message) {
	handler.log.Infof("Received message: %#v", evt)
	text := evt.Message.GetConversation()
	var quote *string = nil
	etm := evt.Message.ExtendedTextMessage
	if etm != nil {
		quote = etm.ContextInfo.QuotedMessage.Conversation
	}
	purple_display_text_message(handler.username, &evt.Info.ID, evt.Info.MessageSource.Chat.ToNonAD().String(), evt.Info.MessageSource.IsGroup, evt.Info.MessageSource.IsFromMe, evt.Info.MessageSource.Sender.ToNonAD().String(), &evt.Info.PushName, evt.Info.Timestamp, text, quote)
	// TODO: investigate evt.Info.SourceString() in context of group messages

	// TODO: implement receiving files
	img := evt.Message.GetImageMessage()
	if img != nil {
		// data, err := cli.Download(img)
		exts, _ := mime.ExtensionsByType(img.GetMimetype())
		path := fmt.Sprintf("%s%s", evt.Info.ID, exts[0])
		handler.log.Infof("ImageMessage: %s", path)
	}
}
