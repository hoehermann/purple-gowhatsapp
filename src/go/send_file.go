package main

import (
	"os"
	"path"
	"context"
	"net/http"
	"go.mau.fi/whatsmeow"
	"google.golang.org/protobuf/proto"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// based on https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func (handler *Handler) send_file(who string, filename string) int {
	recipient, ok := handler.parseJID(who) // calls purple_error directly
	if !ok {
		return 14 // EFAULT "Bad address"
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		handler.log.Errorf("Failed to open %s: %v", filename, err)
		return 5 //EIO "Input/output error"
	}
	var msg *waProto.Message
	mimetype := http.DetectContentType(data)
	if mimetype == "image/jpeg" {
		msg, err = handler.send_file_image(data, mimetype)
	} else {
		msg, err = handler.send_file_document(data, mimetype, path.Base(filename))
	}
	if err != nil {
		handler.log.Errorf("Failed to upload file: %v", err)
		return 32 //EPIPE 
	}
	ts, err := handler.client.SendMessage(recipient, "", msg)
	if err != nil {
		handler.log.Errorf("Error sending file: %v", err)
		return 70 //ECOMM
	} else {
		handler.log.Infof("Attachment sent (server timestamp: %s)", ts)
		return 0
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
