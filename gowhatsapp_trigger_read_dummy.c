// gcc -o libdummy.so -shared gowhatsapp_trigger_read_dummy.c

#include <stdio.h>

void gowhatsapp_trigger_read() {
    printf("This is the dummy.");
}
