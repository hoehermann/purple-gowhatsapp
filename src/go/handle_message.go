package main

/*
#include "../c/constants.h"
#include "../c/bridge.h"
*/
import "C"

import (
	"encoding/hex"
	"fmt"
	"go.mau.fi/whatsmeow/types/events"
	"mime"
	"strings"
)

func (handler *Handler) handle_message(evt *events.Message) {
	handler.log.Infof("Received message: %#v", evt)
	info := evt.Info
	if purple_get_bool(handler.account, C.GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, false) && info.MessageSource.Chat.ToNonAD().String() == "status@broadcast" {
		handler.log.Warnf("This is a status broadcast. Ignoring message as requested by user settings.")
		return
	}

	message := evt.Message
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
		// note: info.PushName always denotes the sender (not the chat)
		purple_display_text_message(handler.account, info.MessageSource.Chat.ToNonAD().String(), info.MessageSource.IsGroup, info.MessageSource.IsFromMe, info.MessageSource.Sender.ToNonAD().String(), &info.PushName, info.Timestamp, text)
		if !info.MessageSource.IsFromMe { // do not send receipt for own messages
			handler.mark_read_defer(info.ID, info.MessageSource.Chat, info.MessageSource.Sender)
			handler.mark_read_if_on_receival(info.MessageSource.Chat)
		}
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
		data      []byte
		err       error
		filename  = ""
		data_type C.int
		mimetype  *string
	)
	message := evt.Message
	ms := evt.Info.MessageSource
	chat := ms.Chat.ToNonAD().String()

	im := message.GetImageMessage()
	if im != nil {
		data, err = handler.client.Download(im)
		filename = hex.EncodeToString(im.FileSha256) + extension_from_mimetype(im.Mimetype)
		data_type = C.gowhatsapp_attachment_type_image
		mimetype = im.Mimetype
	}
	vm := message.GetVideoMessage()
	if vm != nil {
		data, err = handler.client.Download(vm)
		filename = hex.EncodeToString(vm.FileSha256) + extension_from_mimetype(vm.Mimetype)
		data_type = C.gowhatsapp_attachment_type_video
	}
	am := message.GetAudioMessage()
	if am != nil {
		data, err = handler.client.Download(am)
		filename = hex.EncodeToString(am.FileSha256) + extension_from_mimetype(am.Mimetype)
		data_type = C.gowhatsapp_attachment_type_audio
	}
	dm := message.GetDocumentMessage()
	if dm != nil {
		data, err = handler.client.Download(dm)
		filename = *message.GetDocumentMessage().Title
		// TODO: sanitize filename
		data_type = C.gowhatsapp_attachment_type_document
	}
	sm := message.GetStickerMessage()
	if sm != nil {
		data, err = handler.client.Download(sm)
		filename = hex.EncodeToString(sm.FileSha256) + extension_from_mimetype(sm.Mimetype)
		data_type = C.gowhatsapp_attachment_type_sticker
		mimetype = sm.Mimetype
	}
	if err != nil {
		purple_display_system_message(handler.account, chat, ms.IsGroup, fmt.Sprintf("Message contained an attachment, but the download failed: %#v", err))
		return
	}
	if filename != "" {
		sender := ms.Sender.ToNonAD()
		if ms.IsGroup {
			// put original sender username into file-name
			// so source is known even when receiving from group chats
			filename = fmt.Sprintf("%s_%s", sender.User, filename)
		}
		purple_handle_attachment(handler.account, chat, ms.IsGroup, sender.String(), ms.IsFromMe, data_type, mimetype, filename, data)
	}
}
