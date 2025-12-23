package main

/*
#include "constants.h"
*/
import "C"

import (
	"encoding/json"
	"os"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

/*
 * Add a message to the message cache to it can be looked up later.
 * Useful for replying to a specific message and for displaying relevant information when dealing with reactions.
 */
func (handler *Handler) add_to_cache(message *waE2E.Message, info types.MessageInfo) {
	// from https://www.delftstack.com/howto/go/queue-implementation-in-golang/
	handler.cachedMessages = append(handler.cachedMessages, CachedMessage{
		Message: waE2E.Message{
			Conversation:        message.Conversation,
			ExtendedTextMessage: message.ExtendedTextMessage,
			ImageMessage:        message.ImageMessage,
			VideoMessage:        message.VideoMessage,
			PtvMessage:          message.PtvMessage,
			AudioMessage:        message.AudioMessage,
			StickerMessage:      message.StickerMessage,
			DocumentMessage:     message.DocumentMessage,
		},
		Info: info,
	})
	if len(handler.cachedMessages) > purple_get_int(handler.account, C.GOWHATSAPP_MESSAGE_CACHE_SIZE_OPTION, 1000) {
		handler.cachedMessages = handler.cachedMessages[1:]
	}
}

func (handler *Handler) lookup_cached_message_by_id(id string) *CachedMessage {
	// TODO: check whether the look-up does work for outgoing image messages
	// TODO: add/check all kinds of outgoing messages (I do not remember if text-messages are already working)
	for i := range handler.cachedMessages {
		if handler.cachedMessages[i].Info.ID == id {
			return &handler.cachedMessages[i]
		}
	}
	return nil
}

// SaveCachedMessages saves the cached messages to a file.
func (handler *Handler) SaveCachedMessages(filePath string) {
	handler.log.Infof("SaveCachedMessages...")
	file, err := os.Create(filePath)
	if err != nil {
		handler.log.Errorf("failed to create file: %v", err)
		return
	}
	defer file.Close()

	// Encode cached messages to JSON format
	data, err := json.Marshal(handler.cachedMessages)
	if err != nil {
		handler.log.Errorf("failed to marshal cached messages to JSON: %v", err)
		return
	}

	_, err = file.Write(data)
	if err != nil {
		handler.log.Errorf("failed to write cached messages to file: %v", err)
	}
}

// LoadCachedMessages loads the cached messages from a file.
func (handler *Handler) LoadCachedMessages(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		handler.log.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		handler.log.Errorf("failed to read file: %v", err)
	}

	var cachedMessages []CachedMessage
	err = json.Unmarshal(data, &cachedMessages)
	if err != nil {
		handler.log.Errorf("failed to unmarshal cached messages: %v", err)
	}

	handler.cachedMessages = cachedMessages
	handler.log.Infof("Loaded cached messages: %v", cachedMessages)
}
