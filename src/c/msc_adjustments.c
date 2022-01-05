#ifdef _MSC_VER
#include <stdio.h>
#include <malloc.h>
// resolves LNK2001: unresolved external symbol _fprintf, __imp___iob, __mingw_free
// from https://stackoverflow.com/questions/30412951/unresolved-external-symbol-imp-fprintf-and-imp-iob-func-sdl2#comment72347668_36504365
// and https://stackoverflow.com/questions/49353467/unresolved-symbol-imp-iob-not-imp-iob
#pragma comment(lib, "legacy_stdio_definitions.lib")
FILE iob[3];
FILE * __cdecl _iob(void) { 
    iob[0] = *stdin;
    iob[1] = *stdout;
    iob[2] = *stderr;
    return iob; 
}
void __mingw_free (void * memblock) {
    free(memblock);
}
#endif
