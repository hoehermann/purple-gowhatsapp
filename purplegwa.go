// package name: purplegwa
package main

/* 
#include <stdint.h>

struct gowhatsapp_message { 
uint64_t timestamp; 
char *id;
char *remoteJid; 
char *text;
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
)

type waHandler struct{}

var messageQueue = make(chan whatsapp.TextMessage, 100)

//export gowhatsapp_go_getMessage
func gowhatsapp_go_getMessage() C.struct_gowhatsapp_message {
    select {
    case message := <- messageQueue:
		return C.struct_gowhatsapp_message{
			C.uint64_t(message.Info.Timestamp), 
			C.CString(message.Info.Id), 
			C.CString(message.Info.RemoteJid), 
			C.CString(message.Text)}
    default:
		return C.struct_gowhatsapp_message{
			C.uint64_t(0), 
			nil, 
			nil, 
			nil}
    }
}

//HandleError needs to be implemented to be a valid WhatsApp handler
func (*waHandler) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "gowhatsapp error occoured: %v", err)
}

func (*waHandler) HandleTextMessage(message whatsapp.TextMessage) {
	messageQueue <- message
	//fmt.Printf("gowhatsapp TextMessage %v %v %v %v\n\t%v\n", message.Info.Timestamp, message.Info.Id, message.Info.RemoteJid, message.Info.QuotedMessageID, message.Text)
}

var wac *whatsapp.Conn

//export gowhatsapp_go_login
func gowhatsapp_go_login() bool {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gowhatsapp error creating connection: %v\n", err)
	} else {
		
		//Add handler
		wac.AddHandler(&waHandler{})
		
		err = login(wac)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gowhatsapp error logging in: %v\n", err)
		} else {
			return true
		}
	}
	wac = nil
	return false
}

//export gowhatsapp_go_close
func gowhatsapp_go_close() {
	wac = nil
	fmt.Fprintf(os.Stderr, "gowhatsapp close() \n")
}

func login(wac *whatsapp.Conn) error {
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreSession(session)
		if err != nil {
			return fmt.Errorf("gowhatsapp restoring failed: %v\n", err)
		}
	} else {
		return fmt.Errorf("gowhatsapp error during login: no session stored\n")
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
