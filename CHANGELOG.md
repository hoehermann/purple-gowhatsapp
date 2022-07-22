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
* Change: Strip local extension form file name when sending as document (was append WhatsApp suppled extension).
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
