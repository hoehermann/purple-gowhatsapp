# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp. Being developed on Ubuntu 22.04. 

This is a re-write of [purple-gowhatsapp](https://github.com/hoehermann/purple-gowhatsapp/tree/gowhatsapp), switching back-ends from [go-whatsapp](https://github.com/Rhymen/go-whatsapp) to [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow is written by Tulir Asokan. It has multi-device support.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")

### Features

Standard features:

* Connecting to existing account via QR-code.
* Receiving messages, sending messages.
* Receiving files (images, videos, voice, document, stickers).
* Received images are displayed in the conversation window (optional).
* Sending images as image messages.
* Sending ogg audio files as voice messages.
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
  * Stickers are not animated.
* Special messages:
  * Voice calls are not supported (a warning is displayed).
  * Votes are not supported (a warning is displayed).
  * Other special messages are irgnored silently.

Other planned features:

* Display receipts in conversation window.
* Join group chat via link.
* View group icon.
* Gracefully handle group updates.
* Action to refresh groups.
* Support [sending mentions](https://github.com/tulir/whatsmeow/discussions/259).

These features will not be worked on:

* Accessing microphone and camera for recording voice or video messages.  
  To prepare a voice message, you can use other tools for recording. I like to use [ffmpeg](https://ffmpeg.org/download.html):
  
      ffmpeg -f pulse -i default -ac 1 -ar 16000 -c:a libopus -y voicemessage.ogg # on Linux with PulseAudio

### Building

#### Pre-Built Binaries

* [Nightly Build](https://buildbot.hehoe.de/purple-whatsmeow/builds/) (Windows, Ubuntu).

#### Instructions

Dependencies: 

* pidgin (libpurple glib gtk)
* pkg-config
* cmake (3.8 or newer)
* make
* go (1.18 or newer)
* gcc (9.2.0 or newer)
* libgdk-pixbuf-2.0 (optional)
* libopusfile (optional)

This project uses CMake.

    mkdir build
    cd build
    cmake ..
    cmake --build .
    sudo make install/strip

#### Windows Specific

CMake will try to set-up a development environment automatically. 

Additional dependencies:

* [go 1.18 or newer (32 bit)](https://go.dev/dl/go1.18.9.windows-386.msi)
* [gcc 9.2.0 or newer (32 bit)](https://osdn.net/projects/mingw/)

go and gcc must be in `%PATH%`.  

This is known to work with MSYS make and CMake generator "MSYS Makefiles".  
The project can be opened using Microsoft Visual Studio 2022.  
Compiling with MSVC results in an unusable binary. NOT recommended.  

For sending opus in ogg audio files as voice messages, add a static win32 build of opusfile to the prefix path:

    vcpkg.exe install opusfile:x86-mingw-static
    cmake -G "MSYS Makefiles" -DCMAKE_PREFIX_PATH=wherever/vcpkg/installed/x86-mingw-static ..

### Installation

* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

#### Set-Up

* Create a new account  
  You must enter your phone's internationalized number followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.

* Upon login, a QR code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with WhatsApp Web.  
  *Note:* On headless clients such as Spectrum, the QR code will be wrapped in a message by a fake contact called "Logon QR Code". You may need to temporarily configure your UI to accept messages from unsolicited users for linking purposes.  
  Wait until the connection has been fully set up. Unfortunately, there is no progress indicator while keys are exchanged and old messages are fetched. Usually, a couple of seconds is enough. Some power users with many groups and contacts reported the process can take more than a minute. If the plug-in is not yet ready, outgoing messages may be dropped silently (see issue #142).

#### Purple Settings

* `qrcode-size`  
  The size of the QR code shown for login purposes, in pixels (default: 256). 
  When set to zero, the QR code will be delivered as a text message.
  
* `fetch-contacts`  
  If set to true (default), buddy list will be populated with contacts sent by server. 
  This is useful for the first login in particular. If enabled while connecting, it will also fetch the current list of WhatsApp groups.
  
* `fake-online`  
  If set to true (default), contacts currently not online will be regarded as "away" (so they still appear in the buddy list).
  If set to false, offline contacts will be regarded as "offline" (no messages can be sent).

* `send-receipt`  
  Selects when to send receipts "double blue tick" notifications:
  
    * `immediately`: immediately upon message receival
    * `on-interact`: as the user interacts with the conversation window (only usable with Pidgin)
    * `on-answer`: as soon as the user sends an answer (default)
    * `never`: never

* `message-cache-size`  
  Stores a number (default: 100) of messages in local volatile memory. Cached messages are used to provide context when displaying reactions.

* `fetch-history`  
  If set to true (default: false), the history of conversations will be displayed. This feature is experimental. WhatsApp servers send the history once during the linking process. Messages will appear out-of-order. Files will be downloaded again. If used when linking for the first time (without prior population of the buddy list), it may add name-less contacts to the buddy list.

* `inline-images`  
  If set to true (default), images will automatically be downloaded and embedded in the conversation window.
  
* `inline-stickers`  
  If set to true (default), stickers will automatically be downloaded and may embedded in the conversation window if an appropriate webp GDK pixbuf loader is present.

* `group-is-file-origin`  
  It set to true (default), when a file is posted into a group chat, that chat will be the origin of the file. If set to false, the file will originate from the group chat *participant*. At time of writing, Bitlbee wants this to be false.  
  Note: File transfers for group chats are supported since libpurple 2.14.0.
  
* `attachment-message`  
  This system message is written to the conversation for each incoming attachment. It can have two `%s` place-holders which wil be fed into a call to `printf`.
  
  1. The first `%s` will be replaced by the original sender (the participant, not the group chat).  
  1. The second `%s` will be replaced by the original document file name. For non-document attachments, this falls back to the hash mandated by WhatsApp.

  Default value is `Preparing to store "%s" sent by %s...`.

  If built and used in a Linux environment with GLib 2.68 or newer, you can also use place-holders for more flexibility:

  * `$sender`: Denotes the original sender (the participant, not the group chat).  
  * `$filename`: Refers to the original document file name. For non-document attachments, this falls back to the hash mandated by WhatsApp.

* `get-icons`  
  If set to true (default: false), profile pictures are updated every time the plug-in connects.

* `ignore-status-broadcast`  
  If set to true (default), your contact's status broadcasts are ignored.

* `bridge-compatibility`  
  Special compatibility setting for protocol bridges like Spectrum or bitlbee. Setting this to true (default: false) will treat system messages just like normal messages, allowing them to be logged and forwarded. This only affects soft errors regarding a specific conversation, e.g. "message could not be sent".
    
* `echo-sent-messages`  
  Selects when to put an outgoing message into the local conversation window:
  
    * `internal`: After the WhatsApp server has received the message, and lock-up the UI until it does (default).
    * `on-success`: After the WhatsApp server has received the message, but do not lock-up the UI **(use this for speed)**.
    * `immediately`: Immediately after hitting send (message may not actually have been sent).
    * `never`: Never (some protocol bridges want this).
    
  Note: Neither of these indicate whether the message has been received by the *contact*.

* `autojoin-chats`  
  Automatically join all chats representing the WhatsApp groups after connecting and every time group information is provided. This is useful for protocol bridges.
  
* `database-address`  
  whatsmeow stores all session information in a SQL database.
  
  This setting can have place-holders:
  
  * `$purple_user_dir`: Will be replaced by the user directory, e.g. `~/.purple`.
  * `$username`: Will be replaced by the username as entered in the account details.
  
  Default: `file:$purple_user_dir/whatsmeow.db?_foreign_keys=on&_busy_timeout=3000`  
  Folder must exist, `whatsmeow.db` is created automatically.
  
  By default, the driver will be `sqlite3` for a file-backed SQLite database. This is not recommended for multi-account-applications (e.g. spectrum or bitlbee) due to a [limitation in the driver](https://github.com/mattn/go-sqlite3/issues/209). The file-system (see addess option) must support locking and be responsive. Network shares (especially SMB) **do not work**.
  
  If the setting starts with `postgres:`, the suffix will be passed to [database/sql.Open](https://pkg.go.dev/database/sql#Open) as `dataSourceName` for the [pq](https://github.com/lib/pq) PostgreSQL driver. At time of writing, there are no further drivers [supported by whatsmeow](https://github.com/tulir/whatsmeow/blob/b078a9e/store/sqlstore/container.go#L34). Support for MySQL/MariaDB has been [requested](https://github.com/tulir/whatsmeow/pull/48). 

* `embed-max-file-size`  
  When set to a value greater than 0 (default, in megabytes), the plug-in tries to detect link-only messages such as `https://example.com/voicemessage.oga` for forwarding.
  
  If enabled, this plug-in tries to download and forward the linked file, choosing the appropriate media type automatically. This way, your contacts do not see a link to an image, video or a voice message, but instead can play the content directly in their app. The message must consist of one URL exactly, including whitespace. For this reason, this mode is incompatible with Pidgin's OTR plug-in, see [this bug report](https://developer.pidgin.im/ticket/10280). 
  
  At time of writing, the maximum file-size supported by WhatsApp is 2 GB according to [wabetainfo](https://wabetainfo.com/whatsapp-is-testing-sharing-media-files-up-to-2gb-in-size/) and confirmed by iOS users in Germany.
  
* `trusted-url-regex`  
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

An image may be sent as an image message (JPEG, `image/jpeg`). This is relatively straight-forward.

A voice message must meet these criteria:

* Mime-Type: `application/ogg`, `audio/ogg`, sent as `audio/ogg; codecs=opus`
* Container: `ogg`
* Codec: `opus`

Additional recommendations:

* Channels: 1 (mono)

This feature is only available if the plug-in has been built with liboggfile.

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
  Overrides the presence which is being sent to WhatsApp servers. The displayed connection state may no longer match the advertised connection state. This can be used to appear unavailable while still being able to receive messages for logging or notification purposes. Using this command may result in unexpected behaviour. Use `/presence` (without a suffix) to give back control to the plug-in's internals.

* `?logout`  
  Performs a log-out. The QR-code will be requested upon connecting again.

#### Attachment Handling and Memory Consumption

Attachments (images, videos, voice messages, stickers, document) are *always* downloaded as *soon as the message is processed*. The user is then asked where they want the file to be written. During this time, the file data is residing in memory multiple times:

* in the input buffer
* in the decryption buffer
* in the go → C message buffer
* in the output buffer

On systems with many concurrent connections, this could exhaust memory.

As of writing, whatsmeow does not offer an interface to read the file in chunks.

#### Acknowledgements

* [Gary 'grim' Kramlich](https://www.twitch.tv/rw_grim/) for developing Pidgin and purple
* [Eion Robb](https://github.com/EionRobb/) for sharing his invaluable purple advice
* [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy) for initiating the re-write
* [yourealwaysbe](https://github.com/yourealwaysbe) for proper group chats, support and tests against [bitlee](https://github.com/bitlbee/bitlbee)
* [vitalyster](https://github.com/vitalyster) for support, packaging and adjustments for [spectrum2](https://github.com/SpectrumIM/spectrum2)
* Martin Sebald from [hot-chilli.net](https://jabber.hot-chilli.net/) for extensive stress-testing 
* [JimB](https://stackoverflow.com/users/32880/jimb) for golang insights
* [HVV](https://www.hvv.de/) for providing free wifi at their stations

