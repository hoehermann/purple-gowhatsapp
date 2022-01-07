This is purple-whatsmeow. It is a re-write of [purple-gowhatsapp](https://github.com/hoehermann/purple-gowhatsapp/tree/gowhatsapp), switching back-ends from [go-whatsapp](https://github.com/Rhymen/go-whatsapp) to [whatsmeow](https://github.com/tulir/whatsmeow). whatsmeow has multi-device support.

## Features

Major differences from the go-whatsapp vesion:

* Messages are sent asynchronously.
* Incoming messages are not filtered (whatsmeow keeps track of already received messages internally).
* Receiving files (images, videos, voice, document, stickers).

Other improvements:

* There is an "away" state.
* Logging happens via purple.

Missing features which are presend in the go-whatsapp version:

* Error handling (currently logged to console only).
* Sending files.
* Downloading profile pictures.
* Proxy support.
* Send receipts conditionally.

Other planned features.

* Join group chat via link.

## Run-Time Configuration

### Environment Variables:

These are read when the plug-in loads (before a connection is established):

* `PURPLE_GOWHATSAPP_DATABASE_DIALECT`  
  default: `sqlite`  

* `PURPLE_GOWHATSAPP_DATABASE_ADDRESS`  
  default: `file:purple_user_dir/whatsmeow.db?_foreign_keys=on`  
  Folder must exist, `whatsmeow.db` is created automatically.
  The file-system must support file locking and be responsive, else the database driver will not work.

## Building

### Windows specific

CMake will try to set-up a development environment automatically. 
The project can be opened using Microsoft Visual Studio 2022.

Additional dependencies:

* [go (32 bit)](https://go.dev/dl/go1.17.5.windows-386.msi)
* [gcc (32 bit)](https://osdn.net/projects/mingw/)

go and gcc must be in %PATH%.

## Notes

### Attachment handling and memory consumption

Attachments (images, videos, voice messages, stickers, document) are *always* downloaded as *soon as the message is processed*. The user is then asked where they want the file to be written. During this time, the file data is residing in memory multiple times:

* in the input buffer
* in the decryption buffer
* in the go → C message buffer
* in the C → purple output buffer
* in the output buffer

On systems with many concurrent connection, this could exhaust memory.

As of writing, whatsmeow does not offer an interface to read the file in chunks.
