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