* `qrcode-size` int  
  The size of the QR code shown for login purposes, in pixels (default: 256). 
  When set to zero, the QR code will be delivered as a text message.
  
* `fetch-contacts-after-linking` bool  
  If set to true (default), buddy list will be populated with the contacts fetched from the main device once right after linking. Does not include groups.
  
* `request-contacts-after-login` bool  
  If set to true (default), buddy list will be populated with the updated contacts and group chats after connecting. 
  
* `update-buddy-on-message` bool  
  If set to true (default), buddy list will be populated with contacts and group chats when receiving a message. 

* `fake-online` bool  
  If set to true (default), contacts currently not online will be regarded as "away" (so they still appear in the buddy list).
  If set to false, offline contacts will be regarded as "offline" (no messages can be sent).

* `send-receipt` string choice  
  Selects when to send receipts "double blue tick" notifications:
  
    * `immediately`: immediately upon message receival
    * `on-interact`: as the user interacts with the conversation window (only usable with Pidgin)
    * `on-answer`: as soon as the user sends an answer (default)
    * `never`: never
  
* `receipt-display` string choice  
  Selects how receipts received for own sent messages are shown:
  
    * `none`: not at all (default). Front-ends and plug-ins can still draw them by listening to the `gowhatsapp-receipt` signal.
    * `text`: a read receipt appears as a system message in the conversation window (can be chatty when many messages are read at once)
  
* `message-cache-size` int  
  Stores a number (default: 0) of messages in local volatile memory. Cached messages are used to provide context when displaying reactions or quoting the message while replying to a specific message. See the [notes](./NOTES.md) for details on how to use the reply feature.  
  Note: Cached messages are persisted to the purple home directory as `username.json`.

* `discard-old-messages` bool  
  If set to true (default: false), messages older than the connection will be discarded.  
  Note: This is implemented without time-zone information. This might not work as expected when chatting with someone in a different time-zone.

* `handle-images` string choice  
  What to do with images:
  
    * `inline`: embed in the conversation window
    * `xfer`: treat as file download
    * `both`: do both (default)

  Note: Attachment download behaviour is influenced by the `attachment-path-template` setting.

* `group-is-file-origin` bool  
  It set to true (default), when a file is posted into a group chat, that chat will be the origin of the file. If set to false, the file will originate from the group chat *participant*. At time of writing, Bitlbee wants this to be false.  
  Note: File transfers for group chats are supported since libpurple 2.14.0.

* `attachment-path-template` string  
  This is a template for specifying a path to a local file-name. Setting this to a non-empty value will enable the automated downloader which stores attachments immediately, completely bypassing libpurple's file transfer mechanism. This can be useful for message bridges with limited resources. Also it can help with maintaining the order of messages. Sub-directories will be created as needed. Profile pictures will be stored in the contact's directory.

  Default value is the empty string.

  The template is passed through `strftime` and accepts time and date format parameters such as `%Y-%m-%d_%H:%M:%S`. The result may not be longer than 128 bytes! Then the replacements are done:

	* `$home`: User directory (same as `~`).
	* `$purple`: Purple configuration directory (usually `~/.purple`).
	* `$direction`: Whether this attachment was "received" (sent by a contact) or "sent" (other device on the own account).
	* `$remote`: The ID of the contact or group chat this attachment has been posted to.
	* `$sender`: The ID of the contact who posted this attachment to the group chat. Empty if not posted in a group chat.
	* `$name`: The human readable name of the contact who sent the attachment. Local alias supplied by the buddy list takes precedence over the name supplied by the server.
	* `$title`: The human readable title of the group chat what was posted to. Local alias supplied by the buddy list takes precedence over the title supplied by the server.
	* `$messageid`: The ID of the message.
	* `$hash`: The file's SHA256 (useful for avoiding clashes and for de-duplication, not set for profile pictures).
	* `$filename`: The sender-supplied file-name (only for Document messages and profile pictures, otherwise empty). Does not contain the extension.
	* `$extension`: A file-name extension fitting the mime-type. Includes the dot.

  Example: `/var/run/purple/$remote/$filename$hash$extension`

  There is no shell expansion (`~` will not become the home directory). Relative paths are resolved to the application's working directory. Using an absolute path is recommended.

* `attachment-symlink` bool   
  Linux only: Uses the template to derive the attachment's local file path again, but with `$remote` and `$sender` being replaced by the respective human readable aliases (as stored in the purple buddy list). Symbolic links pointing to the numeric directories are created. Enabled by default.  
  This feature is not available on Windows since there is no straight-forward C API to create a directory junction.

* `attachment-url-template` string  
  This is a template for an URL to write to the conversation after a file has been stored directly. For the supported place-holders, see `attachment-path-template`.

  Default value is the empty string. A local `file://` URL will be generated on a best-effort basis.

* `get-icons` string choice  
  Every time the plug-in connects, profile pictures are updated:
  
    * `no`: They are not (default).
    * `preview`: The small thumbnail is downloaded from the WhatsApp servers.
    * `original`: The original picture is downloaded from the WhatsApp servers.
    
  Also updates the profile picture when the contact changes it. May incur serious hiccups.

* `ignore-status-broadcast` bool  
  If set to true (default: false), your contact's status broadcasts are ignored.

* `bridge-compatibility` bool  
  Special compatibility setting for protocol bridges like Spectrum or bitlbee. Setting this to true (default: false) will treat system messages just like normal messages, allowing them to be logged and forwarded. This only affects soft errors regarding a specific conversation, e.g. "message could not be sent".
    
* `echo-sent-messages` string choice  
  Selects when to put an outgoing message into the local conversation window:
  
    * `internal`: After the WhatsApp server has received the message, and lock-up the UI until it does (default).
    * `on-success`: After the WhatsApp server has received the message, but do not lock-up the UI **(use this for speed)**.
    * `immediately`: Immediately after hitting send (message may not actually have been sent).
    * `never`: Never (some protocol bridges want this).
    
  Note: Neither of these indicate whether the message has been received by the *contact*.

* `display-message-id` bool  
  If set to true, the ID of a text message will be appended to the displayed text. For outgoing messages, this only has effect if `echo-sent-messages` is set to `on-success`.

* `autojoin-chats` bool  
  Automatically join all chats representing the WhatsApp groups after connecting and every time group information is provided. This is useful for protocol bridges.
  
* `database-address` string  
  whatsmeow stores all session information in a SQL database.
  
  This setting can have place-holders:
  
  * `$purple_user_dir`: Will be replaced by the user directory, e.g. `~/.purple`.
  * `$username`: Will be replaced by the username as entered in the account details.
  
  Default: `file:$purple_user_dir/whatsmeow.db?_pragma=foreign_keys(1)`  
  Folder must exist, `whatsmeow.db` is created automatically.
  
  By default, the driver will be `sqlite` for a file-backed SQLite database. This is not recommended for multi-account-applications (e.g. spectrum or bitlbee). The file-system (see addess option) must support locking and be responsive. Network shares (especially SMB) **do not work**.
  
  If the setting starts with `postgres:`, the suffix will be passed to [database/sql.Open](https://pkg.go.dev/database/sql#Open) as `dataSourceName` for the [pq](https://github.com/lib/pq) PostgreSQL driver. At time of writing, there are no further drivers [supported by whatsmeow](https://github.com/tulir/whatsmeow/blob/4313827/store/sqlstore/container.go#L38). Support for MySQL/MariaDB has been [requested](https://github.com/tulir/whatsmeow/pull/48). 

* `embed-max-file-size` int  
  When set to a value greater than 0 (default, in megabytes), the plug-in tries to detect link-only messages such as `https://example.com/voicemessage.oga` for forwarding.
  
  If enabled, this plug-in tries to download and forward the linked file, choosing the appropriate media type automatically. This way, your contacts do not see a link to an image, video or a voice message, but instead can play the content directly in their app. The message must consist of one URL exactly, including whitespace. For this reason, this mode is incompatible with Pidgin's OTR plug-in, see [this bug report](https://developer.pidgin.im/ticket/10280). 
  
  At time of writing, the maximum file-size supported by WhatsApp is 2 GB according to [wabetainfo](https://wabetainfo.com/whatsapp-is-testing-sharing-media-files-up-to-2gb-in-size/) and confirmed by iOS users in Germany.
  
* `trusted-url-regex` string  
  In case a link-only message does not point to an image, video or audio file, the file may be sent as a document message. For reasons of safety, this will only happen for files from trusted sources. An URL must match this [regular expression](https://golangbyexample.com/golang-regex-match-full-string/) to be considered trustworthy. Matching is case-sensitive. The match spans the entire verbatim URL, so query and fragment need to be considered. Do not forget the caret and/or dollar sign. Some examples:
  
  * `^https://www\.example\.com` trust files from `www.example.com` via HTTPS.
  * `^https://[^/]*example\.com` trust files from `example.com` and subdomains, authentication data via HTTPS.
  * `^[^?#]+\.pdf$` trust all PDFs.
  * `^[^?#]+\.(pdf|png)$` trust all PDFs and PNGs.
  * `^https://www\.example\.com[^?#]+\.(pdf|png)$` trust all PDFs and PNGs from `https://www.example.com`.
  * `.*` trust anything.
  * `^$` trust nothing (default).
  
  In case of image, video or audio files, further conditions need to be met, see [NOTES.md](./NOTES.md). 