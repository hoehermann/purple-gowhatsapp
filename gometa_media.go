//go:build gometa

package main

// Facebook picture/attachment SEND. Plaintext threads: upload to Facebook's Mercury media endpoint
// and send a MEDIA task referencing the attachment fbid. Encrypted threads: upload+encrypt via
// whatsmeow and send an armadillo ImageMessage over the E2EE transport. Mirrors mautrix-meta's
// msgconv (SendMercuryUploadRequest / reuploadMediaToWhatsApp).

/*
#include <stdlib.h>
#include "opusreader.h"
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/mautrix-meta/pkg/messagix/httpclient"
	"go.mau.fi/mautrix-meta/pkg/messagix/methods"
	"go.mau.fi/mautrix-meta/pkg/messagix/socket"
	"go.mau.fi/mautrix-meta/pkg/messagix/table"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waConsumerApplication"
	"go.mau.fi/whatsmeow/proto/waMediaTransport"
	"go.mau.fi/whatsmeow/proto/waMsgApplication"
	waTypes "go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Send a local file to a thread. Returns "" on success or an error string (freed by the C caller).
//
//export gometa_go_send_file
func gometa_go_send_file(account *PurpleAccount, who *C.char, filename *C.char) *C.char {
	h, ok := gometaHandlers[account]
	if !ok {
		return C.CString("Facebook connection not found")
	}
	if err := h.sendFile(C.GoString(who), C.GoString(filename)); err != nil {
		h.logger.Warn().Err(err).Msg("send file failed")
		return C.CString(err.Error())
	}
	return C.CString("")
}

func (h *gometaHandler) sendFile(who, filename string) error {
	threadID, err := strconv.ParseInt(who, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid recipient %q", who)
	}
	if h.client == nil {
		return fmt.Errorf("Facebook not connected")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}
	mime := http.DetectContentType(data)
	isImage := strings.HasPrefix(mime, "image/")
	// The transport transcodes recorded voice notes to Ogg/Opus; http.DetectContentType reports those
	// as "application/ogg". Treat any ogg/audio as a voice note.
	isAudio := mime == "application/ogg" || strings.HasPrefix(mime, "audio/")
	isVideo := strings.HasPrefix(mime, "video/")

	h.mu.Lock()
	isE2EE := h.e2eeContacts[threadID]
	h.mu.Unlock()
	if isE2EE {
		if h.e2ee == nil {
			return fmt.Errorf("encrypted thread and E2EE isn't connected yet")
		}
		if isImage {
			return h.sendImageE2EE(threadID, data, mime)
		}
		if isAudio {
			return h.sendAudioE2EE(threadID, data)
		}
		if isVideo {
			return h.sendVideoE2EE(threadID, data, mime)
		}
		// Anything else (pdf/docx/xlsx/zip/...) goes as an armadillo DocumentMessage.
		return h.sendDocumentE2EE(threadID, data, filepath.Base(filename), mime)
	}
	return h.sendMediaPlaintext(threadID, data, filepath.Base(filename), mime)
}

// sendMediaPlaintext uploads to Facebook and sends a MEDIA task. If Facebook doesn't confirm the
// send (the E2EE-thread rejection), it auto-retries the image over the encrypted transport.
func (h *gometaHandler) sendMediaPlaintext(threadID int64, data []byte, filename, mime string) error {
	ctx, cancel := context.WithTimeout(h.ctx, 120*time.Second)
	defer cancel()
	if err := h.client.WaitUntilCanSendMessages(ctx, 20*time.Second); err != nil {
		return err
	}
	resp, err := h.client.GetHTTP().SendMercuryUploadRequest(ctx, threadID, &httpclient.MercuryUploadMedia{
		Filename:  filename,
		MimeType:  mime,
		MediaData: data,
	})
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	attachmentID := resp.Payload.RealMetadata.GetFbId()
	if attachmentID == 0 {
		return fmt.Errorf("upload returned no attachment id")
	}
	otid := methods.GenerateEpochID()
	h.mu.Lock()
	h.sentOtids[otid] = true
	h.mu.Unlock()
	task := &socket.SendMessageTask{
		ThreadId:         threadID,
		Otid:             otid,
		Source:           table.MESSENGER_INBOX_IN_THREAD,
		InitiatingSource: table.FACEBOOK_INBOX,
		SendType:         table.MEDIA,
		SyncGroup:        1,
		AttachmentFBIds:  []int64{attachmentID},
	}
	sendResp, err := h.client.ExecuteTasks(ctx, task)
	if err != nil {
		return err
	}
	otidStr := strconv.FormatInt(otid, 10)
	if sendResp != nil {
		for _, r := range sendResp.LSReplaceOptimsiticMessage {
			if r.OfflineThreadingId == otidStr {
				h.logger.Info().Int64("thread", threadID).Msg("media sent")
				return nil
			}
		}
	}
	// Not confirmed — likely an encrypted thread. Retry the media over E2EE and remember it.
	h.mu.Lock()
	delete(h.sentOtids, otid)
	h.mu.Unlock()
	isImg := strings.HasPrefix(mime, "image/")
	isAud := mime == "application/ogg" || strings.HasPrefix(mime, "audio/")
	isVid := strings.HasPrefix(mime, "video/")
	if h.e2ee != nil {
		h.mu.Lock()
		h.e2eeContacts[threadID] = true
		h.mu.Unlock()
		h.logger.Warn().Int64("thread", threadID).Msg("media send not confirmed; retrying over E2EE")
		if isAud {
			return h.sendAudioE2EE(threadID, data)
		}
		if isVid {
			return h.sendVideoE2EE(threadID, data, mime)
		}
		if isImg {
			return h.sendImageE2EE(threadID, data, mime)
		}
		return h.sendDocumentE2EE(threadID, data, filename, mime)
	}
	return fmt.Errorf("send not confirmed (thread may be encrypted)")
}

// sendImageE2EE uploads+encrypts an image via whatsmeow and sends it over the E2EE transport.
func (h *gometaHandler) sendImageE2EE(threadID int64, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(h.ctx, 120*time.Second)
	defer cancel()
	uploaded, err := h.e2ee.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("e2ee upload: %w", err)
	}
	w, ht := imageDims(data)
	mediaTransport := &waMediaTransport.WAMediaTransport{
		Integral: &waMediaTransport.WAMediaTransport_Integral{
			FileSHA256:        uploaded.FileSHA256,
			MediaKey:          uploaded.MediaKey,
			FileEncSHA256:     uploaded.FileEncSHA256,
			DirectPath:        &uploaded.DirectPath,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
		},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{
			FileLength: proto.Uint64(uint64(len(data))),
			Mimetype:   &mime,
			// Messenger iOS/Android refuse to render media without a thumbnail w/h.
			Thumbnail: &waMediaTransport.WAMediaTransport_Ancillary_Thumbnail{
				ThumbnailWidth:  proto.Uint32(uint32(w)),
				ThumbnailHeight: proto.Uint32(uint32(ht)),
			},
			ObjectID: &uploaded.ObjectID,
		},
	}
	imageMsg := &waConsumerApplication.ConsumerApplication_ImageMessage{}
	if err := imageMsg.Set(&waMediaTransport.ImageTransport{
		Integral: &waMediaTransport.ImageTransport_Integral{Transport: mediaTransport},
		Ancillary: &waMediaTransport.ImageTransport_Ancillary{
			Height: proto.Uint32(uint32(ht)),
			Width:  proto.Uint32(uint32(w)),
		},
	}); err != nil {
		return fmt.Errorf("build image message: %w", err)
	}
	msg := &waConsumerApplication.ConsumerApplication{
		Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{
				Content: &waConsumerApplication.ConsumerApplication_Content{
					Content: &waConsumerApplication.ConsumerApplication_Content_ImageMessage{ImageMessage: imageMsg},
				},
			},
		},
	}
	otid := methods.GenerateEpochID()
	otidStr := strconv.FormatInt(otid, 10)
	h.mu.Lock()
	h.sentOtids[otid] = true
	// Record this as our own message so a later reaction over E2EE builds a FromMe=true MessageKey
	// (mirrors sendE2EE for text).
	h.e2eeMsgMeta[otidStr] = e2eeMsgInfo{fromMe: true, sender: strconv.FormatInt(h.selfID, 10)}
	h.mu.Unlock()
	to := waTypes.JID{User: strconv.FormatInt(threadID, 10), Server: waTypes.MessengerServer}
	if _, err := h.e2ee.SendFBMessage(ctx, to, msg, &waMsgApplication.MessageApplication_Metadata{},
		whatsmeow.SendRequestExtra{ID: waTypes.MessageID(otidStr)}); err != nil {
		return fmt.Errorf("e2ee image send: %w", err)
	}
	h.logger.Info().Int64("thread", threadID).Msg("e2ee image sent")
	return nil
}

// sendVideoE2EE uploads+encrypts a video via whatsmeow and sends it as an armadillo VideoMessage
// over the E2EE transport. Mirrors sendImageE2EE; dimensions/duration come from the mp4 boxes so
// Messenger clients render (and can scrub) the video instead of showing a broken tile.
func (h *gometaHandler) sendVideoE2EE(threadID int64, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(h.ctx, 120*time.Second)
	defer cancel()
	uploaded, err := h.e2ee.Upload(ctx, data, whatsmeow.MediaVideo)
	if err != nil {
		return fmt.Errorf("e2ee video upload: %w", err)
	}
	w, ht, secs := videoInfoMP4(data)
	if w == 0 || ht == 0 {
		// Messenger refuses media without non-zero dimensions; fall back to a 4:3 default.
		w, ht = 640, 480
	}
	mediaTransport := &waMediaTransport.WAMediaTransport{
		Integral: &waMediaTransport.WAMediaTransport_Integral{
			FileSHA256:        uploaded.FileSHA256,
			MediaKey:          uploaded.MediaKey,
			FileEncSHA256:     uploaded.FileEncSHA256,
			DirectPath:        &uploaded.DirectPath,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
		},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{
			FileLength: proto.Uint64(uint64(len(data))),
			Mimetype:   &mime,
			Thumbnail: &waMediaTransport.WAMediaTransport_Ancillary_Thumbnail{
				ThumbnailWidth:  proto.Uint32(w),
				ThumbnailHeight: proto.Uint32(ht),
			},
			ObjectID: &uploaded.ObjectID,
		},
	}
	videoMsg := &waConsumerApplication.ConsumerApplication_VideoMessage{}
	if err := videoMsg.Set(&waMediaTransport.VideoTransport{
		Integral: &waMediaTransport.VideoTransport_Integral{Transport: mediaTransport},
		Ancillary: &waMediaTransport.VideoTransport_Ancillary{
			Height:      proto.Uint32(ht),
			Width:       proto.Uint32(w),
			Seconds:     proto.Uint32(secs),
			GifPlayback: proto.Bool(false), // a real video, not a looping GIF
		},
	}); err != nil {
		return fmt.Errorf("build video message: %w", err)
	}
	msg := &waConsumerApplication.ConsumerApplication{
		Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{
				Content: &waConsumerApplication.ConsumerApplication_Content{
					Content: &waConsumerApplication.ConsumerApplication_Content_VideoMessage{VideoMessage: videoMsg},
				},
			},
		},
	}
	otid := methods.GenerateEpochID()
	otidStr := strconv.FormatInt(otid, 10)
	h.mu.Lock()
	h.sentOtids[otid] = true
	h.e2eeMsgMeta[otidStr] = e2eeMsgInfo{fromMe: true, sender: strconv.FormatInt(h.selfID, 10)}
	h.mu.Unlock()
	to := waTypes.JID{User: strconv.FormatInt(threadID, 10), Server: waTypes.MessengerServer}
	if _, err := h.e2ee.SendFBMessage(ctx, to, msg, &waMsgApplication.MessageApplication_Metadata{},
		whatsmeow.SendRequestExtra{ID: waTypes.MessageID(otidStr)}); err != nil {
		return fmt.Errorf("e2ee video send: %w", err)
	}
	h.logger.Info().Int64("thread", threadID).Uint32("seconds", secs).Msg("e2ee video sent")
	return nil
}

// sendDocumentE2EE uploads+encrypts a file via whatsmeow and sends it as an armadillo DocumentMessage
// over the E2EE transport. Mirrors sendImageE2EE; the original filename rides on the DocumentMessage so
// Messenger shows it (mirrors mautrix-meta's MsgFile path).
func (h *gometaHandler) sendDocumentE2EE(threadID int64, data []byte, filename, mime string) error {
	ctx, cancel := context.WithTimeout(h.ctx, 120*time.Second)
	defer cancel()
	uploaded, err := h.e2ee.Upload(ctx, data, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("e2ee document upload: %w", err)
	}
	mediaTransport := &waMediaTransport.WAMediaTransport{
		Integral: &waMediaTransport.WAMediaTransport_Integral{
			FileSHA256:        uploaded.FileSHA256,
			MediaKey:          uploaded.MediaKey,
			FileEncSHA256:     uploaded.FileEncSHA256,
			DirectPath:        &uploaded.DirectPath,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
		},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{
			FileLength: proto.Uint64(uint64(len(data))),
			Mimetype:   &mime,
			ObjectID:   &uploaded.ObjectID,
		},
	}
	docMsg := &waConsumerApplication.ConsumerApplication_DocumentMessage{FileName: &filename}
	if err := docMsg.Set(&waMediaTransport.DocumentTransport{
		Integral:  &waMediaTransport.DocumentTransport_Integral{Transport: mediaTransport},
		Ancillary: &waMediaTransport.DocumentTransport_Ancillary{},
	}); err != nil {
		return fmt.Errorf("build document message: %w", err)
	}
	msg := &waConsumerApplication.ConsumerApplication{
		Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{
				Content: &waConsumerApplication.ConsumerApplication_Content{
					Content: &waConsumerApplication.ConsumerApplication_Content_DocumentMessage{DocumentMessage: docMsg},
				},
			},
		},
	}
	otid := methods.GenerateEpochID()
	otidStr := strconv.FormatInt(otid, 10)
	h.mu.Lock()
	h.sentOtids[otid] = true
	h.e2eeMsgMeta[otidStr] = e2eeMsgInfo{fromMe: true, sender: strconv.FormatInt(h.selfID, 10)}
	h.mu.Unlock()
	to := waTypes.JID{User: strconv.FormatInt(threadID, 10), Server: waTypes.MessengerServer}
	if _, err := h.e2ee.SendFBMessage(ctx, to, msg, &waMsgApplication.MessageApplication_Metadata{},
		whatsmeow.SendRequestExtra{ID: waTypes.MessageID(otidStr)}); err != nil {
		return fmt.Errorf("e2ee document send: %w", err)
	}
	h.logger.Info().Int64("thread", threadID).Str("filename", filename).Msg("e2ee document sent")
	return nil
}

// sendAudioE2EE uploads+encrypts an Ogg/Opus voice note via whatsmeow and sends it as a PTT audio
// message over the E2EE transport (armadillo AudioTransport, OPUS format). Mirrors sendImageE2EE.
func (h *gometaHandler) sendAudioE2EE(threadID int64, data []byte) error {
	ctx, cancel := context.WithTimeout(h.ctx, 120*time.Second)
	defer cancel()

	// Duration + waveform from the Opus stream (same reader WhatsApp uses in send_file.go).
	cdata := C.CBytes(data)
	info := C.opusfile_get_info(cdata, C.size_t(len(data)))
	C.free(cdata)
	seconds := int64(info.length_seconds)
	if seconds < 0 {
		return fmt.Errorf("invalid ogg/opus voice note")
	}
	waveform := make([]byte, C.WAVEFORM_SAMPLES_COUNT)
	for i := range info.waveform {
		waveform[i] = byte(info.waveform[i])
	}

	uploaded, err := h.e2ee.Upload(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("e2ee audio upload: %w", err)
	}
	mime := "audio/ogg; codecs=opus"
	opusFmt := waMediaTransport.AudioTransport_Integral_OPUS
	mediaTransport := &waMediaTransport.WAMediaTransport{
		Integral: &waMediaTransport.WAMediaTransport_Integral{
			FileSHA256:        uploaded.FileSHA256,
			MediaKey:          uploaded.MediaKey,
			FileEncSHA256:     uploaded.FileEncSHA256,
			DirectPath:        &uploaded.DirectPath,
			MediaKeyTimestamp: proto.Int64(time.Now().Unix()),
		},
		Ancillary: &waMediaTransport.WAMediaTransport_Ancillary{
			FileLength: proto.Uint64(uint64(len(data))),
			Mimetype:   &mime,
			ObjectID:   &uploaded.ObjectID,
		},
	}
	audioMsg := &waConsumerApplication.ConsumerApplication_AudioMessage{PTT: proto.Bool(true)}
	if err := audioMsg.Set(&waMediaTransport.AudioTransport{
		Integral: &waMediaTransport.AudioTransport_Integral{
			Transport:   mediaTransport,
			AudioFormat: &opusFmt,
		},
		Ancillary: &waMediaTransport.AudioTransport_Ancillary{
			Seconds:  proto.Uint32(uint32(seconds)),
			Waveform: waveform,
		},
	}); err != nil {
		return fmt.Errorf("build audio message: %w", err)
	}
	msg := &waConsumerApplication.ConsumerApplication{
		Payload: &waConsumerApplication.ConsumerApplication_Payload{
			Payload: &waConsumerApplication.ConsumerApplication_Payload_Content{
				Content: &waConsumerApplication.ConsumerApplication_Content{
					Content: &waConsumerApplication.ConsumerApplication_Content_AudioMessage{AudioMessage: audioMsg},
				},
			},
		},
	}
	otid := methods.GenerateEpochID()
	otidStr := strconv.FormatInt(otid, 10)
	h.mu.Lock()
	h.sentOtids[otid] = true
	h.e2eeMsgMeta[otidStr] = e2eeMsgInfo{fromMe: true, sender: strconv.FormatInt(h.selfID, 10)}
	h.mu.Unlock()
	to := waTypes.JID{User: strconv.FormatInt(threadID, 10), Server: waTypes.MessengerServer}
	if _, err := h.e2ee.SendFBMessage(ctx, to, msg, &waMsgApplication.MessageApplication_Metadata{},
		whatsmeow.SendRequestExtra{ID: waTypes.MessageID(otidStr)}); err != nil {
		return fmt.Errorf("e2ee audio send: %w", err)
	}
	h.logger.Info().Int64("thread", threadID).Int64("seconds", seconds).Msg("e2ee voice note sent")
	return nil
}

// imageDims returns the image dimensions, or 400x400 if they can't be decoded (Messenger requires
// non-zero width/height on media).
func imageDims(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width == 0 || cfg.Height == 0 {
		return 400, 400
	}
	return cfg.Width, cfg.Height
}
