# 1.9.0

* Feature: Have some IRC-style commands.
* Feature: Can display the server-supplied history (experimental).
* Feature: Information about incoming attachments is written to the conversation (configurable).
* Update: Depends on whatsmeow v0.0.0-20230101112920-9b93048f5e21 or later.
* Update: Requires Go 1.18 since whatsmeow v0.0.0-20221228122648-8db2c068c345.
* Change: Displaying status broadcasts is now optional (ignored by default).
* Change: Queries for group participants are now retried.
* Change: Only one connection may exist per username or WhatsApp number.
* Bugfix: Timeout while getting group information no longer leads to a crash.
* Bugfix: File downloads are properly marked as completed with purple 2.14.10.
* Bugfix: Send file to group chat now actually works.
* Bugfix: Plug-in information is now residing in static memory.

# 1.8.0

* Feature: Link-only messages can send arbitrary files from trusted sources (configurable).
* Feature: Reactions can be displayed (enabled by default).
* Change: Information about a specific group is now requested when joining the respective purple room.
* Change: Files can originate from group chats (enabled by default).
* Bugfix: Extension is stripped from outgoing filename since WhatsApp adds it.

# 1.7.0

* Feature: Client logout is available via action.
* Feature: Files can be sent to group chats (untested).
* Change: Credentials supplied by WhatsApp servers are forwarded to main thread (might affect bitlbee).
* Bugfix: Memory of embedded images is released after message is displayed.

# 1.6.0

* Feature: Conditional support for WebP stickers (not animated).

# 1.5.0

* Update: Depends on whatsmeow v0.0.0-20220711113451 or newer.

# 1.4.0

* Update: Depends on whatsmeow v0.0.0-20220629162100.
* Change: More socket related errors should trigger an automated reconnect within libpurple.

# 1.3.0

* Feature: Socks5 proxy support.
* Change: Echo behaviour can now be controlled via `echo-sent-messages` setting (was previously included implicitly in `bridge-compatibility`).
* Bugfix: Do not forget to update (or create) database structures.

# 1.2.0

* Change: Automatically detect bitlbee when storing credentials. `bridge-compatibility` no longer regarded for this feature.
* Change: SQLite connections limited to 1 (was unlimited).
* Change: Strip local extension form file name when sending as document (was append WhatsApp supplied extension).
* Bugfix: A non-media file is sent as a document (was fail sending message).
* Bugfix: Do not duplicate chat participants.

# 1.1.0

* Feature: Can send a mp4 video as video message.
* Feature: Link-only messages can be converted in media messages.
* Change: Using "file:…" for sqlite3 and "postgres:…" for pq in `database-address` setting.
* Change: Presence subscriptions are issued even when in "away" state (was "online" only).
* Bugfix: Push name updates should now be mentioned only once.

# 1.0.0

First release since re-write.
