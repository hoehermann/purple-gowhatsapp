#### Linux

Dependencies:

* libpurple
* pkg-config
* cmake (3.20 or newer)
* make
* go (1.25.3 or newer)
* gcc (9.2.0 or newer)
* libgdk-pixbuf-2.0 (optional)
* libopusfile (optional)

For Ubuntu, or Debian compliant Linux flavors, use the apt package manager to install these dependencies first:

    sudo apt install libpurple-dev pkg-config cmake make gcc libgdk-pixbuf2.0-dev libopusfile-dev

In case it is really recent, you can use the go compiler shipped with your distribution (e.g. Arch Linux). All others need to obtain a recent version from https://golang.org/dl/.

For systems with pkg-config, a Makefile exists. For all others, this project uses CMake.

    git clone --recurse-submodules https://github.com/hoehermann/purple-gowhatsapp.git purple-whatsmeow
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

* [go 1.25.3 or newer](https://go.dev/dl/go1.25.3.windows-386.msi)
* [gcc 13.2 or newer](https://packages.msys2.org/package/mingw-w64-i686-gcc)

This is known to work with MSYS make and CMake generator "MSYS Makefiles". go and gcc must be in `%PATH%`.  
At time of writing, cgo does not support MSVC.

For sending opus in ogg audio files as voice messages, add a static win32 build of opusfile to CMake's prefix path or use vcpkg's toolchain file:

    vcpkg.exe install opusfile:x86-mingw-static
    cmake -DCMAKE_TOOLCHAIN_FILE="wherever/vcpkg/scripts/buildsystems/vcpkg.cmake" -DVCPKG_TARGET_TRIPLET=x86-mingw-static -DVCPKG_MANIFEST_MODE=OFF -G "MSYS Makefiles" -S . -B build
    
#### Updates

For bleeding-edge builds, execute `rm purple-whatsmeow/go.{mod,sum}` after cloning. The build system will re-generate them using the latest version of whatsmeow which may or may not be compatible with the glue code.

Using the most recent version of whatsmeow is recommended. Based on experience and user-reports, using an old version of whatmeow may work up to half a year before the WhatsApp servers reject the client. Issues with linking may arise when using a version that is older than three months.
