# 1.12.0

* Feature: When receiving a voice-call, a message is displayed (voice calls are not supported).
* Feature: When receiving a poll, a message is displayed (polls are not supported).
* Feature: Support for sending disappearing messages (global setting).
* Change: Errors are only logged (was trigger disconnect).
* Change: Explicit disconnect events are handled (was ignore).
* Change: Textual QR-code now contains <br> tags (was newline only).
* Change: IRC-style commands now start with ? to avoid confusion with actual commands (was start with /).
* Change: The connection is marked as "connected" after fetching contacts for Spectrum (was immediately after connection established).

# 1.11.1

* Bugfix: /versions now actually prints version information.
* Bugfix: Display message sent to a group chat when using `echo` setting `internal`.

# 1.11.0

* Feature: Outgoing voice messages now have a non-zero length.
* Feature: More detailed error description for failed downloads.
* Feature: Can now lock the UI while sending a message until the server answers (for Spectrum).
* Change: Blocking message sending is now the default (was send asynchronously).
* Change: Flag PURPLE_MESSAGE_DELAYED is no longer used.
* Change: Sending an opus file as voice message is now conditional (depending on libopusfile).
* Change: Incoming group messages now take the standard code-path, enabling sounds and other events on Pidin.
* Change: Username must now exactly match the device JID.
* Bugfix: Sending "read" receipts on interacting with the conversation window now actually works on Pidgin.
* Bugfix: No longer mistaking an incoming file name for a contact name.

# 1.10.0

* Change: Incoming files are forwarded to the front-end even if they seem incomplete (was abort transfer).
* Change: When checking linked content, do not rely on server-supplied content-type.
* Change: Accept more kinds of ogg files to send as audio message (was very specific in regard to channels and sample-rate).
* Change: Accept more kinds of mp4 files to send as video message (was disallow content with "edit" tags).
* Change: Presence override is now persistend.
* Change: File transfer anouncement template now also supports printf-style place-holders.

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
