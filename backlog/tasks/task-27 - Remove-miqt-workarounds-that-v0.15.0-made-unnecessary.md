---
id: TASK-27
title: Remove miqt workarounds that v0.15.0 made unnecessary
status: Done
assignee:
  - '@claude'
created_date: '2026-11-05T13:45:00Z'
updated_date: '2026-11-05T13:48:00Z'
labels:
  - chore
dependencies: []
priority: medium
type: chore
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
miqt v0.15.0 (TASK-26) fixed the use-after-free in virtual overrides that return heap types (mappu/miqt#338). The C++ translator shim (translator.cpp/.h) existed only because of it. Checked 2026-11-05: imagecache.cpp and nativetext.cpp fill gaps miqt never bound (QQmlNetworkAccessManagerFactory::create, Qt Quick), and the list-of-pairs setter bug is still in v0.15.0, so those stay. The mod list model stays a QStandardItemModel: a Go data() override on QAbstractListModel returns a QVariant* that C++ copies but never frees, so it would leak per cell read.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Translations go through a QTranslator overridden in Go with miqt; translator.cpp and translator.h are gone
- [x] #2 Translations still work: go test ./... passes and the app shows translated text for a language with a .po file
- [x] #3 modmodel.go and CLAUDE.md give the current reason the model is a QStandardItemModel, and CLAUDE.md no longer lists the translator shim
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Removed the one miqt workaround v0.15.0 made unnecessary: the C++ translator shim.

- translations.go now installs a qt.QTranslator with OnTranslate/OnIsEmpty overrides in Go (installed once; later calls swap the i18n.Translator it asks, as before). translator.cpp/.h and the file's cgo block are gone.
- Verified: go test ./... (TestTranslations runs tr, trN and QML qsTr through Qt, plurals included); the app starts headless with LANGUAGE=uk and loads the Ukrainian translation, no QML warnings.

Kept, with reasons checked against v0.15.0 and written down:
- The mod list model stays a QStandardItemModel: a Go data() override on QAbstractListModel returns a QVariant* that C++ copies but never frees (a leak per cell read), and freeing it through Go's GC could race the copy. modmodel.go and CLAUDE.md now say this instead of the old roleNames reason.
- imagecache and nativetext fill gaps miqt never bound; the list-of-pairs setter bug is still in v0.15.0.

Not run: Windows CI for this branch.
<!-- SECTION:FINAL_SUMMARY:END -->
