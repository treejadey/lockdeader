#ifndef LOCKDEADER_IMAGECACHE_H
#define LOCKDEADER_IMAGECACHE_H

#ifdef __cplusplus
extern "C" {
#endif

// Gives a QQmlEngine an on-disk cache in dir, of at most maxBytes, for
// everything it loads over the network (in practice: images). Responses from
// the hosts in immutableHosts (comma-separated) are taken from the cache
// without checking back with the server.
void lockdeader_install_image_cache(void *engine, const char *dir, long long maxBytes, const char *immutableHosts);

#ifdef __cplusplus
}
#endif

#endif
