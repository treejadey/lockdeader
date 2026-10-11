// miniaudio's implementation and the wrapper in audio.h, in one translation
// unit so both see the same configuration.

// Including stb_vorbis's header first makes miniaudio decode Ogg Vorbis.
#define STB_VORBIS_HEADER_ONLY
#include "miniaudio/stb_vorbis.c"

// Only playback is needed.
#define MA_NO_ENCODING
#define MA_NO_GENERATION
#define MINIAUDIO_IMPLEMENTATION
#include "miniaudio/miniaudio.h"

#undef STB_VORBIS_HEADER_ONLY
#include "miniaudio/stb_vorbis.c"

#include "audio.h"

#include <string.h>

struct ld_engine {
    ma_engine engine;
};

struct ld_sound {
    ma_decoder decoder;
    ma_sound sound;
    void *data; // what the decoder reads from
};

int ld_engine_new(int noDevice, ld_engine **out) {
    ld_engine *e = ma_malloc(sizeof(ld_engine), NULL);
    if (e == NULL)
        return MA_OUT_OF_MEMORY;

    ma_engine_config config = ma_engine_config_init();
    if (noDevice) {
        config.noDevice = MA_TRUE;
        config.channels = 2;
        config.sampleRate = 48000;
    } else {
        // The device only runs while something plays.
        config.noAutoStart = MA_TRUE;
    }

    ma_result result = ma_engine_init(&config, &e->engine);
    if (result != MA_SUCCESS) {
        ma_free(e, NULL);
        return result;
    }
    *out = e;
    return MA_SUCCESS;
}

void ld_engine_free(ld_engine *engine) {
    ma_engine_uninit(&engine->engine);
    ma_free(engine, NULL);
}

int ld_engine_start(ld_engine *engine) {
    return ma_engine_start(&engine->engine);
}

int ld_engine_stop(ld_engine *engine) {
    return ma_engine_stop(&engine->engine);
}

int ld_engine_read(ld_engine *engine, float *out, unsigned long long frameCount) {
    return ma_engine_read_pcm_frames(&engine->engine, out, frameCount, NULL);
}

int ld_sound_new(ld_engine *engine, const void *data, size_t size, ld_sound **out) {
    ld_sound *s = ma_malloc(sizeof(ld_sound), NULL);
    if (s == NULL)
        return MA_OUT_OF_MEMORY;
    s->data = ma_malloc(size, NULL);
    if (s->data == NULL) {
        ma_free(s, NULL);
        return MA_OUT_OF_MEMORY;
    }
    memcpy(s->data, data, size);

    // The engine converts to its own format, so decode to whatever's native.
    ma_decoder_config config = ma_decoder_config_init(ma_format_f32, 0, 0);
    ma_result result = ma_decoder_init_memory(s->data, size, &config, &s->decoder);
    if (result != MA_SUCCESS)
        goto fail_data;

    result = ma_sound_init_from_data_source(&engine->engine, &s->decoder,
                                            MA_SOUND_FLAG_NO_SPATIALIZATION, NULL, &s->sound);
    if (result != MA_SUCCESS)
        goto fail_decoder;

    *out = s;
    return MA_SUCCESS;

fail_decoder:
    ma_decoder_uninit(&s->decoder);
fail_data:
    ma_free(s->data, NULL);
    ma_free(s, NULL);
    return result;
}

void ld_sound_free(ld_sound *sound) {
    ma_sound_uninit(&sound->sound);
    ma_decoder_uninit(&sound->decoder);
    ma_free(sound->data, NULL);
    ma_free(sound, NULL);
}

int ld_sound_play(ld_sound *sound) {
    return ma_sound_start(&sound->sound);
}

int ld_sound_pause(ld_sound *sound) {
    return ma_sound_stop(&sound->sound);
}

int ld_sound_seek(ld_sound *sound, double seconds) {
    return ma_sound_seek_to_second(&sound->sound, (float)seconds);
}

double ld_sound_position(ld_sound *sound) {
    float seconds = 0;
    ma_sound_get_cursor_in_seconds(&sound->sound, &seconds);
    return seconds;
}

double ld_sound_length(ld_sound *sound) {
    float seconds = 0;
    ma_sound_get_length_in_seconds(&sound->sound, &seconds);
    return seconds;
}

int ld_sound_at_end(ld_sound *sound) {
    return ma_sound_at_end(&sound->sound);
}

void ld_sound_set_volume(ld_sound *sound, float volume) {
    ma_sound_set_volume(&sound->sound, volume);
}

const char *ld_result_string(int result) {
    return ma_result_description((ma_result)result);
}
