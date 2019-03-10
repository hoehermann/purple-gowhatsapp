// package name: purplegwa
package main

/*
#include <stdint.h>
#include <time.h>

enum gowhatsapp_message_type {
    gowhatsapp_message_type_error = -1,
    gowhatsapp_message_type_none = 0,
    gowhatsapp_message_type_text,
    gowhatsapp_message_type_image
};

struct gowhatsapp_message {
    int64_t msgtype;
    char *id;
    char *remoteJid;
    char *senderJid;
    char *text;
    void *blob;
    uint64_t blobsize;
    time_t timestamp;
    char fromMe;
    char system;
};
*/
import "C"

import (
	"encoding/gob"
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/Rhymen/go-whatsapp"
	"github.com/skip2/go-qrcode"
)

type MessageAggregate struct {
	text *whatsapp.TextMessage
	image *whatsapp.ImageMessage
	data []byte
	err error
	system bool
}
type waHandler struct {
	wac *whatsapp.Conn
	messages chan MessageAggregate
}
var waHandlers = make(map[C.uintptr_t]*waHandler)

//export gowhatsapp_go_sendMessage
func gowhatsapp_go_sendMessage(connID C.uintptr_t, who *C.char, text *C.char) C.char {
    message := whatsapp.TextMessage{
        Info: whatsapp.MessageInfo{
            RemoteJid: C.GoString(who),
        },
        Text: C.GoString(text),
    }
    handler := waHandlers[connID]
    if err := handler.wac.Send(message); err != nil {
        handler.messages <- makeConversationErrorMessage(message.Info,
            fmt.Sprintf("Unable to send message: %v", err))
        return 0
    }
    return 1
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
	handler := waHandlers[connID]
	select {
	case message := <- handler.messages:
		//fmt.Printf("%+v\n", message)
		if (message.err != nil) {
			// thanks to https://stackoverflow.com/questions/39023475/
			return C.struct_gowhatsapp_message{
				msgtype : C.int64_t(C.gowhatsapp_message_type_error),
				text : C.CString(message.err.Error())}
		}
		if (message.text != nil) {
			return C.struct_gowhatsapp_message{
				msgtype : C.int64_t(C.gowhatsapp_message_type_text),
				timestamp : C.time_t(message.text.Info.Timestamp),
				id : C.CString(message.text.Info.Id),
				remoteJid : C.CString(message.text.Info.RemoteJid),
				senderJid : C.CString(message.text.Info.SenderJid),
				fromMe : bool_to_Cchar(message.text.Info.FromMe),
				text : C.CString(message.text.Text),
				system :  bool_to_Cchar(message.system)}
		}
		if (message.image != nil) {
			return C.struct_gowhatsapp_message{
				msgtype : C.int64_t(C.gowhatsapp_message_type_image),
				timestamp : C.time_t(message.image.Info.Timestamp),
				id : C.CString(message.image.Info.Id),
				remoteJid : C.CString(message.image.Info.RemoteJid),
				senderJid : C.CString(message.image.Info.SenderJid),
				fromMe : bool_to_Cchar(message.image.Info.FromMe),
				text : C.CString(message.image.Caption),
				blob : C.CBytes(message.data),
				blobsize : C.size_t(len(message.data)),
				system : bool_to_Cchar(message.system)}
		}
		return C.struct_gowhatsapp_message{
			msgtype : C.int64_t(C.gowhatsapp_message_type_error),
			text : C.CString("This should actually never happen. Please file a bug.")}
	default:
		return C.struct_gowhatsapp_message{}
	}

}

func (handler *waHandler) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "gowhatsapp error occoured: %v", err)
	handler.messages <- MessageAggregate{err : err}
}

/*
func (handler *waHandler) HandleJsonMessage(message string) {
	fmt.Printf("%+v\n", message)
}
*/

func (handler *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	handler.messages <- MessageAggregate{text : &message}
}

func (handler *waHandler) HandleVideoMessage(message whatsapp.VideoMessage) {
	handler.messages <- makeConversationErrorMessage(message.Info, 
		"Contact sent a video message, but this plug-in cannot handle video messages.")
}

func (handler *waHandler) HandleAudioMessage(message whatsapp.AudioMessage) {
	handler.messages <- makeConversationErrorMessage(message.Info, 
		"Contact sent an audio message, but this plug-in cannot handle audio messages.")
}

func (handler *waHandler) HandleDocumentMessage(message whatsapp.DocumentMessage) {
	handler.messages <- makeConversationErrorMessage(message.Info, 
		"Contact sent a document message, but this plug-in cannot handle document messages.")
}

func makeConversationErrorMessage(originalInfo whatsapp.MessageInfo, errorMessage string) MessageAggregate {
    m := whatsapp.TextMessage{
        Info : originalInfo,
        Text : errorMessage}
    return MessageAggregate{system: true, text : &m}
}

func (handler *waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
    data, err := message.Download() // TODO: do not implicitly download on receival (i.e. due to message already recevived and suppressed), ask user before downloading large images
	if err != nil {
		fmt.Printf("gowhatsapp message %v image from %v download failed: %v\n", message.Info.Timestamp, message.Info.RemoteJid, err)
		handler.messages <- makeConversationErrorMessage(message.Info,
			fmt.Sprintf("An image message with caption \"%v\" was received, but the image download failed: %v", message.Caption, err))
	} else {
		fmt.Printf("gowhatsapp message %v image from %v size is %d.\n", message.Info.Timestamp, message.Info.RemoteJid, len(data))
		handler.messages <- MessageAggregate{image : &message, data : data}
	}
}

func connect_and_login(handler *waHandler) {
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
		err = login(handler)
		if err != nil {
			wac = nil
			handler.messages <- MessageAggregate{err : err}
			fmt.Fprintf(os.Stderr, "gowhatsapp error logging in: %v\n", err)
		}
	}
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(connID C.uintptr_t)  {
	// TODO: protect against concurrent invocation
	handler := waHandler {
		wac : nil,
		messages : make(chan MessageAggregate, 100)}
	waHandlers[connID] = &handler
	go connect_and_login(&handler)
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(connID C.uintptr_t) {
	fmt.Fprintf(os.Stderr, "gowhatsapp close()\n")
	handler := waHandlers[connID]
	handler.wac.Disconnect()
	handler.wac = nil
	waHandlers[connID] = nil
	delete(waHandlers, connID)
}

func login(handler *waHandler) error {
    wac := handler.wac
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			return fmt.Errorf("gowhatsapp restoring failed: %v\n", err)
			// NOTE: "restore session connection timed out" may indicate phone switched off
		}
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
				message := whatsapp.ImageMessage{
					Info:    messageInfo,
					Caption: "Scan this QR code within 20 seconds to log in."}
				handler.messages <- MessageAggregate{image : &message, data : png, system : true}
			}
		}()
		session, err = wac.Login(qr)
		if err != nil {
		    return fmt.Errorf("error during login: %v\n", err)
		}
	}
	//save session
	err = writeSession(session)
	if err != nil {
	    return fmt.Errorf("error saving session: %v\n", err)
	}
	return nil
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	usr, err := user.Current()
	if err != nil {
		return session, err
	}
	file, err := os.Open(usr.HomeDir + "/.whatsappSession.gob")
	if err != nil {
		return session, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&session)
	if err != nil {
		return session, err
	}
	return session, nil
}

func writeSession(session whatsapp.Session) error {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	file, err := os.Create(usr.HomeDir + "/.whatsappSession.gob")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(session)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	gowhatsapp_go_login(0)
	<-time.After(1 * time.Minute)
	gowhatsapp_go_close(0)
}
