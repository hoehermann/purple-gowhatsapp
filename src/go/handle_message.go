package main

import (
	"encoding/hex"
	"go.mau.fi/whatsmeow/types/events"
	"mime"
	"strings"
)

func (handler *Handler) handle_message(evt *events.Message) {
	handler.log.Infof("Received message: %#v", evt)
	message := evt.Message
	info := evt.Info
	text := message.GetConversation()
	etm := message.ExtendedTextMessage
	if etm != nil {
		// message containing quote or link to group
		// link messages have message.Conversation set to nil anyway
		// it should be safe to overwrite here
		// quoted message repeats the text
		ci := etm.ContextInfo
		if ci != nil {
			cm := ci.QuotedMessage
			if cm != nil && cm.Conversation != nil {
				quotelines := strings.Split(*cm.Conversation, "\n")
				text = "> " + strings.Join(quotelines, "\n> ") + "\n"
			}
		}
		if etm.Text != nil {
			text += *etm.Text
		}
	}

	im := message.GetImageMessage()
	if im != nil && im.Caption != nil {
		text += *message.GetImageMessage().Caption
	}
	vm := message.GetVideoMessage()
	if vm != nil && vm.Caption != nil {
		text += *message.GetVideoMessage().Caption
	}

	if text == "" {
		handler.log.Warnf("Received a message without any text.")
	} else {
		purple_display_text_message(handler.username, info.MessageSource.Chat.ToNonAD().String(), info.MessageSource.IsGroup, info.MessageSource.IsFromMe, info.MessageSource.Sender.ToNonAD().String(), &info.PushName, info.Timestamp, text)
		handler.mark_read_immediately(info.ID, info.MessageSource.Chat, info.MessageSource.Sender)
	}

	handler.handle_attachment(evt)
}

func extension_from_mimetype(mimeType *string) string {
	extension := ".data"
	if mimeType != nil {
		extensions, _ := mime.ExtensionsByType(*mimeType)
		if extensions != nil {
			extension = extensions[0]
		}
	}
	return extension
}

// based on https://github.com/FKLC/WhatsAppToDiscord/blob/master/WA2DC.go
func (handler *Handler) handle_attachment(evt *events.Message) {
	var (
		data     []byte
		err      error
		filename = ""
	)
	message := evt.Message
	im := message.GetImageMessage()
	if im != nil {
		data, err = handler.client.Download(im)
		filename = hex.EncodeToString(im.FileSha256) + extension_from_mimetype(im.Mimetype)
	}
	vm := message.GetVideoMessage()
	if vm != nil {
		data, err = handler.client.Download(vm)
		filename = hex.EncodeToString(vm.FileSha256) + extension_from_mimetype(vm.Mimetype)
	}
	am := message.GetAudioMessage()
	if am != nil {
		data, err = handler.client.Download(am)
		filename = hex.EncodeToString(am.FileSha256) + extension_from_mimetype(am.Mimetype)
	}
	dm := message.GetDocumentMessage()
	if dm != nil {
		data, err = handler.client.Download(dm)
		filename = *message.GetDocumentMessage().Title
		// TODO: sanitize filename
	}
	sm := message.GetStickerMessage()
	if sm != nil {
		data, err = handler.client.Download(sm)
		filename = hex.EncodeToString(sm.FileSha256) + extension_from_mimetype(sm.Mimetype)
	}
	if err != nil {
		// TODO: display error in conversation
		handler.log.Errorf("Download failed: %#v", err)
	}
	if filename != "" {
		sender := evt.Info.MessageSource.Sender.ToNonAD().String()
		purple_handle_attachment(handler.username, sender, filename, data)
	}
}
