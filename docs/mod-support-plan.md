# Plan: better mod support

How Lockdeader's mod support should improve on the current approach (2026-10-10). [deadlock-modding.md](deadlock-modding.md) describes how the game loads mods and how Deadlock Mod Manager (DMM) works; this builds on it.

## What's wrong with the current approach

DMM keeps all of its state in the game folder:

- load order is encoded in `pakNN` filenames;
- disabled mods sit in `addons/` under other names;
- every change is a chain of renames, backed by staging files and rollback logic so it can recover from a crash;
- profiles and vanilla launches work by rewriting `gameinfo.gi`, using a built-in copy of the `SearchPaths` block. That copy has already drifted from Valve's (its "vanilla" block isn't the stock one).

Conflicts are resolved a whole mod at a time. Breakage after a game update goes unnoticed.

## Principles

- **Windows is the priority; Linux should be near-perfect.** Most players are on Windows; Linux must not be second-class.
- **Behaviour is the same on every OS.** No feature, setting or wording depends on the platform. Where the implementation has to differ, it stays internal and invisible to users.
- **Never touch files we didn't create.** That includes VPKs added by hand or by DMM. Lockdeader keeps its files in a folder of its own, so this holds by design.
- **Nothing in the game folder is the only copy of anything.** It can all be rebuilt from our data dir, which keeps crashes harmless (see Crash safety).

## 1. Store mods outside the game, sync our own folder to match

**Store.** Every mod VPK lives in our data dir, under `Cache/Mods/` (already created by `appdata`), and is named by its SHA-256.

- Mod VPKs are single files that never change, so they suit being stored by content.
- Two profiles sharing a mod share one copy.
- Updating a mod adds a new file; the old one is garbage-collected once nothing uses it.

**Manifest.** Profiles, enabled mods and load order live in our data dir, not in the game folder: in the database, `Lockdeader.db` (package `appdb`). Its `mods`, `mod_files` and `mod_vpks` tables record each installed mod, the GameBanana archives it came from (file id, size and MD5) and the VPKs taken out of them (path in the archive and SHA-256).

**Our folder.** The game loads our mods from `game/citadel/lockdeader/`, not from the shared `addons/`:

- Everything in it is ours by definition, so there's never any doubt about which files we own, even after a crash.
- Files that the user or DMM put in `addons/` are never at risk.
- The whole folder can be replaced in one step (see Crash safety).
- VPKs already in `addons/` can be imported into the store if the user wants Lockdeader to manage them.

**Apply.** Applying a profile works out what our folder should contain, compares it with what's there, and makes only the changes needed:

- Each enabled mod becomes a `pakNN_dir.vpk` file, numbered by load order.
- Running apply twice does nothing the second time.
- Recovering from a crash is just running apply again on startup. That replaces DMM's staging files and rollback code.
- Switching profiles only changes what's in our folder. `gameinfo.gi` isn't touched.
- Apply is refused while Deadlock is running. On Windows, files the game has open can't be renamed or deleted anyway.

### Putting files into our folder, and Windows

Symlinks are out: on Windows they need admin rights or Developer Mode. Hard links don't.

- **On Windows**, `CreateHardLink`, which Go's `os.Link` uses, works for normal users on NTFS.
- **On Linux**, hard links work the same way.

So one strategy covers both OSes. Each file goes into our folder by the first of these that works:

1. **Hard link.** Instant, and takes no extra space. Needs the store and the game on the same volume.
2. **Reflink** (Linux, btrfs/XFS, via `FICLONE`). Instant, and takes no extra space. Works across subvolumes, but not across filesystems.
3. **Copy.** Always works, but takes the space of each enabled mod twice.

The choice depends on the drive layout, not the OS, and nothing in the UI mentions links. The only thing a user could notice is disk usage when copies are used.

**Open question: where the store lives.** On Windows, the Steam library is often on another drive (`D:\SteamLibrary`) while our data dir is on `C:`. That puts many Windows users on the copy fallback.

- Option: when the game is on another volume, keep the store on that volume, e.g. `<library>\Lockdeader\`.
- Downside: one more place where Lockdeader keeps files.
- Either way, exFAT drives can't hard-link at all, so copying has to remain a supported fallback.

## 2. Add only our own lines to `gameinfo.gi`

- **Insert** our marked lines directly above the first `Game citadel` line, and leave everything else alone. Valve's changes to the block (it has changed it four times since 2024) then don't affect us.
  - `Game citadel/lockdeader` always.
  - `Game citadel/addons` too, but only if the user wants mods installed outside Lockdeader to keep loading. It goes after ours, so our mods win any conflicts.
  - A line the user added by hand isn't ours: we never remove or reorder it.
- **Remove** our lines to launch vanilla; the file is then exactly what it was before.
- **Watch for game updates** through the `buildid` in `appmanifest_1422450.acf`. Re-insert our lines as soon as an update resets the file, not at the next launch.
- **Handle Windows quirks:** clear the read-only attribute that some guides tell users to set, and back up the file in our data dir before the first write.
- **Profiles never touch this file** (see 1). It only changes after a game update, or when switching between modded and vanilla.
- **DMM** replaces the whole `SearchPaths` block, which removes our lines. Detect its `// Deadlock Mod Manager` markers and tell the user, instead of fighting over the file.

## Crash safety

**What a crash can do.** At worst, the game loads an older or partial set of mods (or none) until Lockdeader starts again. It can't lose a mod or stop the game from starting. This rests on four rules:

1. **Our data dir holds everything that matters:** the store and the manifest. Both are written safely:
   - write a temporary file;
   - flush it to disk (`f.Sync()`, which is `FlushFileBuffers` on Windows);
   - rename it over the real file.

   A crash leaves either the old file or the new one, never half of each. The flush matters for power loss: without it, the rename can reach the disk before the data does, leaving an empty file.
2. **Everything in the game folder is rebuilt from the data dir:** the files in our folder, the generated conflict VPK (3), and our lines in `gameinfo.gi`. Any of it can be deleted and recreated. This is the main difference from DMM, where `addons/` is the storage, so a half-finished chain of renames can lose track of mods.
3. **Every half-finished state is harmless.** The game still starts, just with an older or partial set of mods.
4. **Recovery is the normal startup path:** delete our leftover temporary files, then apply the active profile again.

**What a crash leaves behind, by moment:**

| Crash during | Left behind | On next start |
|---|---|---|
| A download | a `.partial` file | resume it, or delete it |
| Extraction / adding to the store | temporary files. Store files only get their final hash-based name once the hash has been checked, so the store itself is never affected | delete them |
| Saving the manifest | the old or the new manifest | nothing to do |
| Applying to our folder | `lockdeader.next/` and/or `lockdeader.old/` (see the swap below) | finish or undo the swap, then apply again |
| Editing `gameinfo.gi` | the old or the new file (written the same safe way) | check our lines are there |
| The game running | nothing: we never write while it runs, and the game only reads these files | nothing to do |

**All-or-nothing apply.** Applying builds the new state in `citadel/lockdeader.next/`, then swaps it in:

1. Rename `lockdeader` to `lockdeader.old`.
2. Rename `lockdeader.next` to `lockdeader`.
3. Delete `lockdeader.old`.

- **If a crash hits between steps 1 and 2,** there is no mod folder, so the game simply launches vanilla. The next start finishes the swap.
- **Building the new folder is cheap.** Mods that haven't changed are hard-linked from the current folder, which is always on the same volume. So only new mods are linked or copied from the store.
- **On drives that can't hard-link (exFAT),** unchanged mods are moved instead. The swap is then no longer all-or-nothing, but a crash still leaves a harmless partial state.

**`gameinfo.gi`** is the one file where a mistake would stop the game from starting. Before writing, check that the edited text is still valid. Keep the backup in our data dir. Even in the worst case, Steam's "Verify integrity of game files" restores it.

**Windows.** Antivirus or Steam can briefly hold a file open, which makes renames and deletes fail. Retry them a few times with a short backoff, as DMM does in `fs_retry.rs`.

**Detecting that the game has exited.** A game crash leaves our files untouched. The only thing to get right is noticing that the process is gone, so we stop refusing to apply.

### Mods that crash the game

- **Launching vanilla is instant:** it's just removing our lines from `gameinfo.gi`.
- **Last known good:**
  - Record which setup was applied at each launch.
  - If the game exits abnormally soon after starting, offer to go back to the last setup that ran without a crash, and show which mods changed since then.

## 3. Per-file conflict resolution

DMM's mod higher in the load order wins every file the two mods share. When they overlap only partly, the result is a broken mix, e.g. one mod's model with the other's materials.

The fix: generate a small `pak01_dir.vpk` that holds only the contested files, using whichever version the user picked for each. The lowest pak number wins, so it takes priority over every other mod.

- It's only a few MB, and it's rebuilt only when the conflicts or the user's choices change.
- Default choices follow load order. Users only touch the files they want to change.
- Warn about partial overlaps: when mod A wins some of the files it shares with B and loses others.

This needs a Go VPK reader and writer (see 7).

## 4. Warn when a game update may have broken a mod

**Locally, for installed mods:**

- Keep a snapshot of each game build's file list: path and CRC32 for every file in `citadel/pak01_dir.vpk`. Only the file list is read, not the file data.
- When the build changes, check every path that an installed mod replaces:
  - **changed in the base game:** "This update changed files this mod replaces; it may be broken";
  - **removed from the base game:** "This mod no longer does anything".

**Before downloading.** Answering "does this mod still work with the current game?" for mods not yet downloaded needs our own API (see Future).

## 5. Work out what a mod is from its contents

Reading a VPK means downloading it first. So this improves installed mods, not browsing. Before download, we're stuck with GameBanana's metadata, as DMM is.

Once a mod is downloaded, its file list tells us:

- **Hero:** from the paths it replaces (heroes' files are organised per hero), instead of DMM's guesses from category names and mod names.
- **Mod type:** from the file extensions: skin, sounds, UI or particles.
- **Matchmaking:** mods that change gameplay data (`*.vdata_c`) only work on custom servers. Label them that way.
- **Variants:** for archives with several VPKs, show what each one changes before the user picks.

Showing any of this while browsing needs our own API (see Future).

## 6. Integrity check

A "check integrity" action, like Steam's "Verify integrity of game files", that confirms every installed mod is intact and repairs any that aren't, without the user having to reinstall anything. The database already stores what it needs.

- **Check** each VPK in the store against the SHA-256 recorded in `mod_vpks`. A file that's missing or has a different hash is damaged. Record each pass in `mod_vpks.verified_at`.
- **Repair** a damaged VPK by downloading its archive again: the GameBanana file id in `mod_files` says which one, its MD5 says whether the download is intact, and the VPK's path in the archive says which file to take out. Check its SHA-256 before putting it back in the store.
- **Our folder** (1) is rebuilt from the store afterwards, so repairing the store repairs the game too.
- **When the archive is gone from GameBanana** (deleted or replaced by the author), say so, and offer to look for the mod's current files instead.
- Hashing every VPK takes a while for big setups, so run it in the background with progress, and maybe check only the enabled mods at startup.

## 7. Smaller things

- **VPK reader and writer in Go.** The format is small: a 28-byte header, a file list, and the file data stored inside the `_dir.vpk`. Items 3, 4 and 5 all need the reader; item 3 also needs the writer.
- **Updates.** Use the file id and `_sMd5Checksum` from `DownloadPage` instead of comparing timestamps. Verify the MD5 after each download.
- **Chunked VPKs** (`_000.vpk`). Detect them and reject them with a clear error, instead of breaking them the way DMM would.
- **Mod limit.** Without item 3's merging, it's 99 mods per profile, one per `pakNN` number. If that's ever too few:
  - add more folders of our own (`lockdeader2/`, …), with a line each in `gameinfo.gi`;
  - or merge the whole profile into one generated VPK.

## Future: our own API

The insights from 4 and 5 only help while browsing if someone has already downloaded and indexed the mods. A small service could:

- index the VPK file lists of GameBanana's Deadlock files: paths and CRCs, which is small;
- serve per-mod data: hero, mod type, whether it changes gameplay data, and which files it replaces;
- compare those file lists against each game build's file list, to flag likely-broken mods before anyone downloads them;
- show conflicts against a user's installed mods before they download.

DMM's backend already does something similar (`/api/v2/vpk-analyse-hashes`). This is noted for later and isn't needed for 1–3.

In addition to the api I think it would also be fair if we provided our own caching in front of the gamebanana servers to make their life a bit easier.

## Order of work

1. `DownloadPage` in `banana`; downloading, MD5 checks and extraction into the store, recorded in the database.
2. The VPK reader.
3. The manifest and apply for our folder (1), the `gameinfo.gi` edit (2), and startup recovery (Crash safety).
4. The integrity check (6).
5. Mod details from contents (5), then update warnings (4) and last known good.
6. Per-file conflict resolution (3), which needs the VPK writer.
