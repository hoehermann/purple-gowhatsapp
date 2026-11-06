package main

/*
#include "bridge.h"
*/
import "C"

import (
	"context"
	"encoding/hex"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/util/fallocate"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func extension_from_mimetype(mimeType *string) string {
	extension := ".data"
	if mimeType != nil {
		// use the most poplular default for some common mimetypes
		if *mimeType == "image/jpeg" {
			return ".jpg"
		}
		if *mimeType == "image/png" {
			return ".png"
		}
		if *mimeType == "video/mp4" {
			return ".mp4"
		}
		// anything else is looked up
		extensions, _ := mime.ExtensionsByType(*mimeType)
		if extensions != nil {
			extension = extensions[0]
		}
	}
	return extension
}

/*
 * The file name of a document is chosen by the sender.
 * Make it a single path component so it cannot point outside the attachment directory
 * when it is used in a file-naming template or suggested for a file transfer.
 */
func sanitize_file_name(name string) string {
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, name)
	if name == "." || name == ".." {
		return "_"
	}
	return name
}

// based on https://github.com/FKLC/WhatsAppToDiscord/blob/master/WA2DC.go
func (handler *Handler) handle_attachment(message *waE2E.Message, id string, source types.MessageSource, timestamp time.Time) {
	var (
		caption                                    = ""
		length       uint64                        = 0
		hash                                       = ""
		filename                                   = ""
		extension                                  = ""
		data_type    C.int                         = C.gowhatsapp_attachment_type_none
		downloadable whatsmeow.DownloadableMessage = nil
		mimetype                                   = ""
	)
	{
		im := message.GetImageMessage()
		if im != nil {
			downloadable = im
			hash = hex.EncodeToString(im.GetFileSHA256())
			extension = extension_from_mimetype(im.Mimetype)
			data_type = C.gowhatsapp_attachment_type_image
			mimetype = im.GetMimetype()
			length = im.GetFileLength()
			caption = im.GetCaption()
		}
	}
	{
		vm := message.GetVideoMessage()
		if vm != nil {
			downloadable = vm
			hash = hex.EncodeToString(vm.GetFileSHA256())
			extension = extension_from_mimetype(vm.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
			mimetype = vm.GetMimetype()
			length = vm.GetFileLength()
			caption = vm.GetCaption()
		}
	}
	{
		ptv := message.GetPtvMessage()
		if ptv != nil {
			downloadable = ptv
			hash = hex.EncodeToString(ptv.GetFileSHA256())
			extension = extension_from_mimetype(ptv.Mimetype)
			data_type = C.gowhatsapp_attachment_type_video
			mimetype = ptv.GetMimetype()
			length = ptv.GetFileLength()
			caption = ptv.GetCaption()
		}
	}
	{
		am := message.GetAudioMessage()
		if am != nil {
			downloadable = am
			hash = hex.EncodeToString(am.GetFileSHA256())
			extension = extension_from_mimetype(am.Mimetype)
			data_type = C.gowhatsapp_attachment_type_audio
			mimetype = am.GetMimetype()
			length = am.GetFileLength()
		}
	}
	{
		sm := message.GetStickerMessage()
		if sm != nil {
			downloadable = sm
			hash = hex.EncodeToString(sm.GetFileSHA256())
			extension = extension_from_mimetype(sm.Mimetype)
			data_type = C.gowhatsapp_attachment_type_sticker
			mimetype = sm.GetMimetype()
			length = sm.GetFileLength()
		}
	}
	{
		dm := message.GetDocumentMessage()
		if dm != nil {
			downloadable = dm
			hash = hex.EncodeToString(dm.GetFileSHA256())
			filename = dm.GetFileName()
			extension = filepath.Ext(filename)
			if extension != "" {
				// remove extension from filename for consistency when using file-naming template
				filename = strings.TrimSuffix(filename, extension)
			} else {
				extension = extension_from_mimetype(dm.Mimetype)
			}
			filename = sanitize_file_name(filename)
			data_type = C.gowhatsapp_attachment_type_document
			mimetype = dm.GetMimetype()
			length = dm.GetFileLength()
			caption = dm.GetCaption()
		}
	}
	if data_type != C.gowhatsapp_attachment_type_none {
		caption = escape_text(caption)
		chat := source.Chat.ToNonAD().String()
		sender := source.Sender.ToNonAD().String()
		purple_handle_attachment(handler.account, chat, source.IsGroup, sender, caption, id, timestamp, data_type, filename, extension, mimetype, hash, length, downloadable)
	}
}

// unpreallocated_file hides the *os.File from whatsmeow, so DownloadToFile does not try to
// preallocate it. The media is still written to disk as it arrives.
type unpreallocated_file struct {
	*os.File
}

// fallocate_supported tells whether the file system holding file can preallocate space.
// whatsmeow's DownloadToFile preallocates an *os.File with fallocate(2) and gives up on the
// media host if that fails, which it does on every host where the file system lacks it
// (tmpfs and vfat on older Linux kernels, for example).
func fallocate_supported(file *os.File) bool {
	// the same call whatsmeow makes; on Linux it keeps the file size, so the empty file stays empty
	err := fallocate.Fallocate(file, 1)
	return !errors.Is(err, errors.ErrUnsupported)
}

func (handler *Handler) download_attachment(local_file_path string, message whatsmeow.DownloadableMessage) error {
	err := os.MkdirAll(filepath.Dir(local_file_path), 0o755)
	if err != nil {
		return err
	}
	file, err := os.Create(local_file_path)
	if err != nil {
		return err
	}
	var destination whatsmeow.File = file
	if !fallocate_supported(file) {
		destination = unpreallocated_file{file}
	}
	err = handler.client.DownloadToFile(context.TODO(), message, destination)
	close_err := file.Close()
	if err != nil {
		return err
	}
	return close_err
}
