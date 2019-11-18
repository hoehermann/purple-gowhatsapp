// gcc -o libdummy.so -shared gowhatsapp_trigger_read_dummy.c

#include <stdio.h>
#include <stdint.h>

void gowhatsapp_trigger_read(uintptr_t userdata) {
    printf("This is the dummy.");
}
