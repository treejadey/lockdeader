#include "imagecache.h"

#include <QtNetwork/QNetworkAccessManager>
#include <QtNetwork/QNetworkDiskCache>
#include <QtNetwork/QNetworkRequest>
#include <QtQml/QQmlEngine>
#include <QtQml/QQmlNetworkAccessManagerFactory>

namespace {

// Serves files from hosts whose files never change straight from the disk
// cache once they're in it, without checking back with the server.
//
// That has to be AlwaysCache: Qt 6 treats PreferCache like the default and
// still re-checks anything its heuristics consider stale, which, for servers
// that send no caching headers (like GameBanana's), is every recent upload.
class CachingNetworkAccessManager : public QNetworkAccessManager {
public:
    CachingNetworkAccessManager(QStringList immutableHosts, QObject *parent)
        : QNetworkAccessManager(parent), immutableHosts(std::move(immutableHosts)) {}

protected:
    QNetworkReply *createRequest(Operation op, const QNetworkRequest &request, QIODevice *outgoingData) override {
        const bool cached = op == GetOperation
            && immutableHosts.contains(request.url().host())
            && cache() && cache()->metaData(request.url()).isValid();
        if (!cached)
            return QNetworkAccessManager::createRequest(op, request, outgoingData);

        QNetworkRequest fromCache(request);
        fromCache.setAttribute(QNetworkRequest::CacheLoadControlAttribute, QNetworkRequest::AlwaysCache);
        return QNetworkAccessManager::createRequest(op, fromCache, outgoingData);
    }

private:
    const QStringList immutableHosts;
};

// QML calls create() for each thread that loads things (images load on a
// thread of their own), so every manager gets its own disk cache; they all
// share the one directory.
class CachingFactory : public QQmlNetworkAccessManagerFactory {
public:
    CachingFactory(QString dir, qint64 maxBytes, QStringList immutableHosts)
        : dir(std::move(dir)), maxBytes(maxBytes), immutableHosts(std::move(immutableHosts)) {}

    QNetworkAccessManager *create(QObject *parent) override {
        auto *manager = new CachingNetworkAccessManager(immutableHosts, parent);
        auto *cache = new QNetworkDiskCache(manager);
        cache->setCacheDirectory(dir);
        cache->setMaximumCacheSize(maxBytes);
        manager->setCache(cache);
        return manager;
    }

private:
    const QString dir;
    const qint64 maxBytes;
    const QStringList immutableHosts;
};

} // namespace

void lockdeader_install_image_cache(void *engine, const char *dir, long long maxBytes, const char *immutableHosts) {
    QStringList hosts = QString::fromUtf8(immutableHosts).split(u',', Qt::SkipEmptyParts);
    // The engine doesn't take ownership of its factory, and the factory has to
    // outlive the engine, so it's deliberately never deleted.
    static_cast<QQmlEngine *>(engine)->setNetworkAccessManagerFactory(
        new CachingFactory(QString::fromUtf8(dir), maxBytes, std::move(hosts)));
}
