/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2019 Hermann Höhne
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

/*
 * Apparently, the main package must be named main, even though this is a library
 */
package main

/*
#cgo LDFLAGS: gwa-to-purple.o
#cgo CFLAGS: -DCGO

#include <stdint.h>
#include <time.h>
#include <unistd.h>

#include "constants.h"
#include "proxy.h"

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_session,
    gowhatsapp_message_type_contactlist_refresh,
    gowhatsapp_message_type_presence,
};

// C compatible representation of one received message.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    uintptr_t connection; /// an int representation of the purple connection pointer
    int64_t msgtype; /// message type – see above
    char *id; /// message id
    char *remoteJid; /// conversation identifier (may be a singel contact or a group)
    char *senderJid; /// message author's identifier (useful in group chats)
    char *text; /// the message payload (interpretation depends on type)
    void *blob; /// binary payload (used for inlining images)
    size_t blobsize; /// size of binary payload in bytes
    time_t timestamp; /// timestamp the message was sent(?)
    char fromMe; /// this is (a copy of) an outgoing message
    char system; /// this is a system-message, not user-generated
    // these are only used for transporting session data
    char *clientId;
    char *clientToken;
    char *serverToken;
    char *encKey_b64;
    char *macKey_b64;
    char *wid;
};

extern void gowhatsapp_process_message_bridge(void * gwamsg);
extern void * gowhatsapp_get_account(uintptr_t pc);
extern int gowhatsapp_account_get_bool(void *account, const char *name, int default_value);
extern const char * gowhatsapp_account_get_string(void *account, const char *name, const char *default_value);
extern const PurpleProxyInfo * gowhatsapp_account_get_proxy(void *account);
*/
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unsafe"
	"os"

	"github.com/Rhymen/go-whatsapp"
	"github.com/pkg/errors"
	"github.com/skip2/go-qrcode"
)

/*
 * Holds all data of one message which is going to be exposed
 * to the C part ("front-end") of the plug-in. However, it is converted
 * to a C compatible gowhatsapp_message type first.
 */
type MessageAggregate struct {
	info    whatsapp.MessageInfo
	text    string
	session *whatsapp.Session
	data    []byte
	err     error
	system  bool
}

/*
 * Holds all data for one account (connection) instance.
 */
type waHandler struct {
	wac                *whatsapp.Conn
	downloadsDirectory string
	connID             C.uintptr_t
}

/*
 * This plug-in can handle multiple connections (identified by C pointer adresses).
 */
var waHandlers = make(map[C.uintptr_t]*waHandler)

/*
 * Send a gereric message.
 * Forward error message to front-end.
 */
func (handler *waHandler) sendMessage(message interface{}, info whatsapp.MessageInfo) *C.char {
	msgId, err := handler.wac.Send(message)
	if err != nil {
		handler.presentMessage(makeConversationErrorMessage(info,
			fmt.Sprintf("Unable to send message: %v", err)))
		return nil
	}
	return C.CString(msgId)
}

/*
 * Send a text message.
 * Called by the front-end.
 */
//export gowhatsapp_go_sendMessage
func gowhatsapp_go_sendMessage(connID C.uintptr_t, who *C.char, text *C.char) *C.char {
	remoteJid := C.GoString(who)
	if remoteJid == "login@s.whatsapp.net" {
		return nil
	}
	handler := waHandlers[connID]
	info := whatsapp.MessageInfo{
		RemoteJid: remoteJid,
	}
	message := whatsapp.TextMessage{
		Info: info,
		Text: C.GoString(text),
	}
	return handler.sendMessage(message, info)
}

/*
 * Send a media message.
 * Called by the front-end.
 */
//export gowhatsapp_go_sendMedia
func gowhatsapp_go_sendMedia(connID C.uintptr_t, who *C.char, filename *C.char) *C.char {
	remoteJid := C.GoString(who)
	if remoteJid == "login@s.whatsapp.net" {
		return nil
	}
	handler := waHandlers[connID]
	info := whatsapp.MessageInfo{
		RemoteJid: remoteJid,
	}
	return handler.sendMediaMessage(info, C.GoString(filename))
}

// TODO: find out how to enable C99's bool type in cgo
func bool_to_Cchar(b bool) C.char {
	if b {
		return C.char(1)
	} else {
		return C.char(0)
	}
}

func Cint_to_bool(i C.int) bool {
	return i != 0
}

/*
 * Forwards a message to the front-end.
 */
func (handler *waHandler) presentMessage(message MessageAggregate) {
	//fmt.Fprintf(os.Stderr, "presentMessage(%v)\n", message)
	if message.err == nil && message.session == nil && !message.system {
		handler.wac.Read(message.info.RemoteJid, message.info.Id) // mark message as "displayed"
	}
	cmessage := convertMessage(handler.connID, message)
	C.gowhatsapp_process_message_bridge(unsafe.Pointer(&cmessage))
}

/*
 * Converts a message from go structs to a unified C struct.
 */
func convertMessage(connID C.uintptr_t, message MessageAggregate) C.struct_gowhatsapp_message {
	if message.err != nil {
		// thanks to https://stackoverflow.com/questions/39023475/
		return C.struct_gowhatsapp_message{
			connection: connID,
			msgtype:    C.int64_t(C.gowhatsapp_message_type_error),
			text:       C.CString(message.err.Error()),
		}
	}
	if message.session != nil {
		return C.struct_gowhatsapp_message{
			connection:  connID,
			remoteJid:   C.CString("login@s.whatsapp.net"),
			system:      bool_to_Cchar(true),
			msgtype:     C.int64_t(C.gowhatsapp_message_type_session),
			clientId:    C.CString(message.session.ClientId),
			clientToken: C.CString(message.session.ClientToken),
			serverToken: C.CString(message.session.ServerToken),
			encKey_b64:  C.CString(base64.StdEncoding.EncodeToString(message.session.EncKey)),
			macKey_b64:  C.CString(base64.StdEncoding.EncodeToString(message.session.MacKey)),
			wid:         C.CString(message.session.Wid),
		}
	}
	C_message_type := C.gowhatsapp_message_type_text
	info := message.info
	if info.Id == "login" { /* TODO: do not abuse Id to distinguish message type */
		C_message_type = C.gowhatsapp_message_type_login
	}
	if message.system {
		info.Id = ""
	}
	if (info.Source != nil && info.Source.Participant != nil) {
		info.SenderJid = *info.Source.Participant
	}
	return C.struct_gowhatsapp_message{
		connection: connID,
		msgtype:    C.int64_t(C_message_type),
		timestamp:  C.time_t(info.Timestamp),
		id:         C.CString(info.Id),
		remoteJid:  C.CString(info.RemoteJid),
		senderJid:  C.CString(info.SenderJid), /* Note: info.SenderJid seems to be nil or empty most of the time */
		fromMe:     bool_to_Cchar(info.FromMe),
		text:       C.CString(strings.Replace(message.text,"\n"," \n", -1)),
		system:     bool_to_Cchar(message.system),
		blob:       C.CBytes(message.data),
		blobsize:   C.size_t(len(message.data)), // contrary to https://golang.org/pkg/builtin/#len and https://golang.org/ref/spec#Numeric_types, len returns an int of 64 bits on 32 bit Windows machines (see https://github.com/hoehermann/purple-gowhatsapp/issues/1)
	}
}

/*
 * Handles errors reported by go-whatsapp.
 * Errors will likely cause the front-end to destroy the connection.
 */
func (handler *waHandler) HandleError(err error) {
	cause := errors.Cause(err)
	if cause == whatsapp.ErrInvalidWsData ||
		cause == whatsapp.ErrInvalidWsState ||
		strings.Contains(err.Error(), "invalid string with tag 174") { // TODO: less ugly error comparison
		// these errors are not actually errors, rather than warnings
		//fmt.Fprintf(os.Stderr, "gowhatsapp: %v ignored.\n", err)
	} else {
		handler.presentMessage(MessageAggregate{err: err})
	}
}

/*
 * Creates an error message as reaction to an actual message.
 */
func makeConversationErrorMessage(originalInfo whatsapp.MessageInfo, errorMessage string) MessageAggregate {
	return MessageAggregate{system: true, info: originalInfo, text: errorMessage}
}

func (handler *waHandler) handleDownloadableMessage(message DownloadableMessage, info whatsapp.MessageInfo, inline bool) []byte {
	downloadsEnabled := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_DOWNLOAD_ATTACHMENTS_OPTION, 0))
	storeFailedDownload := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_DOWNLOAD_TRY_ONLY_ONCE_OPTION, 1))
	return handler.presentDownloadableMessage(message, info, downloadsEnabled, storeFailedDownload, inline)
}

func (handler *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	handler.presentMessage(MessageAggregate{text: message.Text, info: message.Info})
}

func (handler *waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
	inlineImages := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_INLINE_IMAGES_OPTION, 1))
	data := handler.handleDownloadableMessage(DownloadableMessage{Message: &message, Type: message.Type}, message.Info, inlineImages)
	handler.presentMessage(MessageAggregate{text: message.Caption, info: message.Info, data: data})
}

func (handler *waHandler) HandleStickerMessage(message whatsapp.StickerMessage) {
	inlineImages := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_INLINE_IMAGES_OPTION, 0))
	data := handler.handleDownloadableMessage(DownloadableMessage{Message: &message, Type: message.Type}, message.Info, inlineImages)
	if inlineImages {
		handler.presentMessage(MessageAggregate{text: "Contact sent a sticker: ", info: message.Info, data: data})
	}
}

func (handler *waHandler) HandleVideoMessage(message whatsapp.VideoMessage) {
	handler.presentMessage(MessageAggregate{text: message.Caption, info: message.Info})
	handler.handleDownloadableMessage(DownloadableMessage{Message: &message, Type: message.Type}, message.Info, false)
}

func (handler *waHandler) HandleAudioMessage(message whatsapp.AudioMessage) {
	handler.handleDownloadableMessage(DownloadableMessage{Message: &message, Type: message.Type}, message.Info, false)
}

func (handler *waHandler) HandleDocumentMessage(message whatsapp.DocumentMessage) {
	handler.handleDownloadableMessage(DownloadableMessage{Message: &message, Type: message.Type}, message.Info, false)
}

func (handler *waHandler) HandleContactList(contacts []whatsapp.Contact) {
	for _, contact := range contacts {
		C_message_type := C.gowhatsapp_message_type_contactlist_refresh

		cmessage := C.struct_gowhatsapp_message{
			connection: handler.connID,
			msgtype:    C.int64_t(C_message_type),
			remoteJid:  C.CString(contact.Jid),
			senderJid:  C.CString(contact.Notify),
			text:       C.CString(contact.Name),
		}
		C.gowhatsapp_process_message_bridge(unsafe.Pointer(&cmessage))
		handler.wac.SubscribePresence(contact.Jid)
	}
}

type JSONMessage []json.RawMessage
type JSONMessageType string

const (
	MessageMsgInfo   JSONMessageType = "MsgInfo"
	MessageMsg       JSONMessageType = "Msg"
	MessagePresence  JSONMessageType = "Presence"
	MessageStream    JSONMessageType = "Stream"
	MessageConn      JSONMessageType = "Conn"
	MessageProps     JSONMessageType = "Props"
	MessageCmd       JSONMessageType = "Cmd"
	MessageChat      JSONMessageType = "Chat"
	MessageCall      JSONMessageType = "Call"
	MessageBlocklist JSONMessageType = "Blocklist"
)

type Presence struct {
	JID       string `json:"id"`
	Status    string `json:"type"`
	Timestamp int64  `json:"t"`
	Deny      bool   `json:"deny"`
}

const (
	OldUserSuffix = "@c.us"
	NewUserSuffix = "@s.whatsapp.net"
)

func (handler *waHandler) handleMessagePresence(message []byte) {
	var event Presence
	err := json.Unmarshal(message, &event)
	if err != nil {
		return
	}
	event.JID = strings.Replace(event.JID, OldUserSuffix, NewUserSuffix, 1)

	C_message_type := C.gowhatsapp_message_type_presence

	cmessage := C.struct_gowhatsapp_message{
		connection: handler.connID,
		msgtype:    C.int64_t(C_message_type),
		timestamp:  C.time_t(event.Timestamp),
		remoteJid:  C.CString(event.JID),
		fromMe:     bool_to_Cchar(event.Deny),
		text:       C.CString(event.Status),
		system:     bool_to_Cchar(true),
	}
	C.gowhatsapp_process_message_bridge(unsafe.Pointer(&cmessage))
}

func (handler *waHandler) HandleJsonMessage(message string) {
	msg := JSONMessage{}
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil || len(msg) < 2 {
		return
	}

	var msgType JSONMessageType
	json.Unmarshal(msg[0], &msgType)

	switch msgType {
	case MessagePresence:
		handler.handleMessagePresence(msg[1])
	case MessageStream:
	case MessageConn:
	case MessageProps:
	case MessageMsgInfo, MessageMsg:
	case MessageCmd:
	case MessageChat:
	case MessageCall:
	case MessageBlocklist:
	default:
		//fmt.Fprintf(os.Stderr, "gowhatsapp: Unhandled JsonMessage(%v)\n", message)
	}
}

//TBD
func (handler *waHandler) HandleContactMessage(message whatsapp.ContactMessage) {
	//fmt.Fprintf(os.Stderr, "HandleContactMessage(%v)\n", message)
}

type ProfilePicInfo struct {
	URL string `json:"eurl"`
	Tag string `json:"tag"`

	Status int16 `json:"status"`
}

//export gowhatsapp_get_icon_url
func gowhatsapp_get_icon_url(connID C.uintptr_t, who *C.char) *C.char {
	handler, ok := waHandlers[connID]
	if ok {
		jid := C.GoString(who)
		data, err := handler.wac.GetProfilePicThumb(jid)
		if err != nil {
			return nil
		}
		content := <-data
		info := &ProfilePicInfo{}
		err = json.Unmarshal([]byte(content), info)
		if err != nil {
			return nil
		}
		if info.Status == 0 {
			return C.CString(info.URL)
		}
	}
	return nil
}

func connect_and_login(handler *waHandler, session *whatsapp.Session) {
	//create new WhatsApp connection
	timeout := 10 * time.Second // TODO: make timeout user configurable
	proxyInfo := C.gowhatsapp_account_get_proxy(C.gowhatsapp_get_account(handler.connID))
	var err error
	var wac *whatsapp.Conn
	if proxyInfo != nil && proxyInfo._type > 0 {
		scheme := ""
		switch proxyInfo._type {
		case C.PURPLE_PROXY_HTTP:
			scheme = "http" // no idea what to do for https proxies
		case C.PURPLE_PROXY_SOCKS4:
			scheme = "socks"
		case C.PURPLE_PROXY_SOCKS5:
			scheme = "socks5"
		case C.PURPLE_PROXY_USE_ENVVAR:
			// TODO
			//func ProxyFromEnvironment(req *Request) (*url.URL, error)
			fmt.Fprintf(os.Stderr, "gowhatsapp: Proxy settings according to environment not implemented.\n")
		case C.PURPLE_PROXY_TOR:
			scheme = "socks5" // no idea if this is correct
		}
		proxyUrl := url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", C.GoString(proxyInfo.host), proxyInfo.port),
			User:   url.UserPassword(C.GoString(proxyInfo.username), C.GoString(proxyInfo.password)),
		}
		fmt.Fprintf(os.Stderr, "gowhatsapp: Using proxy URL: %v.\n", proxyUrl)
		proxy := http.ProxyURL(&proxyUrl)
		wac, err = whatsapp.NewConnWithProxy(timeout, proxy)
	} else {
		wac, err = whatsapp.NewConn(timeout)
	}
	handler.wac = wac
	if err != nil {
		wac = nil
		handler.presentMessage(MessageAggregate{err: err, system: true})
	} else {
		wac.AddHandler(handler)
		err = login(handler, session)
		if err != nil {
			wac = nil
			handler.presentMessage(MessageAggregate{err: err, system: true})
		}
	}
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(
	connID C.uintptr_t,
	restoreSession C.int,
	downloadsDirectory *C.char,
) {
	// TODO: protect against concurrent invocation
	handler := waHandler{
		wac:                nil,
		downloadsDirectory: C.GoString(downloadsDirectory),
		connID:             connID,
	}
	waHandlers[connID] = &handler
	var session *whatsapp.Session
	if Cint_to_bool(restoreSession) {
		clientId := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_CLIENDID_KEY, nil)
		clientToken := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_CLIENTTOKEN_KEY, nil)
		serverToken := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_SERVERTOKEN_KEY, nil)
		encKey_b64 := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_ENCKEY_KEY, nil)
		macKey_b64 := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_MACKEY_KEY, nil)
		wid := C.gowhatsapp_account_get_string(C.gowhatsapp_get_account(connID), C.GOWHATSAPP_SESSION_WID_KEY, nil)
		if clientId != nil && clientToken != nil && serverToken != nil && encKey_b64 != nil && macKey_b64 != nil && wid != nil {
			encKey, encKeyErr := base64.StdEncoding.DecodeString(C.GoString(encKey_b64))
			macKey, macKeyErr := base64.StdEncoding.DecodeString(C.GoString(macKey_b64))
			if encKeyErr == nil && macKeyErr == nil {
				session = &whatsapp.Session{
					ClientId:    C.GoString(clientId),
					ClientToken: C.GoString(clientToken),
					ServerToken: C.GoString(serverToken),
					EncKey:      encKey,
					MacKey:      macKey,
					Wid:         C.GoString(wid)}
			}
		}
	}
	go connect_and_login(&handler, session)
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(connID C.uintptr_t) {
	handler, ok := waHandlers[connID]
	if ok {
		if handler.wac != nil {
			handler.wac.Disconnect()
		}
		delete(waHandlers, connID)
	}
}

func login(handler *waHandler, login_session *whatsapp.Session) error {
	wac := handler.wac
	if login_session != nil {
		//restore session
		session, err := wac.RestoreWithSession(*login_session)
		if err != nil {
			return fmt.Errorf("restoring failed: %v\n", err)
			// NOTE: "restore session connection timed out" may indicate phone switched off OR session data being invalid (after log-out via phone) O_o
		}
		handler.presentMessage(MessageAggregate{session: &session})
	} else {
		//no saved session -> login via qr code
		qr := make(chan string)
		go func() {
			qr_data := <-qr
			png, err := qrcode.Encode(qr_data, qrcode.Medium, 256) // TODO: make size user configurable
			if err != nil {
				handler.presentMessage(MessageAggregate{err: fmt.Errorf("login qr code generation failed: %v\n", err)})
			} else {
				messageInfo := whatsapp.MessageInfo{
					RemoteJid: "login@s.whatsapp.net",
					Id:        "login"}
				handler.presentMessage(MessageAggregate{
					text:   qr_data,
					info:   messageInfo,
					system: true,
					data:   png})
			}
		}()
		session, err := wac.Login(qr)
		if err != nil {
			return fmt.Errorf("error during login: %v\n", err)
		}
		// login successful. forward session data to front-end:
		handler.presentMessage(MessageAggregate{session: &session})
	}
	return nil
}

func main() {
	gowhatsapp_go_login(0, 0, C.CString("."))
	<-time.After(1 * time.Minute)
	gowhatsapp_go_close(0)
}
