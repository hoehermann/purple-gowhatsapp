package main

import (
	"context"
	"fmt"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
)

/*
 * This is the go part of purple's login() function.
 */
func login(username string) {
	// TODO: protect against concurrent invocation
	deviceStore, err := container.GetFirstDevice() // TODO: find out how to use a device jid and use .GetDevice(jid)
	log := PurpleLogger("Handler")
	if err != nil {
		purple_error(username, fmt.Sprintf("%#v", err))
	} else {
		handler := Handler{
			username: username,
			client:   whatsmeow.NewClient(deviceStore, PurpleLogger("Client")),
			log:      log,
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
			purple_error(handler.username, fmt.Sprintf("%#v", err))
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256) // TODO: make size user configurable
				if err != nil {
					purple_error(handler.username, fmt.Sprintf("%#v", err))
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
			purple_error(handler.username, fmt.Sprintf("%#v", err))
		}
	}
}

/*
 * This is the go part of purple's close() function.
 */
func close(username string) {
	handler, ok := handlers[username]
	if ok {
		handler.client.Disconnect()
		delete(handlers, username)
	}
}
