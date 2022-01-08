# purple-whatsmeow

A libpurple/Pidgin plugin for WhatsApp. Being developed on Ubuntu 20.04. 

This is a re-write of [purple-gowhatsapp](https://github.com/hoehermann/purple-gowhatsapp/tree/gowhatsapp), switching back-ends from [go-whatsapp](https://github.com/Rhymen/go-whatsapp) to [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow is written by Tulir Asokan. It has multi-device support.

![Instant Message](/instant_message.png?raw=true "Instant Message Screenshot")

### Features

Standard features:

* Connecting to existing account via QR-code.
* Receiving messages, sending messages.
* Receiving files (images, videos, voice, document, stickers).
* Reasonable support for group chats by [yourealwaysbe](https://github.com/yourealwaysbe).
* Under the hood: Reasonable callback mechanism thanks to [Eiron Robb](https://github.com/EionRobb).
* Sending images (everything else is considered a document).

Major differences from the go-whatsapp vesion:

* Messages are sent asynchronously.
* Incoming messages are not filtered (whatsmeow keeps track of already received messages internally).

Other improvements:

* There is an "away" state.
* Logging happens via purple.

Features which are present in the go-whatsapp version but missing here:

* Consistent error handling (currently logged to console only in some places).
* Show friendly names in buddy list.
* Get list of participants in group chat.
* Download profile pictures.
* Proxy support.
* Send receipts conditionally.

Known issues:

* After successfully scanning the QR code, connection is not marked as "connected". Manual reconnect resolves the issue.
* In case the QR code times out, the request window remains visible until dismissed by user.
* Notification about friendly name is shown every time it is received.

Other planned features:

* Sending proper voice and video messages.
* Join group chat via link.
* Display receipts in conversation window.

### Building

#### Pre-Built Binaries

To be announced

#### Instructions

Dependencies: pidgin pkg-config cmake make go gcc

This project uses CMake.

    mkdir build
    cd build
    cmake ..
    cmake --build .
    sudo cmake --install .

#### Windows specific

CMake will try to set-up a development environment automatically. 

Additional dependencies:

* [go (32 bit)](https://go.dev/dl/go1.17.5.windows-386.msi)
* [gcc (32 bit)](https://osdn.net/projects/mingw/)

go and gcc must be in `%PATH%`.

MSYS make and CMake generator "MSYS Makefiles" are recommended.  
The project can be opened using Microsoft Visual Studio 2022.  
Compiling with MSVC results in an unusable binary. NOT recommended.  

### Installation

* Place the binary in your Pidgin's plugin directory (`~/.purple/plugins` on Linux).

#### Set-up

* Create a new account  
  You can enter an arbitrary username. 
  However, it is recommended to use your own internationalized number, followed by `@s.whatsapp.net`.  
  Example: `123456789` from Germany would use `49123456789@s.whatsapp.net`.  
  This way, Pidgin's logs look sane.
* Upon login, a QR code is shown in a Pidgin request window.  
  Using your phone's camera, scan the code within 20 seconds – just like you would do with the browser-based WhatsApp Web.  

#### Environment Variables:

whatsmeow stores all session information in a file-based sqlite database. These variables are read when the plug-in loads (even before a connection is established):

* `PURPLE_GOWHATSAPP_DATABASE_DIALECT`  
  default: `sqlite`  

* `PURPLE_GOWHATSAPP_DATABASE_ADDRESS`  
  default: `file:purple_user_dir/whatsmeow.db?_foreign_keys=on`  
  Folder must exist, `whatsmeow.db` is created automatically.
  The file-system must support file locking and be responsive, else the database driver will not work.
  
Other [SQLDrivers](https://github.com/golang/go/wiki/SQLDrivers) may be added upon request.

### Notes

#### Attachment handling and memory consumption

Attachments (images, videos, voice messages, stickers, document) are *always* downloaded as *soon as the message is processed*. The user is then asked where they want the file to be written. During this time, the file data is residing in memory multiple times:

* in the input buffer
* in the decryption buffer
* in the go → C message buffer
* in the C → purple output buffer
* in the output buffer

On systems with many concurrent connection, this could exhaust memory.

As of writing, whatsmeow does not offer an interface to read the file in chunks.
