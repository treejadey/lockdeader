#include "nativetext.h"

#include <QtQuick/QQuickWindow>

void lockdeader_use_native_text_rendering(void) {
    QQuickWindow::setTextRenderType(QQuickWindow::NativeTextRendering);
}
