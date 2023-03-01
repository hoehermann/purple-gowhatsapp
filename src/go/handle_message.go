package main

/*
#include "../c/constants.h"
#include "../c/bridge.h"
*/
import "C"

import (
	"encoding/hex"
	"errors"
	"fmt"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/net/http2"
	"mime"
	"strings"
	"time"
)

func (handler *Handler) handle_message(message *waProto.Message, id string, source types.MessageSource, name *string, timestamp time.Time, is_historical bool) {
	//handler.log.Infof("message: %#v", message)
	if source.Chat == types.StatusBroadcastJID && purple_get_bool(handler.account, C.GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, true) {
		handler.log.Warnf("Ignoring status broadcast.")
		// there have been numerous user reports of status broadcasts crashing the plug-in
		// or other undesired behaviour such as just being annoying
		return
	}
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

	rm := message.GetReactionMessage()
	if rm != nil && rm.Text != nil && rm.Key != nil && rm.Key.Id != nil {
		quote := ""
		for i := range handler.cachedMessages {
			if handler.cachedMessages[i].id == *rm.Key.Id {
				message := &handler.cachedMessages[i]
				quote = fmt.Sprintf("message \"%.50s\" from %s", message.text, message.timestamp.Format(time.RFC822))
				// TODO: truncate string when storing, not when displaying
				break
			}
		}
		if quote == "" {
			quote = fmt.Sprintf("unknown message with ID %s", *rm.Key.Id)
		}
		if *rm.Text == "" {
			text += fmt.Sprintf("removed their reaction to %s.", quote)
		} else {
			text += fmt.Sprintf("reacted with %s to %s.", *rm.Text, quote)
		}
	}

	if text == "" {
		handler.log.Warnf("Received a message without any text.")
	} else {
		// note: info.PushName always denotes the sender (not the chat)
		purple_display_text_message(handler.account, source.Chat.ToNonAD().String(), source.IsGroup, source.IsFromMe, source.Sender.ToNonAD().String(), name, timestamp, text)
		handler.addToCache(CachedMessage{id: id, text: text, timestamp: timestamp})
		if !source.IsFromMe && !is_historical { // do not send receipt for own messages or historical messages
			handler.mark_read_defer(id, source.Chat, source.Sender)
			handler.mark_read_if_on_receival(source.Chat)
		}
	}

	handler.handle_attachment(message, source)
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
func (handler *Handler) handle_attachment(message *waProto.Message, source types.MessageSource) {
	var (
		data      []byte
		err       error
		filename  = ""
		data_type C.int
		mimetype  *string
	)
	chat := source.Chat.ToNonAD().String()

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
		if data == nil || len(data) == 0 {
			errmsg := fmt.Sprintf("Message contained an attachment, but the download failed: %v", err)
			var h2se *http2.StreamError
			if errors.As(err, &h2se) {
				err = h2se.Cause
			}
			purple_display_system_message(handler.account, chat, source.IsGroup, errmsg)
			return
		} else {
			handler.log.Warnf("Forwarding file %s to frontend regardless of error: %v", filename, err)
		}
	}
	if filename != "" {
		sender := source.Sender.ToNonAD()
		if source.IsGroup {
			// put original sender username into file-name
			// so source is known even when receiving from group chats
			filename = fmt.Sprintf("%s_%s", sender.User, filename)
		}
		purple_handle_attachment(handler.account, chat, source.IsGroup, sender.String(), source.IsFromMe, data_type, mimetype, filename, data)
	}
}
