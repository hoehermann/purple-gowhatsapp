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
	client := Client{
		client:   nil,
		username: username,
	}
	clients[username] = &client

	deviceStore, err := container.GetFirstDevice()     // TODO: find out how to use a device jid and use .GetDevice(jid)
	clientLog := waLog.Stdout("Client", "DEBUG", true) // TODO: have logger write to purple
	if err != nil {
		clientLog.Errorf("%v", err)
	} else {
		client.client = whatsmeow.NewClient(deviceStore, clientLog)
		go connect(client, clientLog)
	}
}

/*
 * Helper function for login procedure.
 * Calls whatsmeow.Client.Connect().
 */
func connect(client_ Client, clientLog waLog.Logger) {
	client := client_.client
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
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256) // TODO: make size user configurable
				if err != nil {
					clientLog.Errorf("%v", png)
				} else {
					purple_display_qrcode(client_.username, evt.Code, png)
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
	handler, ok := clients[username]
	if ok {
		if handler.client != nil {
			handler.client.Disconnect()
		}
		delete(clients, username)
	}
}
