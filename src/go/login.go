package main

import (
	"context"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
)

/*
 * This is the go part of purple's login() function.
 */
func login(username string) {
	// TODO: protect against concurrent invocation
	deviceStore, err := container.GetFirstDevice()     // TODO: find out how to use a device jid and use .GetDevice(jid)
	clientLog := waLog.Stdout("Purple", "DEBUG", true) // TODO: have logger write to purple
	if err != nil {
		clientLog.Errorf("%v", err)
	} else {
		handler := Handler{
			client:   whatsmeow.NewClient(deviceStore, clientLog),
			username: username,
			log:      clientLog,
		}
		handlers[username] = &handler
		handler.client.AddEventHandler(handler.eventHandler)
		go handler.connect()
	}
}

/*
 * Helper function for login procedure.
 * Calls whatsmeow.Client.Connect().
 */
func (handler *Handler) connect() {
	clientLog := handler.log
	client := handler.client
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			clientLog.Errorf("%v", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256) // TODO: make size user configurable
				if err != nil {
					clientLog.Errorf("%v", png)
				} else {
					purple_display_qrcode(handler.username, evt.Code, png)
				}
			} else {
				clientLog.Infof("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := client.Connect()
		if err != nil {
			clientLog.Errorf("%v", err)
		}
	}
}

/*
 * This is the go part of purple's close() function.
 */
func close(username string) {
	handler, ok := handlers[username]
	if ok {
		if handler.client != nil {
			handler.client.Disconnect()
		}
		delete(handlers, username)
	}
}
