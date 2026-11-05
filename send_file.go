package main

/*
#include "opusreader.h"
*/
import "C"

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// based on https://github.com/tulir/whatsmeow/blob/main/mdtest/main.go
func (handler *Handler) send_file(who string, filename string) string {
	isGroup := false // TODO: there is a gowhatsapp_chat_send_file, respect the group chat.
	recipient, err := parseJID(who)
	if err != nil {
		return fmt.Sprintf("%#v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Sprintf("Failed to open %s: %v", filename, err)
	}
	_, err = handler.send_file_bytes(recipient, isGroup, data, filepath.Base(filename), true)
	if err != nil {
		return err.Error()
	}
	return ""
}

/*
 * Sends binary data as message.
 * Type of message (image/voice/video/generic document) is decided by detecting magic bytes.
 * The filename shall only be the basename, not the full path.
 */
// TODO: The allowDocument parameter feels whacky. Do that check in the caller.
func (handler *Handler) send_file_bytes(recipient types.JID, isGroup bool, data []byte, filename string, allowDocument bool) (*whatsmeow.SendResponse, error) {
	var err error = nil
	var msg *waE2E.Message = nil
	mimetype := http.DetectContentType(data) // Do not trust the server-supplied mime-type. The server may be misconfigured or mistaken.
	handler.log.Infof("Attachment mime type is %s.", mimetype)
	switch mimetype {
	case "image/jpeg", "image/png":
		msg, err = handler.send_file_image(data, mimetype)
	case "image/webp":
		// WhatsApp uses WebP for stickers
		msg, err = handler.send_file_sticker(data, mimetype)
	case "application/ogg", "audio/ogg":
		cdata := C.CBytes(data)
		opusfile_info := C.opusfile_get_info(cdata, C.size_t(len(data)))
		C.free(cdata)
		seconds := int64(opusfile_info.length_seconds)
		handler.log.Infof("Opus length is %d seconds.", seconds)
		if seconds >= 0 {
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus", uint32(seconds), opusfile_info.waveform)
		} else {
			handler.log.Infof("An ogg audio file was provided, but it was invalid due to %v.", err)
			err = nil // reset error for retrying to send as document
		}
	case "video/mp4":
		err = check_mp4(data)
		if err == nil {
			msg, err = handler.send_file_video(data, "video/mp4")
		} else {
			handler.log.Infof("An mp4 video file was provided, but it was invalid due to %v.", err)
			err = nil // reset error for retrying to send as document
		}
	default:
		// generic other file.
		if !allowDocument {
			return nil, fmt.Errorf("sending file as document message has been suppressed")
		}
	}
	if msg == nil && err == nil {
		// WhatsApp server seems to add extention, so we must remove it.
		filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		msg, err = handler.send_file_document(data, mimetype, filename)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %v", err)
	}
	send_response, err := handler.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return nil, fmt.Errorf("error sending file: %v", err)
	}
	msg.Conversation = &filename // this ends up in the cache, so the message may be looked up by the name of the file
	handler.add_to_cache(msg, send_response.ID, recipient, send_response.Sender, types.MessageSource{Chat: recipient, Sender: send_response.Sender, IsFromMe: true, IsGroup: isGroup}, send_response.Timestamp)
	return &send_response, nil
}

func (handler *Handler) send_file_image(data []byte, mimetype string) (*waE2E.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}

/*
 * Reads the canvas size and the animation flag from the header of a WebP image.
 * See https://developers.google.com/speed/webp/docs/riff_container.
 */
func webp_info(data []byte) (width uint32, height uint32, animated bool, ok bool) {
	if len(data) < 30 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, false, false
	}
	le24 := func(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 }
	switch string(data[12:16]) {
	case "VP8X":
		return le24(data[24:27]) + 1, le24(data[27:30]) + 1, data[20]&0x02 != 0, true
	case "VP8 ":
		if data[23] != 0x9d || data[24] != 0x01 || data[25] != 0x2a {
			return 0, 0, false, false
		}
		return (uint32(data[26]) | uint32(data[27])<<8) & 0x3fff, (uint32(data[28]) | uint32(data[29])<<8) & 0x3fff, false, true
	case "VP8L":
		if data[20] != 0x2f {
			return 0, 0, false, false
		}
		bits := uint32(data[21]) | uint32(data[22])<<8 | uint32(data[23])<<16 | uint32(data[24])<<24
		return bits&0x3fff + 1, (bits>>14)&0x3fff + 1, false, true
	}
	return 0, 0, false, false
}

func (handler *Handler) send_file_sticker(data []byte, mimetype string) (*waE2E.Message, error) {
	width, height, animated, ok := webp_info(data)
	if !ok {
		// the size of WhatsApp's sticker canvas
		width, height = 512, 512
	}
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Width:         proto.Uint32(width),
		Height:        proto.Uint32(height),
		IsAnimated:    proto.Bool(animated),
	}}
	return msg, nil
}

func (handler *Handler) send_file_audio(data []byte, mimetype string, seconds uint32, c_waveform [C.WAVEFORM_SAMPLES_COUNT]C.char) (*waE2E.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return nil, err
	}
	waveform := make([]byte, C.WAVEFORM_SAMPLES_COUNT)
	for i := range c_waveform {
		waveform[i] = byte(c_waveform[i]) // convert while copying
	}
	msg := &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Seconds:       proto.Uint32(seconds),
		PTT:           proto.Bool(true),
		Waveform:      waveform,
	}}
	return msg, nil
}

func (handler *Handler) send_file_video(data []byte, mimetype string) (*waE2E.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}

func (handler *Handler) send_file_document(data []byte, mimetype string, filename string) (*waE2E.Message, error) {
	uploaded, err := handler.client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return nil, err
	}
	msg := &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
		Title:         proto.String(filename),
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(mimetype),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	return msg, nil
}
