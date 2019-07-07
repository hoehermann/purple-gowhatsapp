// package name: purplegwa
package main

/*
#include <stdint.h>
#include <time.h>

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_session,
};

struct gowhatsapp_message {
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
*/
import "C"

import (
    "encoding/base64"
	"fmt"
	"os"
	"time"
	"strings"

	"github.com/Rhymen/go-whatsapp"
	"github.com/skip2/go-qrcode"
)

type MessageAggregate struct {
    info whatsapp.MessageInfo
    text string
	session *whatsapp.Session
	data []byte
	err error
	system bool
}
type waHandler struct {
	wac *whatsapp.Conn
	messages chan MessageAggregate
	downloadsDirectory string
	doDownloads bool
}
var waHandlers = make(map[C.uintptr_t]*waHandler)

func (handler *waHandler) sendMessage(message interface{}, info whatsapp.MessageInfo) *C.char {
	msgId, err := handler.wac.Send(message)
	if err != nil {
		handler.messages <- makeConversationErrorMessage(info,
			fmt.Sprintf("Unable to send message: %v", err))
			return nil
	}
	return C.CString(msgId)
}

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
	if (strings.HasPrefix(gotext, "/sendmedia")) {
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
	if (b) {
		return C.char(1)
	} else {
		return C.char(0)
	}
}

//export gowhatsapp_go_getMessage
func gowhatsapp_go_getMessage(connID C.uintptr_t) C.struct_gowhatsapp_message {
	handler, ok := waHandlers[connID]
	if (!ok) {
		return C.struct_gowhatsapp_message{
			msgtype : C.int64_t(C.gowhatsapp_message_type_error),
			text    : C.CString("Tried to get a message from a non-existant connection. Please file a bug.")}
	}
	select {
	case message := <- handler.messages:
		//fmt.Printf("%+v\n", message)
		if (message.err != nil) {
			// thanks to https://stackoverflow.com/questions/39023475/
			return C.struct_gowhatsapp_message{
				msgtype : C.int64_t(C.gowhatsapp_message_type_error),
				text    : C.CString(message.err.Error()),
			}
		}
		if (message.session != nil) {
			return C.struct_gowhatsapp_message{
			    msgtype     : C.int64_t(C.gowhatsapp_message_type_session),
				clientId    : C.CString(message.session.ClientId),
				clientToken : C.CString(message.session.ClientToken),
				serverToken : C.CString(message.session.ServerToken),
				encKey_b64  : C.CString(base64.StdEncoding.EncodeToString(message.session.EncKey)),
				macKey_b64  : C.CString(base64.StdEncoding.EncodeToString(message.session.MacKey)),
				wid         : C.CString(message.session.Wid),
			}
		}
		info := message.info
		if (!message.system) {
		    handler.wac.Read(info.RemoteJid, info.Id) // mark message as "displayed"
		} else {
		    info.Id = ""
		}
		return C.struct_gowhatsapp_message{
		    msgtype   : C.int64_t(C.gowhatsapp_message_type_text),
			timestamp : C.time_t(info.Timestamp),
			id        : C.CString(info.Id),
			remoteJid : C.CString(info.RemoteJid),
			senderJid : C.CString(info.SenderJid),
			fromMe    : bool_to_Cchar(info.FromMe),
			text      : C.CString(message.text),
			system    : bool_to_Cchar(message.system),
		}
	default:
		return C.struct_gowhatsapp_message{}
	}

}

func (handler *waHandler) HandleError(err error) {
	if (strings.Contains(err.Error(), whatsapp.ErrInvalidWsData.Error())) { // TODO: less ugly error comparison
	    //fmt.Fprintf(os.Stderr, "gowhatsapp: %v ignored.\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "gowhatsapp: error occoured: %v", err)
		handler.messages <- MessageAggregate{err : err}
	}
}

func makeConversationErrorMessage(originalInfo whatsapp.MessageInfo, errorMessage string) MessageAggregate {
    return MessageAggregate{system: true, info : originalInfo, text : errorMessage}
}

func (handler *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
    handler.messages <- MessageAggregate{text : message.Text, info : message.Info}
}

func (handler *waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
    handler.messages <- MessageAggregate{text : message.Caption, info : message.Info}
    handler.downloadMessage(&message, message.Info)
}

func (handler *waHandler) HandleVideoMessage(message whatsapp.VideoMessage) {
    handler.messages <- MessageAggregate{text : message.Caption, info : message.Info}
    handler.downloadMessage(&message, message.Info)
}

func (handler *waHandler) HandleAudioMessage(message whatsapp.AudioMessage) {
    handler.downloadMessage(&message, message.Info)
}

func (handler *waHandler) HandleDocumentMessage(message whatsapp.DocumentMessage) {
    handler.downloadMessage(&message, message.Info)
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
		handler.messages <- MessageAggregate{err : err}
		fmt.Fprintf(os.Stderr, "gowhatsapp error creating connection: %v\n", err)
	} else {
	    wac.AddHandler(handler)
		err = login(handler, session)
		if err != nil {
			wac = nil
			handler.messages <- MessageAggregate{err : err}
			fmt.Fprintf(os.Stderr, "gowhatsapp error logging in: %v\n", err)
		}
	}
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(
	connID C.uintptr_t,
	clientId *C.char,
	clientToken *C.char,
	serverToken *C.char,
	encKey_b64 *C.char,
	macKey_b64 *C.char,
	wid *C.char,
	downloadsDirectory *C.char,
	doDownloads bool,
	) {
	// TODO: protect against concurrent invocation
	handler := waHandler {
		wac : nil,
		messages : make(chan MessageAggregate, 100),
		downloadsDirectory : C.GoString(downloadsDirectory),
		doDownloads : doDownloads,
	}
	waHandlers[connID] = &handler
	var session *whatsapp.Session;
	if (clientId != nil && clientToken != nil && serverToken != nil && encKey_b64 != nil && macKey_b64 != nil && wid != nil) {
		encKey, encKeyErr := base64.StdEncoding.DecodeString(C.GoString(encKey_b64))
		macKey, macKeyErr := base64.StdEncoding.DecodeString(C.GoString(macKey_b64))
		if (encKeyErr == nil && macKeyErr == nil) {
			session = &whatsapp.Session{
				ClientId : C.GoString(clientId),
				ClientToken : C.GoString(clientToken),
				ServerToken : C.GoString(serverToken),
				EncKey : encKey,
				MacKey : macKey,
				Wid : C.GoString(wid)}
		}
	}
	go connect_and_login(&handler, session)
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(connID C.uintptr_t) {
	fmt.Fprintf(os.Stderr, "gowhatsapp close()\n")
	handler, ok := waHandlers[connID]
	if (ok) {
		if (handler.wac != nil) {
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
		handler.messages <- MessageAggregate{session : &session}
	} else {
		//no saved session -> login via qr code
		qr := make(chan string)
		go func() {
			png, err := qrcode.Encode(<-qr, qrcode.Medium, 256) // TODO: make size user configurable
			if err != nil {
			    handler.messages <- MessageAggregate{err : fmt.Errorf("login qr code generation failed: %v\n", err)}
			} else {
			    messageInfo := whatsapp.MessageInfo{
				    RemoteJid: "login@s.whatsapp.net"}
				handler.messages <- MessageAggregate{
				    text : "Scan this QR code within 20 seconds to log in.",
					info : messageInfo,
					system : true}
				filename := generateFilepath(handler.downloadsDirectory, messageInfo)
				handler.storeDownloadedData(messageInfo, filename, png)
			}
		}()
		session, err := wac.Login(qr)
		if err != nil {
		    return fmt.Errorf("error during login: %v\n", err)
		}
		handler.messages <- MessageAggregate{session : &session}
		messageInfo := whatsapp.MessageInfo{
			RemoteJid: "login@s.whatsapp.net"}
		handler.messages <- MessageAggregate{
		    text : "You are now logged in. You may close this window.",
			info : messageInfo,
			system : true}
	}
	return nil
}

func main() {
    gowhatsapp_go_login(0, nil, nil, nil, nil, nil, nil, C.CString("."), true)
	<-time.After(1 * time.Minute)
	gowhatsapp_go_close(0)
}
