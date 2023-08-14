package main

/*
#include "../c/opusreader.h"
*/
import "C"

import (
	"context"
	"fmt"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	err = handler.send_file_bytes(recipient, isGroup, data, filename)
	if err != nil {
		return err.Error()
	}
	return ""
}

func (handler *Handler) send_file_bytes(recipient types.JID, isGroup bool, data []byte, filename string) error {
	var err error = nil
	var msg *waProto.Message = nil
	mimetype := http.DetectContentType(data)
	handler.log.Infof("Attachment mime type is %s.", mimetype)
	switch mimetype {
	case "image/jpeg":
		msg, err = handler.send_file_image(data, mimetype)
	case "application/ogg", "audio/ogg":
		opusfile_info := C.opusfile_get_info(C.CBytes(data), C.size_t(len(data)))
		seconds := int64(opusfile_info.length_seconds)
		handler.log.Infof("Opus length is %d seconds.", seconds)
		if seconds >= 0 {
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus", uint32(seconds), opusfile_info.waveform)
		} else {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, fmt.Sprintf("An ogg audio file was provided, but it was invalid. Sending file as document...", err))
			err = nil // reset error for retrying to send as document
		}
	case "video/mp4":
		err = check_mp4(data)
		if err == nil {
			msg, err = handler.send_file_video(data, "video/mp4")
		} else {
			purple_display_system_message(handler.account, recipient.ToNonAD().String(), isGroup, fmt.Sprintf("%s Sending file as document...", err))
			err = nil // reset error for retrying to send as document
		}
	default:
		// generic other file. nothing special to do here.
	}
	if msg == nil && err == nil {
		basename := filepath.Base(filename)
		basename = strings.TrimSuffix(basename, filepath.Ext(basename))
		// WhatsApp server seems to add extention, remove it here.
		msg, err = handler.send_file_document(data, mimetype, basename)
	}
	if err != nil {
		return fmt.Errorf("Failed to upload file: %v", err)
	}
	_, err = handler.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return fmt.Errorf("Error sending file: %v", err)
	} else {
		return nil
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

func (handler *Handler) send_file_audio(data []byte, mimetype string, seconds uint32, c_waveform [C.WAVEFORM_SAMPLES_COUNT]C.char) (*waProto.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}
	waveform := make([]byte, C.WAVEFORM_SAMPLES_COUNT)
	for i := range c_waveform {
		waveform[i] = byte(c_waveform[i]) // convert while copying
	}
	msg := &waProto.Message{AudioMessage: &waProto.AudioMessage{
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Seconds:       proto.Uint32(seconds),
		Ptt:           proto.Bool(true),
		Waveform:      waveform,
	}}
	return msg, nil
}

func (handler *Handler) send_file_video(data []byte, mimetype string) (*waProto.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}
	msg := &waProto.Message{VideoMessage: &waProto.VideoMessage{
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
