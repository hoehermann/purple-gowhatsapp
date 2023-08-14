#include "opusreader.h"

#if __has_include("opusfile.h")

#include <opusfile.h>
#include <glib.h>
#include <math.h>

// TODO: assert ogg_int64_t is int64_t because that is what cgo is written to expect

/**
 * Generates a "waveform" consisting of WAVEFORM_SAMPLES_COUNT values of average amplitude over time.
 * 
 * Assumes a one channel input.
 */
static void opusfile_make_waveform(OggOpusFile * of, ogg_int64_t samples_count, char *waveform) {
  // TODO: read file in small chunks to not exhaust memory
  const ogg_int64_t bufsize = sizeof(float)*samples_count;
  float * pcm = g_malloc0(bufsize); // MEMCHECK: released here
  float * pcm_cursor = pcm;
  for (
    int samples_read_count = op_read_float(of, pcm_cursor, samples_count, NULL); 
    samples_read_count > 0;
    pcm_cursor += samples_read_count, samples_read_count = op_read_float(of, pcm_cursor, samples_count-(pcm-pcm_cursor), NULL)
  ) {
    // TODO: check number of channels here
  }
  
  // sum up sample values into waveform bins
  pcm_cursor = pcm;
  float waveform_float[WAVEFORM_SAMPLES_COUNT] = {};
  for (ogg_int64_t waveform_index = 0; waveform_index < WAVEFORM_SAMPLES_COUNT; waveform_index++) { 
    const ogg_int64_t sample_index_max = samples_count/WAVEFORM_SAMPLES_COUNT; // note: this may drop some of the last samples
    for (ogg_int64_t sample_index = 0; sample_index < sample_index_max; sample_index++) {
      waveform_float[waveform_index] += fabs(*pcm_cursor++);
    }
  }
  free(pcm);
  
  // find maximum for normalization
  float max = 0.0f;
  for (int waveform_index = 0; waveform_index < WAVEFORM_SAMPLES_COUNT; waveform_index++) {
    if (max < waveform_float[waveform_index]) {
      max = waveform_float[waveform_index];
    }
    //printf("Intensity #%d before normalization is %f\n", waveform_index, waveform_float[waveform_index]);
  }
  
  // normalize and convert to percent
  for (int waveform_index = 0; waveform_index < WAVEFORM_SAMPLES_COUNT; waveform_index++) {
    waveform[waveform_index] = (char)(waveform_float[waveform_index]/max*100);
    //printf("Intensity #%d after normalization is %d\n", waveform_index, (int)waveform[waveform_index]);
  }
}

struct opusfile_info opusfile_get_info(void *data, size_t size) {
  struct opusfile_info info = {.length_seconds = -1}; // use negative length to indicate error
  int error;
  OggOpusFile * of = op_open_memory(data, size, &error); // MEMCHECK: released here
  if (of != NULL) {
    ogg_int64_t samples_count = op_pcm_total(of, -1);
    info.length_seconds = samples_count/48000;
    opusfile_make_waveform(of, samples_count, info.waveform);
    op_free(of);
  }
  return info;
}

#else

#pragma message "Warning: Building without opusfile. Sending voice messages is disabled."
struct opusfile_info opusfile_get_info(void *data, size_t size) {
  struct opusfile_info info = {.length_seconds = -1}; // use negative length to indicate error
  return info;
}

#endif
