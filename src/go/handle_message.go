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
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/net/http2"
)

func (handler *Handler) handle_message(message *waE2E.Message, id string, source types.MessageSource, name *string, timestamp time.Time, is_historical bool) {
	//handler.log.Infof("message: %#v", message)
	if source.Chat == types.StatusBroadcastJID && purple_get_bool(handler.account, C.GOWHATSAPP_IGNORE_STATUS_BROADCAST_OPTION, true) {
		handler.log.Warnf("Ignoring status broadcast.")
		// there have been numerous user reports of status broadcasts crashing the plug-in
		// or other undesired behaviour such as just being annoying
		return
	}
	if handler.blocklist != nil {
		// TODO find out whether locally checking the blocklist is actually necessary or if WhatsApp servers do the filtering for us
		for _, blockedJID := range handler.blocklist.JIDs {
			if blockedJID.ToNonAD() == source.Sender.ToNonAD() {
				handler.log.Infof("Ignoring message from %s since they are on the blocklist.", source.Sender.ToNonAD().String())
			}
		}
	}
	text := ""
	{
		if pm := message.GetProtocolMessage(); pm != nil {
			if em := pm.GetEditedMessage(); em != nil {
				message = em
				text = "[EDIT] "
			}
		}
	}
	text += message.GetConversation()
	{
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

	}
	{
		im := message.GetImageMessage()
		if im != nil && im.Caption != nil {
			text += im.GetCaption()
		}
	}
	{
		vm := message.GetVideoMessage()
		if vm != nil && vm.Caption != nil {
			text += vm.GetCaption()
		}
	}
	{
		ptv := message.GetPtvMessage()
		if ptv != nil && ptv.Caption != nil {
			text += ptv.GetCaption()
		}
	}
	{
		rm := message.GetReactionMessage()
		if rm != nil && rm.Text != nil && rm.Key != nil && rm.Key.ID != nil {
			quote := ""
			for i := range handler.cachedMessages {
				if handler.cachedMessages[i].id == rm.Key.GetID() {
					message := &handler.cachedMessages[i]
					quote = fmt.Sprintf("message \"%.50s\" from %s", message.text, message.timestamp.Format(time.RFC822))
					// TODO: truncate string when storing, not when displaying
					break
				}
			}
			if quote == "" {
				quote = fmt.Sprintf("unknown message with ID %s", rm.Key.GetID())
			}
			if *rm.Text == "" {
				text += fmt.Sprintf("removed their reaction to %s.", quote)
			} else {
				text += fmt.Sprintf("reacted with %s to %s.", *rm.Text, quote)
			}
		}
	}
	if message.GetPollCreationMessage() != nil || message.GetPollCreationMessageV2() != nil || message.GetPollCreationMessageV3() != nil {
		text = "created a poll, but this plug-in cannot display polls."
		// TODO: display poll content
		// TODO: also use GetPollUpdateMessage()
	}
	if text == "" {
		handler.log.Warnf("Received a message without any text.")
	} else {
		// note: info.PushName always denotes the sender (not the chat)
		purple_display_text_message(handler.account, source.Chat.ToNonAD().String(), source.IsGroup, false, source.Sender.ToNonAD().String(), name, timestamp, text, &id)
		handler.addToCache(CachedMessage{id: id, text: text, timestamp: timestamp})
		if !source.IsFromMe && !is_historical { // do not send receipt for own messages or historical messages
			handler.mark_read_defer(id, source.Chat, source.Sender)
			handler.mark_read_if_on_receival(source.Chat)
		}
	}
	handler.handle_attachment(message, id, source, timestamp)
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
func (handler *Handler) handle_attachment(message *waE2E.Message, id string, source types.MessageSource, timestamp time.Time) {
	var (
		data      []byte
		err       error
		filename  = ""
		data_type C.int
		mimetype  *string
	)
	chat := source.Chat.ToNonAD().String()
	{
		im := message.GetImageMessage()
		if im != nil {
			data, err = handler.client.Download(im)
			filename = hex.EncodeToString(im.GetFileSHA256()) + extension_from_mimetype(im.Mimetype)
			data_type = C.gowhatsapp_attachment_type_image
			mimetype = im.Mimetype
		}
	}
	{
		vm := message.GetVideoMessage()
		if vm != nil {
			data, err = handler.client.Download(vm)
			filename = hex.EncodeToString(vm.GetFileSHA256()) + extension_from_mimetype(vm.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
		}
	}
	{
		ptv := message.GetPtvMessage()
		if ptv != nil {
			data, err = handler.client.Download(ptv)
			filename = hex.EncodeToString(ptv.GetFileSHA256()) + extension_from_mimetype(ptv.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
		}
	}
	{
		am := message.GetAudioMessage()
		if am != nil {
			data, err = handler.client.Download(am)
			filename = hex.EncodeToString(am.GetFileSHA256()) + extension_from_mimetype(am.Mimetype)
			data_type = C.gowhatsapp_attachment_type_audio
		}
	}
	{
		dm := message.GetDocumentMessage()
		if dm != nil {
			data, err = handler.client.Download(dm)
			filename = *message.GetDocumentMessage().Title
			// TODO: sanitize filename
			data_type = C.gowhatsapp_attachment_type_document
		}
	}
	{
		sm := message.GetStickerMessage()
		if sm != nil {
			data, err = handler.client.Download(sm)
			filename = hex.EncodeToString(sm.GetFileSHA256()) + extension_from_mimetype(sm.Mimetype)
			data_type = C.gowhatsapp_attachment_type_sticker
			mimetype = sm.Mimetype
		}
	}
	if err != nil {
		if len(data) == 0 {
			errmsg := fmt.Sprintf("Message contained an attachment, but the download failed: %v", err)
			var h2se *http2.StreamError
			if errors.As(err, &h2se) {
				err = h2se.Cause
				errmsg = fmt.Sprintf("%s %v", errmsg, err)
			}
			purple_display_system_message(handler.account, chat, source.IsGroup, errmsg)
			return
		} else {
			handler.log.Warnf("Forwarding file %s to frontend regardless of error: %v", filename, err)
		}
	}
	if filename != "" {
		sender := source.Sender.ToNonAD().String()
		directory := purple_get_string(handler.account, C.GOWHATSAPP_ATTACHMENT_DIRECTORY_OPTION, C.GOWHATSAPP_ATTACHMENT_DIRECTORY_DEFAULT)
		if directory != "" {
			local_path := filepath.Join(directory, chat, filename)
			os.MkdirAll(filepath.Join(directory, chat), os.ModePerm)
			file, err := os.Create(local_path)
			if err != nil {
				errmsg := fmt.Sprintf("Unable to store file at %s due to %v", local_path, err)
				purple_display_system_message(handler.account, chat, source.IsGroup, errmsg)
			} else {
				file.Write(data)
				file.Close()
				url_prefix := purple_get_string(handler.account, C.GOWHATSAPP_ATTACHMENT_BASE_URL_OPTION, C.GOWHATSAPP_ATTACHMENT_BASE_URL_DEFAULT)
				url := fmt.Sprintf("%s/%s/%s", url_prefix, chat, filename)
				if url_prefix == "" {
					local_path, _ := filepath.Abs(local_path)
					url = "file://" + local_path
				}
				text := url
				purple_display_text_message(handler.account, chat, source.IsGroup, false, sender, nil, timestamp, text, &id)
			}
		} else {
			purple_handle_attachment(handler.account, chat, source.IsGroup, sender, false, data_type, mimetype, filename, data, id)
		}
	}
}
