#include "opusreader.h"

#if __has_include("opusfile.h")

#include <opusfile.h>
int64_t opus_get_seconds(void *data, size_t size) {
  ogg_int64_t seconds = -1; // use negative length to indicate error
  int error;
  OggOpusFile * of = op_open_memory(data, size, &error);
  if (of != NULL) {
    ogg_int64_t nsamples = op_pcm_total(of, -1);
    seconds = nsamples/48000;
    op_free(of);
  }
  return seconds;
}

#else

#pragma message "Warning: Building without opusfile. Sending voice messages is disabled."
int64_t opus_get_seconds(void *data, size_t size) {
  return -1; // use negative length to indicate error
}

#endif
