package main

/*
#include "../c/constants.h"
#include "../c/opusreader.h"
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
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
 * Sending may happen asynchronously.
 *
 * Upon success, message is fed back into client iff
 * GOWHATSAPP_ECHO_OPTION is set to GOWHATSAPP_ECHO_CHOICE_ON_SUCCESS.
 *
 * Returns true on success.
 */
func (handler *Handler) send_text_message(recipient types.JID, isGroup bool, message string) bool {
	msg := &waProto.Message{Conversation: &message}
	expiration_days := purple_get_int(handler.account, C.GOWHATSAPP_EXPIRATION_OPTION, 0)
	if expiration_days > 0 {
		expiration_seconds := uint32(expiration_days) * 24 * 60 * 60
		msg = &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: &message,
				ContextInfo: &waProto.ContextInfo{
					Expiration: proto.Uint32(expiration_seconds),
				},
			},
		}
	}
	resp, err := handler.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		errmsg := fmt.Sprintf("Error sending message: %v", err)
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, errmsg)
		return false
	} else {
		// inject message back to self to indicate success
		setting := purple_get_string(handler.account, C.GOWHATSAPP_ECHO_OPTION, C.GOWHATSAPP_ECHO_CHOICE_ON_SUCCESS)
		if setting == C.GoString(C.GOWHATSAPP_ECHO_CHOICE_ON_SUCCESS) {
			ownJid := handler.client.Store.ID.ToNonAD().String()
			recipientJid := recipient.ToNonAD().String()
			purple_display_text_message(handler.account, recipientJid, isGroup, true, ownJid, nil, resp.Timestamp, message)
		}
		handler.addToCache(CachedMessage{id: resp.ID, text: message, timestamp: resp.Timestamp})
		return true
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
 *
 * Returns true on success.
 */
func (handler *Handler) send_message(who string, message string, isGroup bool) bool {
	recipient, err := parseJID(who)
	if err != nil {
		purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
		return false
	} else {
		// I am interacting with this recipient. Mark all messages they have sent as "read".
		handler.mark_read_if_on_answer(recipient)
		// now do the actual sending
		if handler.is_link_only_message(message) {
			// this is a link-only message – try to send the linked file, if compatible
			if handler.send_link_message(recipient, isGroup, message) {
				return true
			} else {
				// sending the link message failed. just send as a normal text message
				return handler.send_text_message(recipient, isGroup, message)
			}
		} else {
			// this is a normal message
			return handler.send_text_message(recipient, isGroup, message)
		}
	}
}

func (handler *Handler) check_url_trust(url string) bool {
	trusted_url_regex := purple_get_string(handler.account, C.GOWHATSAPP_TRUSTED_URL_REGEX_OPTION, C.GOWHATSAPP_TRUSTED_URL_REGEX_DEFAULT)
	if trusted_url_regex != "" {
		matched, err := regexp.MatchString(trusted_url_regex, url)
		if err != nil {
			handler.log.Errorf("Checking URL trust failed due to %v.", err)
			return false
		}
		return matched
	}
	return false
}

/*
 * Checks wheter message contains a link to a file which can be sent as a
 * media message. Performs HTTP request, checks size.
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
	return res.ContentLength <= int64(max_file_size)*1024*1024
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
	mimetype := http.DetectContentType(data) // do not trust the server. he is stupid.
	switch mimetype {
	case "image/jpeg":
		// send jpeg as ImageMessage
		// no checks here
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as image message…")
		msg, err = handler.send_file_image(data, "image/jpeg")
	case "application/ogg", "audio/ogg":
		// send ogg file as AudioMessage
		seconds := int64(C.opus_get_seconds(C.CBytes(data), C.size_t(len(data))))
		if seconds >= 0 {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as audio message…")
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus", uint32(seconds))
		} else {
			handler.log.Infof("An ogg audio file was provided, but it was invalid.", err)
			return false
		}
	case "video/mp4":
		// send mp4 file as VideoMessage
		err = check_mp4(data)
		if err == nil {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, "Compatible file detected. Forwarding as video message…")
			msg, err = handler.send_file_video(data, "video/mp4")
		} else {
			handler.log.Infof("File incompatible: %s", err)
			return false
		}
	default:
		// send any other file type as DocumentMessage
		if handler.check_url_trust(link) {
			filename := filepath.Base(resp.Request.URL.Path)
			msg, err = handler.send_file_document(data, mimetype, filename)
		} else {
			return false
		}
	}
	if err != nil {
		handler.log.Infof("Error while sending file: %s", err)
		return false
	}
	send_response, err := handler.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		handler.log.Infof("Error while sending media message: %v", err)
		return false
	} else {
		purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, fmt.Sprintf("%s has been forwarded.", link))
		handler.addToCache(CachedMessage{id: send_response.ID, text: link, timestamp: send_response.Timestamp})
		return true
	}
}
