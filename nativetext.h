#ifndef LOCKDEADER_NATIVETEXT_H
#define LOCKDEADER_NATIVETEXT_H

#ifdef __cplusplus
extern "C" {
#endif

// Makes Qt Quick draw text with the platform's own font rasterizer.
void lockdeader_use_native_text_rendering(void);

#ifdef __cplusplus
}
#endif

#endif
