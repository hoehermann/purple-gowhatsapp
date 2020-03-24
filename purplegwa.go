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
#include <stdint.h>
#include <time.h>
#include <unistd.h>

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_login,
    gowhatsapp_message_type_session,
};

// C compatible representation of one received message.
// NOTE: If the cgo and gcc compilers disagree on padding or alignment, chaos will ensue.
struct gowhatsapp_message {
    uintptr_t connection;
    int64_t msgtype;
    char *id;
    char *remoteJid;
    char *senderJid;
    char *text;
    void *blob;
    size_t blobsize;
    time_t timestamp;
    char fromMe;
    char system;
    // these are only used for transporting session data
    char *clientId;
    char *clientToken;
    char *serverToken;
    char *encKey_b64;
    char *macKey_b64;
    char *wid;
};

struct gowhatsapp_session {
};

extern void gowhatsapp_process_message_bridge(void * gwamsg);
extern void * gowhatsapp_get_account(uintptr_t pc);
extern int gowhatsapp_account_get_bool(void *account, const char *name, int default_value);
extern const char * gowhatsapp_account_get_string(void *account, const char *name, const char *default_value);
#cgo LDFLAGS: gwa-to-purple.o

#cgo CFLAGS: -DCGO
#include "constants.h"
*/
import "C"

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unsafe"

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
	gotext := C.GoString(text)
	if strings.HasPrefix(gotext, "/sendmedia") {
		return handler.sendMediaMessage(info, gotext)
	} else {
		message := whatsapp.TextMessage{
			Info: info,
			Text: gotext,
		}
		return handler.sendMessage(message, info)
	}
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
	return C.struct_gowhatsapp_message{
		connection: connID,
		msgtype:    C.int64_t(C_message_type),
		timestamp:  C.time_t(info.Timestamp),
		id:         C.CString(info.Id),
		remoteJid:  C.CString(info.RemoteJid),
		senderJid:  C.CString(info.SenderJid),
		fromMe:     bool_to_Cchar(info.FromMe),
		text:       C.CString(message.text),
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

func (handler *waHandler) handleDownloadableMessage(message downloadable, info whatsapp.MessageInfo, inline bool) []byte {
	downloadsEnabled := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_DOWNLOAD_ATTACHMENTS_OPTION, 0))
	storeFailedDownload := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_DOWNLOAD_TRY_ONLY_ONCE_OPTION, 1))
	return handler.presentDownloadableMessage(message, info, downloadsEnabled, storeFailedDownload, inline)
}

func (handler *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	handler.presentMessage(MessageAggregate{text: message.Text, info: message.Info})
}

func (handler *waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
	inlineImages := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_INLINE_IMAGES_OPTION, 1))
	data := handler.handleDownloadableMessage(&message, message.Info, inlineImages)
	handler.presentMessage(MessageAggregate{text: message.Caption, info: message.Info, data: data})
}

func (handler *waHandler) HandleStickerMessage(message whatsapp.StickerMessage) {
	inlineImages := Cint_to_bool(C.gowhatsapp_account_get_bool(C.gowhatsapp_get_account(handler.connID), C.GOWHATSAPP_INLINE_IMAGES_OPTION, 0))
	data := handler.handleDownloadableMessage(&message, message.Info, inlineImages)
	if inlineImages {
		handler.presentMessage(MessageAggregate{text: "Contact sent a sticker: ", info: message.Info, data: data})
	}
}

func (handler *waHandler) HandleVideoMessage(message whatsapp.VideoMessage) {
	handler.presentMessage(MessageAggregate{text: message.Caption, info: message.Info})
	handler.handleDownloadableMessage(&message, message.Info, false)
}

func (handler *waHandler) HandleAudioMessage(message whatsapp.AudioMessage) {
	handler.handleDownloadableMessage(&message, message.Info, false)
}

func (handler *waHandler) HandleDocumentMessage(message whatsapp.DocumentMessage) {
	handler.handleDownloadableMessage(&message, message.Info, false)
}

func connect_and_login(handler *waHandler, session *whatsapp.Session) {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(
		10 * time.Second, // TODO: make timeout user configurable
	//"github.com/kaxap/go-whatsapp", "Pidgin via go-whatsapp"
	)
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
