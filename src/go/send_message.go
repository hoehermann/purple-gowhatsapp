package main

/*
#include "../c/constants.h"
*/
import "C"

import (
	"bytes"
	"fmt"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"io"
	"net/http"
	"strings"
)

// from https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func parseJID(arg string) (types.JID, error) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), nil
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			return recipient, fmt.Errorf("Invalid JID %s: %v", arg, err)
		} else if recipient.User == "" {
			return recipient, fmt.Errorf("Invalid JID %s: no server specified", arg)
		}
		return recipient, nil
	}
}

/*
 * Sends a plain text message to a recipient.
 *
 * Sending happens asynchronously. Upon success, message is fed back into client.
 */
func (handler *Handler) send_text_message(recipient types.JID, isGroup bool, message string) {
	msg := &waProto.Message{Conversation: &message}
	ts, err := handler.client.SendMessage(recipient, "", msg)
	if err != nil {
		errmsg := fmt.Sprintf("Error sending message: %v", err)
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, errmsg)
	} else {
		// inject message back to self to indicate success
		// spectrum users do not want this
		if !purple_get_bool(handler.account, C.GOWHATSAPP_BRIDGE_COMPATIBILITY_OPTION, false) {
			ownJid := "" // TODO: find out if this messes up group chats
			purple_display_text_message(handler.account, recipient.ToNonAD().String(), isGroup, true, ownJid, nil, ts, message)
		}
	}
}

/*
 * Send a message to a contact.
 *
 * In case message contains nothing but a single http link to a compatible
 * image, audio or video file, the file is sent as a media message.
 * This feature must be enabled explicitly and the file must be small enough.
 *
 * Sending a message causes all previously received messages to become "read" (configurable).
 */
func (handler *Handler) send_message(who string, message string, isGroup bool) {
	handler.log.Infof("Message length: %d", len(message))
	recipient, err := parseJID(who)
	if err != nil {
		purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
	} else {
		if handler.is_link_only_message(message) && handler.send_link_message(recipient, isGroup, message) {
			// nothing to do here
		} else {
			handler.send_text_message(recipient, isGroup, message)
		}
		// I have interacted with this recipient. Mark all messages they have sent as "read".
		handler.mark_read_if_on_answer(recipient)
	}
}

/*
 * Checks wheter message contains a link to a file which can be sent as a
 * media message. Performs HTTP request, checks size and server-supplied mime-type.
 */
func (handler *Handler) is_link_only_message(message string) bool {
	max_file_size := purple_get_int(handler.account, C.GOWHATSAPP_EMBED_MAX_FILE_SIZE_OPTION, 0)
	if max_file_size <= 0 {
		return false
	}
	if !strings.HasPrefix(message, "http") {
		return false
	}
	res, err := http.Head(message)
	if err != nil {
		handler.log.Infof("HTTP HEAD request on '%s' failed with %#v.", message, err)
		return false
	}
	if res.StatusCode != 200 {
		handler.log.Infof("HTTP HEAD request on '%s' returned status code %d.", message, res.StatusCode)
		return false
	}
	switch res.Header.Get("Content-Type") {
	case "image/jpeg", "application/ogg", "audio/ogg", "video/mp4":
		return res.ContentLength <= int64(max_file_size)*1024*1024
	default:
		handler.log.Infof("Server returned content-type '%s' for '%s'. No media type available for this content.", res.Header.Get("Content-Type"), message)
		return false
	}
}

/*
 * Downloads a file given as a HTTP link. Sends it to the recipient as a media message.
 */
func (handler *Handler) send_link_message(recipient types.JID, isGroup bool, link string) bool {
	resp, err := http.Get(link)
	if err != nil {
		handler.log.Infof("Unable to perform HTTP GET request on '%s' due to %v.", link, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		handler.log.Infof("HTTP GET on '%s' returned status code %d.", link, resp.StatusCode)
		return false
	}
	var b bytes.Buffer
	_, err = io.Copy(&b, resp.Body)
	if err != nil {
		handler.log.Infof("Error while downloading file from '%s': %v", link, err)
		return false
	}
	data := b.Bytes()
	var msg *waProto.Message = nil
	switch resp.Header.Get("Content-Type") {
	case "image/jpeg":
		// no checks here
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as image message…")
		msg, err = handler.send_file_image(data, "image/jpeg")
	case "application/ogg", "audio/ogg":
		err = check_ogg(data)
		if err == nil {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as audio message…")
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus")
		} else {
			handler.log.Infof("File incompatible: %s", err)
			return false
		}
	case "video/mp4":
		err = check_mp4(data)
		if err == nil {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as video message…")
			msg, err = handler.send_file_video(data, "video/mp4")
		} else {
			handler.log.Infof("File incompatible: %s", err)
			return false
		}
	default:
		return false
	}
	if err != nil {
		handler.log.Infof("Error while sending file: %s", err)
		return false
	}
	_, err = handler.client.SendMessage(recipient, "", msg)
	if err != nil {
		handler.log.Infof("Error while sending media message: %v", err)
		return false
	} else {
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, fmt.Sprintf("%s has been forwarded.", link))
		return true
	}
}
