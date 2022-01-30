package main

import (
	"bytes"
	"context"
	"fmt"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
	"net/http"
	"os"
	"path"
)

// based on https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func (handler *Handler) send_file(who string, filename string) string {
	isGroup := false // can only send to single contacts for now
	recipient, err := parseJID(who)
	if err != nil {
		return fmt.Sprintf("%#v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Sprintf("Failed to open %s: %v", filename, err)
	}
	var msg *waProto.Message
	mimetype := http.DetectContentType(data)
	handler.log.Infof("Attachment mime type is %s.", mimetype)
	if mimetype == "image/jpeg" {
		msg, err = handler.send_file_image(data, mimetype)
	}
	if mimetype == "application/ogg" {
		reader, header, err := NewWith(bytes.NewReader(data))
		if reader != nil && err == nil && header != nil && header.SampleRate == 16000 && header.Channels == 1 {
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus")
		} else {
			errmsg := fmt.Sprintf("An ogg audio file was provided, but it has not the correct format. Expect opus, channel: 1, rate: 16000, error nil. Got channels: %d , rate: %d, error %s. Sending file as document...", header.Channels, header.SampleRate, err)
			purple_display_system_message(handler.account, who, isGroup, errmsg)
			err = nil
		}
	}
	if msg == nil && err == nil {
		basename := path.Base(filename) // TODO: find out whether this should be with or without extension. WhatsApp server seems to add extention.
		msg, err = handler.send_file_document(data, mimetype, basename)
	}
	if err != nil {
		return fmt.Sprintf("Failed to upload file: %v", err)
	}
	_, err = handler.client.SendMessage(recipient, "", msg)
	if err != nil {
		return fmt.Sprintf("Error sending file: %v", err)
	} else {
		return ""
	}
}

func (handler *Handler) send_file_image(data []byte, mimetype string) (*waProto.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{ImageMessage: &waProto.ImageMessage{
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}

func (handler *Handler) send_file_audio(data []byte, mimetype string) (*waProto.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{AudioMessage: &waProto.AudioMessage{
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}

func (handler *Handler) send_file_document(data []byte, mimetype string, filename string) (*waProto.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
		Title:         proto.String(filename),
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}
