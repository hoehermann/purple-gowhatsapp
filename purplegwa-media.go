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
	"os"
	"path/filepath"
	"strings"

	"github.com/Rhymen/go-whatsapp"
)

func (handler *waHandler) sendMediaMessage(info whatsapp.MessageInfo, text string) *C.char {
	data, err := os.Open(filepath.Join(handler.downloadsDirectory, "outgoing"))
	if err != nil {
		handler.presentMessage(makeConversationErrorMessage(info,
			fmt.Sprintf("Unable to read file which was going to be sent: %v", err)))
		return nil
	}
	// TODO: guess mime type
	if strings.Contains(text, "image") {
		message := whatsapp.ImageMessage{
			Info:    info,
			Type:    "image/jpeg",
			Content: data,
		}
		// TODO: inject system message "[File successfully sent.]"
		// TODO: display own message now, else image will be received out (of order) on reconnect
		return handler.sendMessage(message, info)
	} else if strings.Contains(text, "audio") {
		message := whatsapp.AudioMessage{
			Info:    info,
			Type:    "audio/ogg",
			Content: data,
		}
		return handler.sendMessage(message, info)
	} else {
		handler.presentMessage(makeConversationErrorMessage(info,
			"Please specify file type image or audio"))
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

func generateFilepath(downloadsDirectory string, info whatsapp.MessageInfo) string {
	fp, _ := filepath.Abs(filepath.Join(downloadsDirectory, info.Id))
	return fp
}

func (handler *waHandler) wantToDownload(info whatsapp.MessageInfo) (filename string, want bool) {
	fp := generateFilepath(handler.downloadsDirectory, info)
	_, err := os.Stat(fp)
	return fp, os.IsNotExist(err)
}

func (handler *waHandler) storeDownloadedData(info whatsapp.MessageInfo, filename string, data []byte) {
	os.MkdirAll(handler.downloadsDirectory, os.ModePerm)
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		handler.presentMessage(makeConversationErrorMessage(info,
			fmt.Sprintf("Data was downloaded, but file %s creation failed due to %v", filename, err)))
	} else {
		_, err := file.Write(data)
		if err != nil {
			handler.presentMessage(makeConversationErrorMessage(info,
				fmt.Sprintf("Data was downloaded, but could not be written to file %s due to %v", filename, err)))
		} else {
			handler.presentMessage(MessageAggregate{
				text:   fmt.Sprintf("file://%s", filename),
				info:   info,
				system: true})
		}
	}
}

type downloadable interface {
	Download() ([]byte, error)
}

func (handler *waHandler) downloadMessage(message downloadable, info whatsapp.MessageInfo) {
	filename, wtd := handler.wantToDownload(info)
	if wtd {
		if handler.doDownloads {
			if isSaneId(info.Id) {
				data, err := message.Download()
				if err != nil {
					handler.presentMessage(makeConversationErrorMessage(info,
						fmt.Sprintf("A media message (ID %s) was received, but the download failed: %v", info.Id, err)))
				} else {
					handler.storeDownloadedData(info, filename, data)
				}
			} else {
				handler.presentMessage(makeConversationErrorMessage(info,
					fmt.Sprintf("A media message (ID %s) was received, but ID looks not sane – downloading skipped.", info.Id)))
			}
		} else {
			handler.presentMessage(MessageAggregate{text: "[File download disabled in settings.]", system: true})
		}
	}
}
