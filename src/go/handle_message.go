package main

import (
	"fmt"
	"go.mau.fi/whatsmeow/types/events"
	"mime"
	"strings"
)

func (handler *Handler) handle_message(evt *events.Message) {
	handler.log.Infof("Received message: %#v", evt)

	text := evt.Message.GetConversation()
	etm := evt.Message.ExtendedTextMessage
	if etm != nil {
		// message containing quote or link to group
		// link messages have evt.Message.Conversation set to nil anyway
		// it should be safe to overwrite here
		// quoted message repeats the text
		ci := etm.ContextInfo
		if ci != nil {
			cm := ci.QuotedMessage
			if cm != nil && cm.Conversation != nil {
				quotelines := strings.Split(*cm.Conversation, "\n")
				text = "> "+strings.Join(quotelines, "\n> ")+"\n"
			}
		}
		if etm.Text != nil {
			text += *etm.Text
		}
	}
	purple_display_text_message(handler.username, &evt.Info.ID, evt.Info.MessageSource.Chat.ToNonAD().String(), evt.Info.MessageSource.IsGroup, evt.Info.MessageSource.IsFromMe, evt.Info.MessageSource.Sender.ToNonAD().String(), &evt.Info.PushName, evt.Info.Timestamp, text)
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
