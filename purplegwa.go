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

type waHandler struct{}
type downloadedImageMessage struct {
	msg  whatsapp.ImageMessage
	data []byte
}

var textMessages = make(chan whatsapp.TextMessage, 100)
var imageMessages = make(chan downloadedImageMessage, 100)
var errorMessages = make(chan error, 1)

//export gowhatsapp_go_getMessage
func gowhatsapp_go_getMessage() C.struct_gowhatsapp_message {
	select {
	case message := <-textMessages:
		// thanks to https://stackoverflow.com/questions/39023475/
		return C.struct_gowhatsapp_message{
			C.int64_t(C.gowhatsapp_message_type_text),
			C.time_t(message.Info.Timestamp),
			C.CString(message.Info.Id),
			C.CString(message.Info.RemoteJid),
			C.CString(message.Text),
			nil,
			0}
	case message := <-imageMessages:
		return C.struct_gowhatsapp_message{
			C.int64_t(C.gowhatsapp_message_type_image),
			C.time_t(message.msg.Info.Timestamp),
			C.CString(message.msg.Info.Id),
			C.CString(message.msg.Info.RemoteJid),
			C.CString(message.msg.Caption),
			C.CBytes(message.data),
			C.size_t(len(message.data))}
	case err := <-errorMessages:
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

func (*waHandler) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "gowhatsapp error occoured: %v", err)
	errorMessages <- err
}

func (*waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	textMessages <- message
}

func (*waHandler) HandleImageMessage(message whatsapp.ImageMessage) {
    data, err := message.Download() // TODO: do not implicitly download on receival (i.e. due to message already recevived and suppressed), ask user before downloading large images
	if err != nil {
		// TODO: propagate error
		fmt.Printf("gowhatsapp message %v image from %v download failed: %v\n", message.Info.Timestamp, message.Info.RemoteJid, err)
		return
	}
	fmt.Printf("gowhatsapp message %v image from %v size is %d.\n", message.Info.Timestamp, message.Info.RemoteJid, len(data))
	imageMessages <- downloadedImageMessage{message, data}
}

var wac *whatsapp.Conn

func connect_and_login() {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(20 * time.Second) // TODO: make timeout user configurable
	if err != nil {
		wac = nil
		errorMessages <- err
		fmt.Fprintf(os.Stderr, "gowhatsapp error creating connection: %v\n", err)
	} else {
		wac.AddHandler(&waHandler{})
		err = login(wac)
		if err != nil {
			wac = nil
			errorMessages <- err
			fmt.Fprintf(os.Stderr, "gowhatsapp error logging in: %v\n", err)
		}
	}
}

//export gowhatsapp_go_login
func gowhatsapp_go_login() {
	go connect_and_login()
}

//export gowhatsapp_go_close
func gowhatsapp_go_close() {
	fmt.Fprintf(os.Stderr, "gowhatsapp close()\n")
	wac = nil
}

func login(wac *whatsapp.Conn) error {
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
				errorMessages <- fmt.Errorf("gowhatsapp login qr code generation failed: %v\n", err)
			} else {
				messageInfo := whatsapp.MessageInfo{
					RemoteJid: "login@s.whatsapp.net"}
				message := whatsapp.ImageMessage{
					Info:    messageInfo,
					Caption: "Scan this QR code within 20 seconds to log in."}
				imageMessages <- downloadedImageMessage{message, png}
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
		return fmt.Errorf("gowhatsapp error saving session: %v\n", err)
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
	gowhatsapp_go_login()
	<-time.After(1 * time.Minute)
	gowhatsapp_go_close()
}
