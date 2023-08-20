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
  float waveform_float[WAVEFORM_SAMPLES_COUNT] = {};
  ogg_int64_t samples_per_waveform_bin = samples_count/WAVEFORM_SAMPLES_COUNT;
  ogg_int64_t total_samples_position = 0;
  float pcm[120*48]; // minimum buffer size according to https://opus-codec.org/docs/opusfile_api-0.7/group__stream__decoding.html
  int samples_read_count = op_read_float(of, pcm, samples_count, NULL); 
  // read samples to sum up sample values into waveform bins
  while (samples_read_count > 0) {
    for (int sample_index = 0; sample_index < samples_read_count; sample_index++) {
      ogg_int64_t waveform_index = total_samples_position/samples_per_waveform_bin; // note: this may drop some of the last samples
      waveform_float[waveform_index] += fabs(pcm[sample_index]);
      total_samples_position++;
    }
    // next iteration
    samples_read_count = op_read_float(of, pcm, samples_count, NULL);
  }
  
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
    if (op_channel_count(of,-1) == 1) { // only one-channel recordings are supported
      ogg_int64_t samples_count = op_pcm_total(of, -1);
      info.length_seconds = samples_count/48000;
      opusfile_make_waveform(of, samples_count, info.waveform);
    }
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
