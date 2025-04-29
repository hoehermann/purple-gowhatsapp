#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>

#define WAVEFORM_SAMPLES_COUNT 64 // governed by WhatsApp, see https://github.com/tulir/whatsmeow/discussions/432 for details

struct opusfile_info {
    int64_t length_seconds;
    char waveform[WAVEFORM_SAMPLES_COUNT];
};

struct opusfile_info opusfile_get_info(void *data, size_t size);
