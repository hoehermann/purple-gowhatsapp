# purple-gowhatsapp

A libpurple/Pidgin plugin for WhatsApp **Web**.

Powered by [go-whatsapp](https://github.com/Rhymen/go-whatsapp), which is written by Lucas Engelke.

Being developed on Ubuntu 18.04.

### Building

* Have at least [go-whatsapp e51cd7d](https://github.com/Rhymen/go-whatsapp/commit/e51cd7d0bbd46ddee94a6b2115d4c8d9c2e86a33) or newer.
* Build using the supplied Makefile.
* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

### Set-up

* Create a new account  
  You can enter an arbitrary username. 
  However, it is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, Pigin's logs look sane.
* Upon login, a fake conversation should pop up, showing a QR code.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with the browser-based WhatsApp Web.

### Features

* Receive text messages
* Receive image messages
* Sending text messages

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")  

### Missing Features

* Anything beyond simple messaging, really

### What could be done next

* Support group conversations properly.
* Implement receiving audio, media, and document messages.
* Wait for server message received acknowledgement before displaying sent message locally.
* Sort old messages by date.
* Find spurious segfault.
* Implement sending image, audio, media, and document messages.
