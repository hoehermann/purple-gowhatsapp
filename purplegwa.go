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
    int64_t type;
    time_t timestamp;
    char *id;
    char *remoteJid;
    char *text;
    void *blob;
    uint64_t blobsize;
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

type downloadedImageMessage struct {
    msg  whatsapp.ImageMessage
    data []byte
}
type waHandler struct {
    wac *whatsapp.Conn
    textMessages chan whatsapp.TextMessage
    imageMessages chan downloadedImageMessage
    errorMessages chan error
}
var waHandlers = make(map[C.uintptr_t]*waHandler)

//export gowhatsapp_go_getMessage
func gowhatsapp_go_getMessage(connID C.uintptr_t) C.struct_gowhatsapp_message {
    handler := waHandlers[connID]
	select {
	case message := <- handler.textMessages:
		// thanks to https://stackoverflow.com/questions/39023475/
		return C.struct_gowhatsapp_message{
			C.int64_t(C.gowhatsapp_message_type_text),
			C.time_t(message.Info.Timestamp),
			C.CString(message.Info.Id),
			C.CString(message.Info.RemoteJid),
			C.CString(message.Text),
			nil,
			0}
	case message := <- handler.imageMessages:
		return C.struct_gowhatsapp_message{
			C.int64_t(C.gowhatsapp_message_type_image),
			C.time_t(message.msg.Info.Timestamp),
			C.CString(message.msg.Info.Id),
			C.CString(message.msg.Info.RemoteJid),
			C.CString(message.msg.Caption),
			C.CBytes(message.data),
			C.size_t(len(message.data))}
	case err := <- handler.errorMessages:
		return C.struct_gowhatsapp_message{
			C.int64_t(C.gowhatsapp_message_type_error),
			0,
			nil,
			nil,
			C.CString(err.Error()),
			nil,
			0}
	default:
		return C.struct_gowhatsapp_message{
			0,
			0,
			nil,
			nil,
			nil,
			nil,
			0}
	}
}

func (handler *waHandler) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "gowhatsapp error occoured: %v", err)
	handler.errorMessages <- err
}

func (handler *waHandler) HandleTextMessage(message whatsapp.TextMessage) {
    handler.textMessages <- message
}

func (handler *waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
    data, err := message.Download() // TODO: do not implicitly download on receival (i.e. due to message already recevived and suppressed), ask user before downloading large images
	if err != nil {
		// TODO: propagate error
		fmt.Printf("gowhatsapp message %v image from %v download failed: %v\n", message.Info.Timestamp, message.Info.RemoteJid, err)
		return
	}
	fmt.Printf("gowhatsapp message %v image from %v size is %d.\n", message.Info.Timestamp, message.Info.RemoteJid, len(data))
	handler.imageMessages <- downloadedImageMessage{message, data}
}

func connect_and_login(handler *waHandler) {
    //create new WhatsApp connection
    wac, err := whatsapp.NewConn(20 * time.Second) // TODO: make timeout user configurable
    handler.wac = wac
	if err != nil {
		wac = nil
		handler.errorMessages <- err
		fmt.Fprintf(os.Stderr, "gowhatsapp error creating connection: %v\n", err)
	} else {
	    wac.AddHandler(handler)
		err = login(handler)
		if err != nil {
			wac = nil
			handler.errorMessages <- err
			fmt.Fprintf(os.Stderr, "gowhatsapp error logging in: %v\n", err)
		}
	}
}

//export gowhatsapp_go_login
func gowhatsapp_go_login(connID C.uintptr_t)  {
    // TODO: protect against concurrent invocation
    handler := waHandler {
        wac : nil,
        textMessages : make(chan whatsapp.TextMessage, 100),
        imageMessages : make(chan downloadedImageMessage, 100),
        errorMessages : make(chan error, 100)}
    waHandlers[connID] = &handler
    go connect_and_login(&handler)
}

//export gowhatsapp_go_close
func gowhatsapp_go_close(connID C.uintptr_t) {
	fmt.Fprintf(os.Stderr, "gowhatsapp close()\n")
	handler := waHandlers[connID]
	//handler.wac.wsConn.Close() // inaccessible :(
	handler.wac = nil
	waHandlers[connID] = nil
}

func login(handler *waHandler) error {
    wac := handler.wac
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreSession(session)
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
			    handler.errorMessages <- fmt.Errorf("login qr code generation failed: %v\n", err)
			} else {
				messageInfo := whatsapp.MessageInfo{
					RemoteJid: "login@s.whatsapp.net"}
				message := whatsapp.ImageMessage{
					Info:    messageInfo,
					Caption: "Scan this QR code within 20 seconds to log in."}
				handler.imageMessages <- downloadedImageMessage{message, png}
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
