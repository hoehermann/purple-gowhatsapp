package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func (handler *Handler) request_profile_picture(who string) {
	handler.pictureRequests <- who
}

func (handler *Handler) profile_picture_downloader() {
	for who := range handler.pictureRequests {
		if handler.client == nil && !handler.client.IsConnected() {
			// drop requests while not connected
			continue
		}
		jid, err := parseJID(who)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err))
			continue
		}
		preview := true
		ppi, _ := handler.client.GetProfilePictureInfo(jid, preview)
		if ppi == nil {
			// no picture set
			continue
		}
		resp, err := http.Get(ppi.URL)
		if err != nil {
			handler.log.Warnf("Error downloading profile picture for %s: %#v", who, err)
			continue
		}
		defer resp.Body.Close()
		var b bytes.Buffer
		_, err = io.Copy(&b, resp.Body)
		if err != nil {
			handler.log.Warnf("Error while transferring profile picture for %s: %#v", who, err)
			continue
		}
		purple_set_profile_picture(handler.account, who, b.Bytes())
		fmt.Printf("whatsmeow profile_picture_downloader fin\n", err)
	}
}
