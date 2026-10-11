# Vendored audio libraries

Compiled into the `audio` package by `../audio.c`. Both are unmodified copies of upstream files.

| File | Version | Source | Licence |
|---|---|---|---|
| `miniaudio.h` | 0.11.25 | [mackron/miniaudio](https://github.com/mackron/miniaudio), tag `0.11.25` | Public domain or MIT No Attribution, at your choice (`LICENSE`) |
| `stb_vorbis.c` | 1.22 | `extras/stb_vorbis.c` in the same miniaudio release | Public domain or MIT, at your choice (at the end of the file) |

## Uses

- **miniaudio** decodes MP3, FLAC and WAV, and plays sound through the platform's audio API: WASAPI, Core Audio, or PulseAudio/ALSA/JACK. It loads these at run time, so nothing extra is linked.
- **stb_vorbis** adds Ogg Vorbis. Some GameBanana sound previews are `.ogg`.

## Updating

Replace both files with the ones from a newer miniaudio release (`miniaudio.h` from the repository root, `stb_vorbis.c` from `extras/`). Then update the versions above and run `go test ./audio/`.
