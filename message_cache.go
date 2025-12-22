package main

/*
#include "constants.h"
*/
import "C"

/*
 * Add a message to the message cache to it can be looked up later.
 * Useful for replying to a specific message and for displaying relevant information when dealing with reactions.
 */
func (handler *Handler) add_to_cache(message CachedMessage) {
	// from https://www.delftstack.com/howto/go/queue-implementation-in-golang/
	handler.cachedMessages = append(handler.cachedMessages, message)
	if len(handler.cachedMessages) > purple_get_int(handler.account, C.GOWHATSAPP_MESSAGE_CACHE_SIZE_OPTION, 1000) {
		handler.cachedMessages = handler.cachedMessages[1:]
	}
}

func (handler *Handler) lookup_cached_message_by_id(id string) *CachedMessage {
	// TODO: check whether the look-up does work for outgoing image messages
	// TODO: add/check all kinds of outgoing messages (I do not remember if text-messages are already working)
	for i := range handler.cachedMessages {
		if handler.cachedMessages[i].info.ID == id {
			return &handler.cachedMessages[i]
		}
	}
	return nil
}
