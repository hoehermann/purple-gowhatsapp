package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/alfg/mp4"
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
		reader, header, err := NewWith(bytes.NewReader(data)) // TODO: have oggreader in its package
		if reader != nil && err == nil && header != nil && header.SampleRate == 16000 && header.Channels == 1 {
			msg, err = handler.send_file_audio(data, "audio/ogg; codecs=opus")
		} else {
			errmsg := fmt.Sprintf("An ogg audio file was provided, but it has not the correct format.\nNeed opus, channel: 1, rate: 16000, error nil.\nGot channels: %d , rate: %d, error %s.\nSending file as document...", header.Channels, header.SampleRate, err)
			purple_display_system_message(handler.account, who, isGroup, errmsg)
			err = nil
		}
	}
	if mimetype == "video/mp4" {
		mp4, err := mp4.OpenFromBytes(data)
		if err != nil {
			errmsg := fmt.Sprintf("An mp4 video file was provided, but it could not be analyzed because of %v\nSending file as document...", err)
			purple_display_system_message(handler.account, who, isGroup, errmsg)
			err = nil
		}
		// NOTE: I assume brand must be mp42
		// TODO: also accept compatible brands with slices.Contains(mp4.Ftyp.CompatibleBrands, "mp42")
		if mp4.Ftyp.MajorBrand == "mp42" && mp4.Ftyp.MinorVersion == 0 && mp4.Moov.IsFragmented == false {
			// NOTE: I assume video track must be at 0
			// presence of Avc1 box (Box.Name "avc1") indicates h264
			// see https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFChap3/qtff3.html
			video_ok := false
			audio_ok := true // TODO: actually check audio
			if len(mp4.Moov.Traks) == 1 || len(mp4.Moov.Traks) == 2 {
				trak := mp4.Moov.Traks[0]
				video_ok = trak.Mdia.Minf.Stbl != nil && trak.Mdia.Minf.Stbl.Stsd.Avc1 != nil
				// TODO: access h264 stream and check for yuv420
			}
			// TODO: assert mp4.Edts == nil ?
			if video_ok && audio_ok {
				msg, err = handler.send_file_video(data, "video/mp4")
			} else {
				errmsg := fmt.Sprintf("An mp4 video file was provided, but it has not the correct format.\nNeed mp42 container: First track avc1 (aka. h264) video. Second track mp4a (aka. aac) audio (optional).\nSending file as document...")
				purple_display_system_message(handler.account, who, isGroup, errmsg)
				err = nil
			}
		}
		/*
			handler.log.Infof("mp4: %+v", mp4)
			handler.log.Infof("mp4.Ftyp: %+v", mp4.Ftyp)
			handler.log.Infof("mp4.Moov: %+v", mp4.Moov)
			handler.log.Infof("mp4.Moov.Mvhd: %+v", mp4.Moov.Mvhd)
			for i, trak := range mp4.Moov.Traks {
				handler.log.Infof("trak[%d]: %+v", i, trak)
				handler.log.Infof("trak[%d].Tkhd: %+v", i, trak.Tkhd)
				handler.log.Infof("trak[%d].Mdia: %+v", i, trak.Mdia)
				handler.log.Infof("trak[%d].Mdia.Hdlr: %+v", i, trak.Mdia.Hdlr) // Handler:"vide", Handler:"soun"
				handler.log.Infof("trak[%d].Mdia.Mdhd: %+v", i, trak.Mdia.Mdhd)
				handler.log.Infof("trak[%d].Mdia.Minf: %+v", i, trak.Mdia.Minf)
				if trak.Mdia.Minf.Vmhd != nil {
					handler.log.Infof("trak[%d].Mdia.Minf.Vmhd: %+v", i, trak.Mdia.Minf.Vmhd)
				}
				if trak.Mdia.Minf.Hmhd != nil {
					handler.log.Infof("trak[%d].Mdia.Minf.Hmhd: %+v", i, trak.Mdia.Minf.Hmhd)
				}
				if trak.Mdia.Minf.Stbl != nil {
					handler.log.Infof("trak[%d].Mdia.Minf.Stbl: %+v", i, trak.Mdia.Minf.Stbl)
					handler.log.Infof("trak[%d].Mdia.Minf.Stbl.Stts: %+v", i, trak.Mdia.Minf.Stbl.Stts)
					handler.log.Infof("trak[%d].Mdia.Minf.Stbl.Stsd: %+v", i, trak.Mdia.Minf.Stbl.Stsd)
					if trak.Mdia.Minf.Stbl.Stsd.Avc1 != nil {
						handler.log.Infof("trak[%d].Mdia.Minf.Stbl.Stsd.Avc1: %+v", i, trak.Mdia.Minf.Stbl.Stsd.Avc1)
						handler.log.Infof("trak[%d].Mdia.Minf.Stbl.Stsd.Avc1.Box: %+v", i, trak.Mdia.Minf.Stbl.Stsd.Avc1.Box)
					}
				}
			}
		*/
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
