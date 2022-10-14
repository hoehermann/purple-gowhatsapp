package main

/*
#include "../c/constants.h"
*/
import "C"

/*
 * Add a message to the message cache to it can be looked up later.
 * Useful for displaying relevant information when dealing with reactions.
 */
func (handler *Handler) addToCache(message CachedMessage) {
	// from https://www.delftstack.com/howto/go/queue-implementation-in-golang/
	handler.cachedMessages = append(handler.cachedMessages, message)
	if len(handler.cachedMessages) > purple_get_int(handler.account, C.GOWHATSAPP_MESSAGE_CACHE_SIZE_OPTION, 100) {
		handler.cachedMessages = handler.cachedMessages[1:]
	}
}
