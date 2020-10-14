/*
 *   gowhatsapp plugin for libpurple
 *   Copyright (C) 2019 Hermann Höhne
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"C"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/Rhymen/go-whatsapp"
	"github.com/gabriel-vasile/mimetype"
)

type downloadable interface {
	Download() ([]byte, error)
}

type DownloadableMessage struct {
	Message downloadable
	Type    string
}

func (handler *waHandler) sendMediaMessage(info whatsapp.MessageInfo, filename string) *C.char {
	data, err := os.Open(filename)
	if err != nil {
		handler.presentMessage(makeConversationErrorMessage(info,
			fmt.Sprintf("Unable to read file which was going to be sent: %v", err)))
		return nil
	}
	mime, err := mimetype.DetectReader(data)
	data.Seek(0, 0) // mimetype read some bytes – reset read pointer to start of file
	if mime.Is("image/jpeg") {
		message := whatsapp.ImageMessage{
			Info:    info,
			Type:    mime.String(),
			Content: data,
		}
		// TODO: display own message now, else image will be received (out of order) on reconnect
		return handler.sendMessage(message, info)
	} else if mime.Is("audio/ogg") {
		message := whatsapp.AudioMessage{
			Info:    info,
			Type:    mime.String(),
			Content: data,
		}
		return handler.sendMessage(message, info)
	} else {
		handler.presentMessage(makeConversationErrorMessage(info,
			fmt.Sprintf("Document messages currently not supported (file type is %v).", mime)))
		return nil
	}
}

/*
 * Checks for the message ID looking sane.
 * The message ID is used to infer a file-name.
 * This is to migitate attacks breaking out from the downloads directory.
 */
func isSaneId(s string) bool {
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func generateFilepath(downloadsDirectory string, info whatsapp.MessageInfo, mimeType string) string {
	extension := ".bin"
	extensions, _ := mime.ExtensionsByType(mimeType)
	if extensions != nil {
		extension = extensions[0]
	}
	fp, _ := filepath.Abs(filepath.Join(downloadsDirectory, info.Id) + extension)
	return fp
}

func (handler *waHandler) storeDownloadedData(filename string, data []byte) error {
	os.MkdirAll(handler.downloadsDirectory, os.ModePerm)
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("File %s creation failed due to %v.", filename, err)
	} else {
		_, err := file.Write(data)
		if err != nil {
			return fmt.Errorf("Data could not be written to file %s due to %v.", filename, err)
		} else {
			return nil
		}
	}
}

func (handler *waHandler) presentDownloadableMessage(message DownloadableMessage, info whatsapp.MessageInfo, downloadsEnabled bool, storeFailedDownload bool, inline bool) []byte {
	filename := generateFilepath(handler.downloadsDirectory, info, message.Type)
	_, err := os.Stat(filename)
	notYetDownloaded := os.IsNotExist(err)
	if notYetDownloaded {
		if downloadsEnabled {
			if isSaneId(info.Id) {
				data, err := message.Message.Download()
				if err != nil {
					retryComment := ""
					if storeFailedDownload {
						errStore := handler.storeDownloadedData(filename, make([]byte, 0))
						if errStore != nil {
							retryComment = "Will not try to download again."
						} else {
							// for some odd reason, errStore is always set, even on success, yet empty
							// TODO: find out how to handle this properly. os.Truncate does not create files
							//retryComment = fmt.Sprintf("Unable to mark download as failed (%v). Will try to download again.", errStore)
						}
					} else {
						retryComment = "Retrying on next occasion is enabled."
					}
					handler.presentMessage(makeConversationErrorMessage(info,
						fmt.Sprintf("A media message (ID %s) was received, but the download failed: %v. %s", info.Id, err, retryComment)))
				} else {
					if inline {
						return data
					} else {
						err := handler.storeDownloadedData(filename, data)
						if err != nil {
							handler.presentMessage(makeConversationErrorMessage(info, err.Error()))
						} else {
							handler.presentMessage(MessageAggregate{
								text:   fmt.Sprintf("file://%s", filename),
								info:   info,
								system: true})
						}
					}
				}
			} else {
				handler.presentMessage(makeConversationErrorMessage(info,
					fmt.Sprintf("A media message (ID %s) was received, but ID looks not sane – downloading skipped.", info.Id)))
			}
		} else {
			handler.presentMessage(MessageAggregate{
				text:   "[File download disabled in settings.]",
				system: true,
				info:   info})
		}
	}
	return nil
}
