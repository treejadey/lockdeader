#include "translator.h"

#include <cstdlib>

#include <QtCore/QCoreApplication>
#include <QtCore/QTranslator>

namespace {

// miqt v0.14.0 could override QTranslator::translate, but freed the string Go
// returned before Qt read it, so this asks Go itself. v0.15.0 fixes that.
class GoTranslator : public QTranslator {
public:
    QString translate(const char *context, const char *sourceText, const char *disambiguation, int n) const override {
        Q_UNUSED(context);

        if (!sourceText) {
            return QString();
        }

        char *translated = lockdeaderTranslate(const_cast<char *>(sourceText),
                                               const_cast<char *>(disambiguation ? disambiguation : ""), n);
        QString result = QString::fromUtf8(translated);
        std::free(translated);

        return result;
    }

    bool isEmpty() const override {
        return false;
    }
};

GoTranslator *translator = nullptr;

}

void lockdeader_install_translator(void) {
    if (translator) {
        return;
    }

    translator = new GoTranslator();
    QCoreApplication::installTranslator(translator);
}
