package main

import (
	"context"
	"fmt"
	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func login(username string) {
	// TODO: protect against concurrent invocation
	handler := Client{
		client:   nil,
		username: username,
	}
	clients[username] = &handler

	deviceStore, err := container.GetFirstDevice()     // TODO: find out how to use a device jid and use .GetDevice(jid)
	clientLog := waLog.Stdout("Client", "DEBUG", true) // TODO: have logger write to purple
	if err != nil {
		clientLog.Errorf("%v", err)
	} else {
		client := whatsmeow.NewClient(deviceStore, clientLog)
		if client.Store.ID == nil {
			// No ID stored, new login
			qrChan, _ := client.GetQRChannel(context.Background())
			err = client.Connect()
			if err != nil {
				clientLog.Errorf("%v", err)
			}
			for evt := range qrChan {
				if evt.Event == "code" {
					// Render the QR code here
					// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
					fmt.Println("QR code:", evt.Code)
				} else {
					fmt.Println("Login event:", evt.Event)
				}
			}
		} else {
			// Already logged in, just connect
			err = client.Connect()
			if err != nil {
				clientLog.Errorf("%v", err)
			}
		}
	}
}

func close(username string) {
	handler, ok := clients[username]
	if ok {
		if handler.client != nil {
			handler.client.Disconnect()
		}
		delete(clients, username)
	}
}
