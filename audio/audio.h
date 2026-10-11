#ifndef LOCKDEADER_AUDIO_H
#define LOCKDEADER_AUDIO_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// A thin layer over miniaudio's high-level engine, so Go never has to deal
// with miniaudio's structs directly. Functions returning int return a
// miniaudio result code: 0 on success, negative on failure.

typedef struct ld_engine ld_engine;
typedef struct ld_sound ld_sound;

// Creates an engine that plays through the default output device, without
// starting the device yet. With noDevice, it has no device at all, and only
// mixes when ld_engine_read is called.
int ld_engine_new(int noDevice, ld_engine **out);
void ld_engine_free(ld_engine *engine);
int ld_engine_start(ld_engine *engine);
int ld_engine_stop(ld_engine *engine);

// Mixes frameCount stereo frames into out, for engines without a device.
int ld_engine_read(ld_engine *engine, float *out, unsigned long long frameCount);

// Decodes a sound (MP3, Ogg Vorbis, FLAC or WAV) from a copy of data.
int ld_sound_new(ld_engine *engine, const void *data, size_t size, ld_sound **out);
void ld_sound_free(ld_sound *sound);
int ld_sound_play(ld_sound *sound);

// Stops playing, keeping the position.
int ld_sound_pause(ld_sound *sound);

int ld_sound_seek(ld_sound *sound, double seconds);
double ld_sound_position(ld_sound *sound);
double ld_sound_length(ld_sound *sound);
int ld_sound_at_end(ld_sound *sound);
void ld_sound_set_volume(ld_sound *sound, float volume);

const char *ld_result_string(int result);

#ifdef __cplusplus
}
#endif

#endif
