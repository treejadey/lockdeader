# Fonts

Bundled with the app (see `fonts.go`), so it looks the same on every platform. Both fonts are licensed under the SIL Open Font License 1.1; each folder holds its licence.

| Font | Used for | Source |
|---|---|---|
| Inter (`Inter/`) | All UI text | [rsms/inter](https://github.com/rsms/inter) v4.1, `extras/ttf/` in the release zip: Light, Regular, Italic, Medium, SemiBold, Bold and Bold Italic |
| JetBrains Mono (`JetBrainsMono/`) | Times in the sound player, code in mod descriptions | [JetBrains/JetBrainsMono](https://github.com/JetBrains/JetBrainsMono) v2.304, `fonts/ttf/` |

## Changes to Inter

The bundled Inter files have Inter's stylistic set `ss04`, "Disambiguation (no slashed zero)", built in, so letters that are easily confused look different: a lower-case l with a tail, an upper-case I with serifs, and an alternate ß. Zero stays unslashed; `ss02` would slash it too.

Qt can only turn OpenType features on per font through `QFont::setFeature`, which miqt doesn't bind, and that wouldn't reach rich text anyway. So instead, each character the set changes is pointed straight at its alternate glyph in the `cmap` table. All of `ss04`'s substitutions are single, one-glyph-for-another ones, so nothing is lost.

The family is renamed `Inter SS04` (name IDs 1, 4 and 16; the PostScript name, ID 6, becomes `InterSS04-…`; the unique ID, ID 3, gets `;SS04` added), so the app never mixes these files up with an Inter installed on the system, which may lack the change, or be another version. Inter's licence names no Reserved Font Name, so the modified files may keep "Inter" in their name.

To reapply after updating the files, with [fontTools](https://github.com/fonttools/fonttools):

```python
from fontTools.ttLib import TTFont

f = TTFont(path)
gsub = f["GSUB"].table
subs = {}
for record in gsub.FeatureList.FeatureRecord:
    if record.FeatureTag == "ss04":
        for index in record.Feature.LookupListIndex:
            for table in gsub.LookupList.Lookup[index].SubTable:
                subs.update(getattr(table, "ExtSubTable", table).mapping)
for cmap in f["cmap"].tables:
    if cmap.isUnicode():
        for code, glyph in cmap.cmap.items():
            cmap.cmap[code] = subs.get(glyph, glyph)
for record in f["name"].names:
    text = record.toUnicode()
    if record.nameID in (1, 4, 16) and text.startswith("Inter"):
        record.string = "Inter SS04" + text[len("Inter"):]
    elif record.nameID == 6 and text.startswith("Inter-"):
        record.string = "InterSS04" + text[len("Inter"):]
    elif record.nameID == 3:
        record.string = text + ";SS04"
f.save(path.replace("Inter-", "InterSS04-"))
```
