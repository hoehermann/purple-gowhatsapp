# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp powered by [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow is written by Tulir Asokan.

![Instant Message](/docs/instant_message.png?raw=true "Instant Message Screenshot")

### Features

Standard features:

* Connecting to existing account via QR code or 8-character code.
* Receiving messages, sending messages.
* Receiving files (image, video and note, audio and voice, document, sticker).
* Received images are displayed in the conversation window (optional).
* Sending JPEG images as image messages.
* Sending opus audio files as voice messages.
* Sending mp4 video files as video messages.
* Sending other files as documents.
* Fetching all contacts from account, showing friendly names in buddy list, downloading profile pictures ([Markus "nihilus" Gothe](https://github.com/nihilus) for [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy)).
* Sending receipts (configurable).
* Displaying reactions.
* Support for socks5 proxies.
* Reasonable support for group chats by [yourealwaysbe](https://github.com/yourealwaysbe).
* Under the hood: Reasonable callback mechanism thanks to [Eion Robb](https://github.com/EionRobb).

Major differences from the go-whatsapp vesion:

* Incoming messages are not filtered (formerly by Daniele Rogora, now whatsmeow keeps track of already received messages internally).
* Note: Under the hood, gowhatsapp and whatsmeow use completely different prototocls. For this reason, one must establish a new session (scan QR-code) when switching. All old (non-multi-device) sessions will be invalidated. This is a technical requirement.
* Note: This is not a perfect drop-in replacement for the plug-in with the gowhatsapp back-end. For this reason, it has a different ID: `prpl-hehoe-whatsmeow`

Other improvements:

* Contact presence is regarded (buddies are online and offline).
* Typing notifications are handled.
* Logging happens via purple.
* Messages which only consist of a single URL may be sent as media messages (disabled by default).
* Account can be logged out via purple action.
* Reactions are displayed as messages.
* There is an "away" state.
  * For compatibility with the auto-responder plug-in.
  * Other devices (i.e. the main phone) display notifications while plug-in connection is "away".
  * WhatsApp does not send contact presence updates while being "away".
  * Caveat emptor: Other side-effects may occur while using "away" state.

Known issues:

* Contacts:
  * If someone adds you to their contacts and sends you the very first message, the message will not be received. WhatsApp Web shows a notice "message has been delayed – check your phone". This notice is not shown by the plug-in.
* Group Chats:
  * Purple prior to 2.14.0 cannot send files to groups.
  * No notification when being added to a group (the chat will be entered upon receiving a message).
  * The list of participants is not updated if a participants leaves the chat (on Pidgin, closing the window and re-entering the chat triggers a refresh).
* Stickers:
  * A [webp pixbuf loader](https://github.com/aruiz/webp-pixbuf-loader) must be present at runtime.
  * GDK pixbuf headers must be available at build time else presence of loader cannot be checked.
  * Stickers may or may not appear animated depending on loader.
* Special messages:
  * Voice calls are not supported (a warning is displayed).
  * Polls are not supported (a warning is displayed).
  * Other special messages are ignored silently.
* No support for mark-up in outgoing messages.  
  Note: Due to the internal use of [purple_markup_strip_html](https://docs.imfreedom.org/pidgin2/util_8h.html#a0f02bb7e180bb04fb74c8f39564902ee), you need to use a br-tag instead of newline. Pidgin does that automatically, but other clients might not.
* Emojis:
  WhatsApp supports many emojis in text message bodies and reactions. The smiley themes shipped with Pidgin do not cover all emojis. You can install a smiley theme or a font which does, for example the [Google Noto Color Emoji](https://fonts.google.com/noto/specimen/Noto+Color+Emoji) font. On Ubuntu, this is provided by the `fonts-noto-color-emoji` package. There is currently no experience if that works on Windows as well. Feedback is welcome.

Other planned features:

* Support [WhatsApp formatting](https://faq.whatsapp.com/539178204879377/).
* Display receipts in conversation window.
* Join group chat via link.
* View group icon.
* Gracefully handle group updates.
* Action to refresh groups.
* Support [sending mentions](https://github.com/tulir/whatsmeow/discussions/259).
* Support replying to a specific message.

These features will not be worked on:

* Accessing microphone and camera for recording voice or video messages.  
  To prepare a voice message, you can use other tools for recording. I like to use [ffmpeg](https://ffmpeg.org/download.html):
  
      ffmpeg -f pulse -i default -ac 1 -ar 16000 -c:a libopus -y voicemessage.ogg # on Linux with PulseAudio

### Building

#### Linux

This project is being developed on Ubuntu 24.04. Support for other distributions is community effort.

Dependencies:

* libpurple
* pkg-config
* cmake (3.20 or newer)
* make
* go (1.24.2 or newer)
* gcc (9.2.0 or newer)
* libgdk-pixbuf-2.0 (optional)
* libopusfile (optional)

For Ubuntu, or Debian compliant Linux flavors, use the apt package manager to install these dependencies first:

    sudo apt install libpurple-dev pkg-config cmake make gcc libgdk-pixbuf2.0-dev libopusfile-dev

In case it is really recent, you can use the go compiler shipped with your distribution (e.g. Arch Linux). All others need to obtain a recent version from https://golang.org/dl/.

For systems with pkg-config, a Makefile exists. For all others, this project uses CMake.

    git clone --recurse-submodules https://github.com/hoehermann/purple-gowhatsapp.git purple-whatsmeow
    rm purple-whatsmeow/go.{mod,sum} # recommended for bleeding-edge builds
    cmake -S purple-whatsmeow -B build
    cmake --build build
    cmake --install build --strip

You may specify which go compiler binary to use:

    cmake -DCMAKE_Go_COMPILER=/opt/go/bin/go ..

If you configure the project for using user-specific installation paths before building, you may install without sudo:

    cmake -DPURPLE_DATA_DIR:PATH=~/.local/share -DPURPLE_PLUGIN_DIR:PATH=~/.purple/plugins ..

In the build directory, you can also create a Debian package:

    cpack

You should not do that with user-specific paths, obviously.

#### Windows Specific

CMake will try to set-up a development environment automatically. 

Additional dependencies (must be 32 bit aka. win32 aka. x86 aka. 386 aka. i686):

* [go 1.24.2 or newer](https://go.dev/dl/go1.24.2.windows-386.msi)
* [gcc 13.2 or newer](https://packages.msys2.org/package/mingw-w64-i686-gcc)

This is known to work with MSYS make and CMake generator "MSYS Makefiles". go and gcc must be in `%PATH%`.  
At time of writing, cgo does not support MSVC.

For sending opus in ogg audio files as voice messages, add a static win32 build of opusfile to CMake's prefix path or use vcpkg's toolchain file:

    vcpkg.exe install opusfile:x86-mingw-static
    cmake -DCMAKE_TOOLCHAIN_FILE="wherever/vcpkg/scripts/buildsystems/vcpkg.cmake" -DVCPKG_TARGET_TRIPLET=x86-mingw-static -DVCPKG_MANIFEST_MODE=OFF -G "MSYS Makefiles" -S . -B build

### Installation

* Place the binary in your Pidgin's plugin directory (on Linux, that is `~/.purple/plugins`).

#### Set-Up

* Create a new account  
  You must enter your phone's internationalized number followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.

* Upon login, a QR code and the 8-character code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds or enter the 8-character code on your main device – just like you would do with WhatsApp Web.  
  *Note:* On headless clients such as Spectrum, the QR code will be wrapped in a message by a fake contact called "Logon QR Code". You may need to temporarily configure your UI to accept messages from unsolicited users for linking purposes.  
  Wait until the connection has been fully set up. Unfortunately, there is no progress indicator while keys are exchanged and old messages are fetched. Usually, a couple of seconds is enough. Some power users with many groups and contacts reported the process can take more than a minute. If the plug-in is not yet ready, outgoing messages may be dropped silently (see issue #142).

#### Purple Settings

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

* `client-appearance` string choice  
  Selects when to send receipts "double gray tick" notifications:
  
    * `default`: Appear online, but inactive.
    * `appear-online`: Appear online and active. Messages will be marked with the "double gray tick" immediately.
    * `appear-offline`: Receive messages, but appear offline. The implications of this are unknown.

* `passive` bool  
  This connection is passive. No idea what that entails.
  
* `message-cache-size` int  
  Stores a number (default: 100) of messages in local volatile memory. Cached messages are used to provide context when displaying reactions.

* `discard-old-messages` bool  
  If set to true (default: false), messages older than the connection will be discarded.  
  Note: This is implemented without time-zone information. This might not work as expected when chatting with someone in a different time-zone.

* `handle-images` string choice  
  What to do with images:
  
    * `inline`: embed in the conversation window
    * `xfer`: treat as file download
    * `both`: do both
  
* `inline-stickers` bool  
  If set to true (default), stickers will automatically be downloaded and may embedded in the conversation window if an appropriate webp GDK pixbuf loader is present.

* `group-is-file-origin` bool  
  It set to true (default), when a file is posted into a group chat, that chat will be the origin of the file. If set to false, the file will originate from the group chat *participant*. At time of writing, Bitlbee wants this to be false.  
  Note: File transfers for group chats are supported since libpurple 2.14.0.

* `attachment-path-template` string  
  This is a template for specifying a path to a local file-name. Setting this to a non-empty value will store attachments immediately, completely bypassing libpurple's file transfer mechanism. This can be useful for message bridges with limited resources. Sub-directories will be created as needed. Profile pictures will be stored in the contact's directory.

  Default value is the empty string.

  The template is passed through `strftime` and accepts time and date format parameters such as `%Y-%m-%d_%H:%M:%S`. The result may not be longer than 128 bytes! Then the replacements are done:

	* `$home`: User directory (same as `~`).
	* `$purple`: Purple configuration directory (usually `~/.purple`).
	* `$direction`: Whether this attachment was "received" (sent by a contact) or "sent" (other device on the own account).
	* `$remote`: The ID of the contact or group chat this attachment has been posted to.
	* `$sender`: The ID of the contact who posted this attachment to the group chat. Empty if not posted in a group chat.
	* `$messageid`: The ID of the message.
	* `$hash`: The file's SHA256 (useful for avoiding clashes and for de-duplication, not set for profile pictures).
	* `$filename`: The sender-supplied file-name (only for Document messages and profile pictures, otherwise empty). Does not contain the extension.
	* `$extension`: A file-name extension fitting the mime-type. Includes the dot.

  Example: `/var/run/purple/$remote/$filename$hash$extension`

  There is no shell expansion (`~` will not become the home directory). Relative paths are resolved to the application's working directory. Using an absolute path is recommended.

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
  
  In case of image, video or audio files, further conditions need to be met, see below. 

### Notes

#### Conditions for Sending Media Messages

WhatsApp is very picky about media messages. This is actually a good thing for ensuring compatibility on all devices and clients (Android, iOS, all browsers for WhatsApp Web…).

##### Image Message

An image may be sent as an image message (JPEG, `image/jpeg`). This is relatively straight-forward.

##### Voice Message

This feature is only available if the plug-in has been built with liboggfile.

A voice message must meet these criteria:

* Mime-Type: `application/ogg`, `audio/ogg`, sent as `audio/ogg; codecs=opus`
* Container: `ogg`
* Codec: `opus`

Additional recommendations:

* Channels: 1 (mono)

This kind of message is also known as "push to talk" (PTT). While it is possible to send other audio formats as non-voice audio messages, this plug-in only considers data for voice messages. Everything else is send as a document message.

##### Video Message

A video message must meet these criteria:

* Mime-Type: `video/mp4`
* Container Major Brand: `mp42` (observed), `isom` (also accepted)
* Moov Atom Location: Beginning (recommended)
* Video Track:
    * Codec: `h264`
    * Pixel Format: `yuv420p` (assumed, not checked)
* Audio Track (optional):
    * Codec: `aac` (not checked)

Not all of these values are checked by the plug-in. Some of these criteria are guessed and may not actually be WhatsApp restrictions.

##### Document Message

A file is sent as-is.

#### Proxy Support

[whatsmeow](https://github.com/tulir/whatsmeow/blob/9f73bc00d158688a14d0147a93b6b25373facbb8/client.go#L206) offers support for SOCKS5 proxies only. Even if no proxy settings are set in purple, the underlying Go runtime might pick up the `https_proxy` environment variable anyway. 

#### Slash Commands

This plug-in supports a couple of "IRC-style" commands. The user can write them in any chat. These features are experimental hacks. They have been included due to user reuqets. Use them with care.
        
* `?versions`  
  Show version information.
  
* `?contacts`  
  Request re-download of all contacts. Only affects the buddy list if `fetch-contacts` is set to true.

* `?participants` alias `?members`  
  Request the current list of participants. Can only be used in group chat conversations.

* `?presenceavailable`, `?presenceunavailable`, `?presence`  
  Overrides the presence which is being sent to WhatsApp servers. The displayed connection state may no longer match the advertised connection state. This can be used to appear unavailable while still being able to receive messages for logging or notification purposes. Using this command may result in unexpected behaviour. Use `?presence` (without a suffix) to give back control to the plug-in's internals.

* `?logout`  
  Performs a log-out. The QR-code will be requested upon connecting again.

#### Acknowledgements

* [Gary 'grim' Kramlich](https://www.twitch.tv/rw_grim/) for developing Pidgin and purple
* [Eion Robb](https://github.com/EionRobb/) for sharing his invaluable purple advice
* [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy) for initiating the re-write
* [yourealwaysbe](https://github.com/yourealwaysbe) for proper group chats, support and tests against [bitlee](https://github.com/bitlbee/bitlbee)
* [vitalyster](https://github.com/vitalyster) for support, packaging and adjustments for [spectrum2](https://github.com/SpectrumIM/spectrum2)
* Martin Sebald from [hot-chilli.net](https://jabber.hot-chilli.net/) for extensive stress-testing 
* [JimB](https://stackoverflow.com/users/32880/jimb) for golang insights
* [HVV](https://www.hvv.de/) for providing free wifi at their stations

