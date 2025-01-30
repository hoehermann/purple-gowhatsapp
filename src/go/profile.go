package main

/*
#include "../c/constants.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
)

type ProfilePictureRequest struct {
	who          string // JID of the contact whos profile picture is being requested
	picture_date string // age of profile picture currently being displayed (may be "" in case of no picture)
	picture_id   string // id of profile picture currently being displayed (may be "" in case of no picture)
}

func (handler *Handler) request_profile_picture(who string, picture_date string, picture_id string) {
	handler.pictureRequests <- ProfilePictureRequest{who: who, picture_date: picture_date, picture_id: picture_id}
}

/*
 * Background worker for downloading contact profile pictures
 */
func (handler *Handler) profile_picture_downloader() {
	if handler.httpClient != nil {
		// there already is a httpClient on this connection
		// do not start another downloader
		// TODO: protect against concurrent invocation
		return
	}
	handler.httpClient = &http.Client{}
	log := handler.log.Sub("Profile")
	emptyRequest := ProfilePictureRequest{}
	for pdr := range handler.pictureRequests {
		if pdr == emptyRequest {
			// an emptyRequest may be put into the queue to signal clean exit
			log.Infof("WhatsApp session disconnected. Profile picture downloader is shutting down.")
			return
		}
		if handler.httpClient == nil {
			log.Infof("Profile picture downloader has been removed.")
			return
		}
		if handler.client == nil && !handler.client.IsConnected() {
			// drop requests while not connected to WhatsApp
			continue
		}
		jid, err := parseJID(pdr.who)
		if err != nil {
			purple_error(handler.account, fmt.Sprintf("%#v", err), ERROR_FATAL)
			continue
		}
		// check the settings for whether the user wants small previews or big original pictures
		// NOTE: apart from PREVIEW, there is not only ORIGINAL, but also NO.
		// NO is not accounted for here since in that case, this function should not even be executed.
		setting := purple_get_string(handler.account, C.GOWHATSAPP_ICONS_OPTION, C.GOWHATSAPP_ICONS_PREVIEW)
		want_preview := setting == C.GoString(C.GOWHATSAPP_ICONS_PREVIEW)
		ppi, _ := handler.client.GetProfilePictureInfo(
			jid, &whatsmeow.GetProfilePictureParams{
				Preview:     want_preview,
				ExistingID:  pdr.picture_id,
				IsCommunity: false, // TODO: find out if we do or do not want this
			},
		)
		if ppi == nil {
			// no (updated) picture available for this contact
			continue
		}
		req, err := http.NewRequest("GET", ppi.URL, nil)
		if err != nil {
			log.Warnf("Unable to construct request for profile pictore for %s: %#v", pdr.who, err)
			continue
		}
		if pdr.picture_date != "" {
			// include date of local picture in request
			// NOTE: this should no longer be necessary since we include the ExistingID in GetProfilePictureParams now
			req.Header.Add("If-Modified-Since", pdr.picture_date)
		}
		resp, err := handler.httpClient.Do(req)
		if err != nil {
			log.Warnf("Error downloading profile picture for %s: %#v", pdr.who, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 304 { // not modified
			continue
		}
		var b bytes.Buffer
		_, err = io.Copy(&b, resp.Body)
		if err != nil {
			log.Warnf("Error while transferring profile picture for %s: %#v", pdr.who, err)
			continue
		}
		// store profile picture in contact-specific attachment directory
		directory := purple_get_string(handler.account, C.GOWHATSAPP_ATTACHMENT_DIRECTORY_OPTION, C.GOWHATSAPP_ATTACHMENT_DIRECTORY_DEFAULT)
		if directory != "" {
			os.MkdirAll(filepath.Join(directory, pdr.who), os.ModePerm)
			local_path := filepath.Join(directory, pdr.who, "profile.jpg")
			file, err := os.Create(local_path)
			if err == nil {
				file.Write(b.Bytes())
				file.Close()
			}
		}
		purple_set_profile_picture(handler.account, pdr.who, b.Bytes(), resp.Header.Get("Last-Modified"), ppi.ID)
	}
}
