#ifndef LOCKDEADER_TRANSLATOR_H
#define LOCKDEADER_TRANSLATOR_H

#ifdef __cplusplus
extern "C" {
#endif

// Installs, once, a translator that asks Go's lockdeaderTranslate.
void lockdeader_install_translator(void);

// In translations.go. Returns a string for the caller to free.
extern char *lockdeaderTranslate(char *source, char *disambiguation, int n);

#ifdef __cplusplus
}
#endif

#endif
