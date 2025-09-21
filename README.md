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
* Note: Although the name of the project suggests otherwise, this is not a perfect drop-in replacement for the plug-in with the obsolete gowhatsapp back-end. For this reason, it has a different ID: `prpl-hehoe-whatsmeow`

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

Look at the [build instructions](./docs/BUILDING.md). This project is being developed on Ubuntu 24.04. Support for other distributions and operating systems is community effort.


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

You may want to have a loot at the documentation of all the [settings](./docs/SETTINGS.md) the plug-in offers.

Some in-depth documentation about obscure details can be found in the [notes](./docs/NOTES.md).

#### Acknowledgements

* [Gary 'grim' Kramlich](https://www.twitch.tv/rw_grim/) for developing Pidgin and purple
* [Eion Robb](https://github.com/EionRobb/) for sharing his invaluable purple advice
* [Peter "theassemblerguy" Bachmaier](https://github.com/theassemblerguy) for initiating the re-write
* [yourealwaysbe](https://github.com/yourealwaysbe) for proper group chats, support and tests against [bitlee](https://github.com/bitlbee/bitlbee)
* [vitalyster](https://github.com/vitalyster) for support, packaging and adjustments for [spectrum2](https://github.com/SpectrumIM/spectrum2)
* Martin Sebald from [hot-chilli.net](https://jabber.hot-chilli.net/) for extensive stress-testing 
* [JimB](https://stackoverflow.com/users/32880/jimb) for golang insights
* [HVV](https://www.hvv.de/) for providing free wifi at their stations

