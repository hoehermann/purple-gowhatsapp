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
	"net/url"
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
		filename          = ""
		hash              = ""
		extension         = ""
		data_type C.int   = C.gowhatsapp_attachment_type_none
		mimetype  *string = nil
	)
	chat := source.Chat.ToNonAD().String()
	{
		im := message.GetImageMessage()
		if im != nil {
			data, err = handler.client.Download(im)
			hash = hex.EncodeToString(im.GetFileSHA256())
			extension = extension_from_mimetype(im.Mimetype)
			data_type = C.gowhatsapp_attachment_type_image
			mimetype = im.Mimetype
		}
	}
	{
		vm := message.GetVideoMessage()
		if vm != nil {
			data, err = handler.client.Download(vm)
			hash = hex.EncodeToString(vm.GetFileSHA256())
			extension = extension_from_mimetype(vm.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
		}
	}
	{
		ptv := message.GetPtvMessage()
		if ptv != nil {
			data, err = handler.client.Download(ptv)
			hash = hex.EncodeToString(ptv.GetFileSHA256())
			extension = extension_from_mimetype(ptv.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
		}
	}
	{
		am := message.GetAudioMessage()
		if am != nil {
			data, err = handler.client.Download(am)
			hash = hex.EncodeToString(am.GetFileSHA256())
			extension = extension_from_mimetype(am.Mimetype)
			data_type = C.gowhatsapp_attachment_type_audio
		}
	}
	{
		dm := message.GetDocumentMessage()
		if dm != nil {
			data, err = handler.client.Download(dm)
			hash = hex.EncodeToString(dm.GetFileSHA256())
			filename = dm.GetFileName() // TODO: sanitize filename
			extension = filepath.Ext(filename)
			if extension == "" {
				extension = extension_from_mimetype(dm.Mimetype)
			}
			filename = strings.TrimSuffix(filename, extension)
			data_type = C.gowhatsapp_attachment_type_document
		}
	}
	{
		sm := message.GetStickerMessage()
		if sm != nil {
			data, err = handler.client.Download(sm)
			hash = hex.EncodeToString(sm.GetFileSHA256())
			extension = extension_from_mimetype(sm.Mimetype)
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
	if data_type != C.gowhatsapp_attachment_type_none {
		sender := source.Sender.ToNonAD().String()
		local_path_template := purple_get_string(handler.account, C.GOWHATSAPP_ATTACHMENT_PATH_TEMPLATE_OPTION, C.GOWHATSAPP_ATTACHMENT_PATH_TEMPLATE_DEFAULT)
		if local_path_template != "" {
			local_path := local_path_template
			local_path = strings.Replace(local_path, "$remote", chat, -1)
			local_path = strings.Replace(local_path, "$hash", hash, -1)
			local_path = strings.Replace(local_path, "$filename", filename, -1)
			local_path = strings.Replace(local_path, "$extension", extension, -1)
			os.MkdirAll(filepath.Dir(local_path), os.ModePerm)
			file, err := os.Create(local_path)
			if err != nil {
				errmsg := fmt.Sprintf("Unable to store file at %s due to %v", local_path, err)
				purple_display_system_message(handler.account, chat, source.IsGroup, errmsg)
			} else {
				file.Write(data)
				file.Close()
				url_template := purple_get_string(handler.account, C.GOWHATSAPP_ATTACHMENT_URL_TEMPLATE_OPTION, C.GOWHATSAPP_ATTACHMENT_URL_TEMPLATE_DEFAULT)
				url_s := ""
				if url_template == "" {
					u := url.URL{Scheme: "file", Path: local_path} // TODO: use url.FromFilePath(path)
					url_s = u.String()
				} else {
					url_s = url_template
					url_s = strings.Replace(url_s, "$remote", url.PathEscape(chat), -1)
					url_s = strings.Replace(url_s, "$hash", hash, -1)
					url_s = strings.Replace(url_s, "$filename", url.PathEscape(filename), -1)
					url_s = strings.Replace(url_s, "$extension", extension, -1)
				}
				text := url_s
				purple_display_text_message(handler.account, chat, source.IsGroup, false, sender, nil, timestamp, text, &id)
			}
		} else {
			// append extension to file-name (relevant on Windows in particular)
			if data_type == C.gowhatsapp_attachment_type_document {
				// only Document messages offer named files
				filename = filename + extension
			} else {
				// use hash and extension for all the other attachment types
				filename = hash + extension
			}
			purple_handle_attachment(handler.account, chat, source.IsGroup, sender, false, data_type, mimetype, filename, data, id)
		}
	}
}
