Trying to use CMake to build Pidgin/libpurple protocol plug-ins in a way that does not suck.
Because having one Makefile for each toolchain is annoying.

This may or may not become the basis for a re-write of purple-gowhatsapp.

## Windows specific

This will automatically setup a development environment geared towards MS Visual Studio 2022.

Additional dependencies:

* [go (32 bit)](https://go.dev/dl/go1.17.5.windows-386.msi)
* [gcc (32 bit)](https://osdn.net/projects/mingw/)

go and gcc must be in %PATH%.
