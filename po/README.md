# Translations

Lockdeader is translated with gettext PO files. 

It displays text in the first of your system's languages that has a translation
here, and in English otherwise.

- `lockdeader.pot` is the template: all the text in the app, untranslated.
  It's made from the source, so don't edit it by hand.
- `<language>.po` is one language's translation, e.g. `de.po`, `pt_BR.po` or
  `zh_Hant.po`.

## Translating

1. **Start a language** that isn't here yet by making its `.po` from
   `lockdeader.pot`, with Poedit (File › New from POT/PO File), Weblate, or
   `msginit -i lockdeader.pot -l de -o de.po`. Any of them fills in the
   language's plural forms. If you copy the template by hand instead, fill in
   the `Language` and `Plural-Forms` header lines yourself.
2. **Translate** with any PO editor, or a text editor. A few things to know:
   - Keep placeholders such as `%1`, `%s` and `%d`. In Go's text (`%s`,
     `%d`), use `%[2]s` to change the order they come in.
   - `%n` is a number. Messages with a plural have a form for each of your
     language's plural forms. A form only used for 1 may leave `%n` out.
   - Comments starting `#.` say what a placeholder stands for, or where the
     text appears.
   - Translations marked fuzzy aren't used until you've checked them and
     taken the mark off.
3. **Check it** from the repository's root:

   ```sh
   go run ./cmd/translations check
   ```

4. **Try it** by building the app and running it in your language, e.g.
   `LANGUAGE=de ./lockdeader` on Linux. On Windows, change the display
   language in Settings.

## For developers

After changing any text in the app, run:

```sh
go run ./cmd/translations extract
```

This updates `lockdeader.pot`, and brings every `.po` up to date with it. It
keeps existing translations, leaves new text untranslated, and keeps
translations of removed text at the end of the file as obsolete (`#~`), in
case the text comes back. It never translates anything.

The repository has a pre-commit hook that runs `check` on what's staged, so
a commit can't leave the template behind the text. Turn it on once in your
clone:

```sh
git config core.hooksPath .githooks
```
