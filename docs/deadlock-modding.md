# How Deadlock modding works

State of things as of 2026-10-10. This covers how the game loads client mods, how [Deadlock Mod Manager](https://github.com/deadlock-mod-manager/deadlock-mod-manager) (DMM, deadlockmods.app) drives that, and how it talks to GameBanana. Anything marked *(community)* comes from testing or from mod-manager code, not from Valve. Valve publishes no modding documentation.

## The game side

Deadlock is a Source 2 game, Steam app id `1422450`. It is Windows-only and runs through Proton on Linux. Its files live in the Steam library, not in the Proton prefix:

```
<steam library>/steamapps/common/Deadlock/game/citadel/
    gameinfo.gi       search paths (and other engine config)
    pak01_dir.vpk     base game content (+ pak01_000.vpk … chunks)
    addons/           mod VPKs; doesn't exist until a user or manager creates it
```

### gameinfo.gi

Mods load only if `gameinfo.gi` puts an addons directory on the search path. Out of the box it doesn't. This is the stock block (comments removed) as of the 2026-09-29 update:

```
SearchPaths
{
    Game_UILanguage     citadel_*LANGUAGE*
    Game_LowViolence    citadel_lv
    Game                citadel
    Game                core
}
```

The modded version adds `Game citadel/addons` before `Game citadel`. This is DMM's block:

```
SearchPaths
{
    Game_UILanguage     citadel_*LANGUAGE*
    Game_LowViolence    citadel_lv
    Game                citadel/addons
    Mod                 citadel
    Write               citadel
    Game                citadel
    Mod                 core
    Write               core
    Game                core
}
```

- **The addons line must come before `Game citadel`.** The `Mod` and `Write` lines matter too: without them, the engine takes the first `Game` path as its mod and write directory, reads `cfg/user_keys_default.vcfg` from there (and fails to start: "Unable to read default keybinding configuration") and writes its shader cache and reports there. `Mod citadel` and `Write citadel` keep those at `citadel`.
- **Game updates and Steam's "Verify integrity" overwrite the file,** which silently turns mods off. Managers have to check the file and re-apply their edit before every launch.
- **Valve has no official route for mods.** There is no `-addon` launch flag, no Workshop, and no stock addons search path. The engine binaries do contain CS2's generic addon code (`ADDONROOT`, "untrusted addon" strings), but Deadlock doesn't expose any of it.

### Mounting and priority

- **Which files load.** From each search-path directory, the engine mounts only files named `pak01_dir.vpk` … `pak99_dir.vpk` (`pak##_dir.vpk`). Any other filename is ignored, and `pak100_dir.vpk` and above are never loaded. *(community; the binary contains the `pak*_dir.vpk` and `pak%02d.vpk` patterns)*
- **Which file wins.** For each file path, the first mounted VPK that contains it is used: *(community)*
  - an earlier search-path line beats a later one;
  - within one directory, a **lower pak number beats a higher one**, so `pak01` overrides `pak02`.
  - Many guides claim that higher numbers "load later and override". That is wrong.
- **No clash with the base game.** `addons/pak01_dir.vpk` doesn't collide with `citadel/pak01_dir.vpk` because they are in different directories.
- **More than 99 mods.** You need another directory with its own `Game` line; DMM uses `citadel/addons2` … `addons10`.

### VPK format (v2)

- **Header:** seven little-endian `uint32`s, 28 bytes in total:
  - magic `0x55AA1234`
  - version `2`
  - tree size, file-data size, archive-MD5 size, other-MD5 size, signature size
- **Directory tree:** nested null-terminated strings, in the order extension → directory → filename. Each file entry is:

  ```
  CRC32 u32, PreloadBytes u16, ArchiveIndex u16, Offset u32, Length u32, 0xFFFF
  ```

- **Where the data is.** `ArchiveIndex = 0x7FFF` means the data is stored in the `_dir.vpk` itself, after the tree. Any other value points to a `pak##_NNN.vpk` chunk.
- **Mods are single files.** A mod is one self-contained `_dir.vpk`; only the base game uses chunks. A mod's identity is therefore just one file that can be renamed freely.
- **Conflicts can be found cheaply.** The tree alone gives every file's path and CRC32, so conflict detection never needs to read file data.

Reference implementation: [ValvePak](https://github.com/ValveResourceFormat/ValvePak).

### What mods can change

- **Compiled assets (`*_c`):** models (`vmdl_c`), materials (`vmat_c`), textures (`vtex_c`), particles (`vpcf_c`), animations, sounds, and Panorama UI (XML/CSS/JS, for the HUD and menus). UI mods are the ones most often broken by patches.
- **VData (gameplay data):** the server is authoritative, so changes only take effect on your own listen or dedicated server, not in Valve matchmaking.
- **Bans:** Valve has published no policy. No bans for cosmetic mods are known. *(community)*

## Deadlock Mod Manager

DMM is a Tauri app: a React frontend over a Rust backend. The paths below are relative to `apps/desktop/src-tauri/src/` in its repo.

### Finding the game

- `steamlocate` finds Steam, then app `1422450`, then the library it's installed in (`mod_manager/steam_manager.rs`).
- On Linux it also tries these Steam roots:
  - `~/.local/share/Steam`, `~/.steam/{steam,root,debian-installation}`
  - Flatpak (`~/.var/app/com.valvesoftware.Steam/…`)
  - snap
- A game path the user sets by hand is only accepted if `game/citadel/gameinfo.gi` exists under it.

### Managing gameinfo.gi

This lives in `mod_manager/game_config_manager.rs`.

- **Its own block.** DMM's `SearchPaths` block sits between `// Deadlock Mod Manager - Start` and `// Deadlock Mod Manager - End`. If the markers are missing (after a patch), it replaces the file's first `SearchPaths { … }` block.
- **Backup.** `gameinfo.gi.bak` is written once and never overwritten.
- **Safe writes.** Each write goes to `gameinfo.gi.tmp` first and is verified there. If the check fails, DMM restores the backup.
- **Detecting state.** It counts the file as modded if it contains the text `citadel/addons`.
- **Profiles and shards.** There is one `Game` line per shard directory, and the first line has the highest priority.
- **Vanilla launch.** DMM rewrites the block to one with no addons line. That block still keeps the `Write citadel`/`Write core` lines, so it isn't byte-identical to the stock file.

### Files on disk

| What | Where / how |
|---|---|
| Downloads | `<app local data>/mods/<modId>/`; archives (zip, rar, 7z) are extracted there, then the archive and extracted tree are deleted |
| Disabled mod | `addons[/<profile>]/{modId}_{originalName}.vpk`. Not mounted, because the name doesn't match `pak##_dir.vpk` |
| Enabled mod | `addons{N}[/<profile>]/pakNN_dir.vpk` |
| Ownership ledger | `.dmm.json` (version 3) in each profile directory: per mod `{enabled, order, shard, currentVpks, disabledVpks, originalVpkNames}` |
| In-progress operations | `.dmm-reorder/`, `.dmm-clear`, `.dmm-update-<id>`, `s<shard>__<file>.pending`; recovered on the next write |
| Ignored conflicts | `.dmm-conflicts.json` |

- **Mod ids:**
  - the GameBanana id (`123456`);
  - `snd-<id>` / `wip-<id>` for sounds and WIPs;
  - `local-<uuid>` for imported files.
- **Enable and disable** are plain `fs::rename` calls between the disabled name and the `pakNN` name, rolled back if any file fails (`vpk_manager/operations.rs`). No symlinks or hard links are involved.
- **Archives with several VPKs.** Every `.vpk` found anywhere in an archive counts as a mod file. If there's more than one, the user picks which (variants).
- **Chunked VPKs.** There is no handling for `_000.vpk`-style chunks: each file would be renamed on its own.

### Load order

This lives in `shard.rs`, `vpk_manager/{naming,reorder}.rs` and `manager/reorder.rs`.

- **Shards.** A shard is one search-path directory: `addons`, `addons2` … `addons10`. Each holds at most 99 mods.
- **Installing.** A new mod goes into the last shard that has room; otherwise DMM opens a new shard. Within that shard it takes the lowest free `pak##` number.
  - DMM never back-fills older shards, because that would move the new mod ahead of mods in later shards.
- **Reordering.** DMM renumbers everything, starting at `pak01` in each shard, and never splits one mod's VPKs across shards. Earlier in the list means higher priority.

### Profiles

- **Where profiles live.** The default profile is `addons/` itself. Every other profile is a subdirectory, `addons/profile_<id>_<name>`, with matching `addonsN/profile_<id>_<name>` directories for its overflow shards.
- **Switching.** Nothing on disk moves; DMM rewrites the `Game` lines in `gameinfo.gi` to point at the other profile's directories.

### Conflicts

This lives in `mod_manager/conflicts.rs`.

- **How they're found.** DMM parses only the VPK directory trees and normalises paths (lowercase, `/`). Two mods conflict when both contain the same path with a different CRC32. The mod with the lower `(shard, pak number)` wins.
- **Ignored as noise:** `addoninfo.txt`, `modinfo.json`, readme/licence/credits files, `panorama/image_compiler.vdata_c` and `materials/default/default_*`.
- **Severity:**
  - critical: `vmdl_c`;
  - normal: `vmat_c`, `vtex_c`, `vpcf_c` and animation graphs;
  - low: everything else.

### Launching

- **The URI.** DMM opens `steam://run/1422450//<args>`:
  - on Windows, through `steam.exe`;
  - on Linux, through `steam` (falling back to `xdg-open`);
  - on macOS, through `open`.
- **No mod-specific launch flags.** The only argument DMM adds itself is `-condebug`, under Flatpak, so it can read the console log.
- **Vanilla vs. modded.** A vanilla launch disables mods in `gameinfo.gi` first. A modded launch re-applies the active profile's search paths first.

## GameBanana integration

DMM's desktop app talks to GameBanana directly (`providers/gamebanana/`). Its own backend, `api.deadlockmods.app`, is only used for extras:

- announcements, shared profiles, crosshairs and the server list;
- identifying loose VPKs by hash (`/api/v2/vpk-analyse-hashes`).

### Endpoints

All of these are relative to `https://gamebanana.com/apiv11/`. Deadlock's game id is `20948`.

| Purpose | Request |
|---|---|
| List | `{Mod\|Sound\|Wip}/Index?_aFilters[Generic_Game]=20948&_nPerpage=50&_nPage=N&_sSort=Generic_LatestModified` (or `Generic_Oldest` for a full crawl; pages 1–250) |
| Details | `{Model}/{id}/ProfilePage` |
| Files | `{Model}/{id}/DownloadPage` → `_aFiles` / `_aArchivedFiles`, each with `_idRow`, `_sFile`, `_nFilesize`, `_sDownloadUrl`, `_sMd5Checksum`, `_tsDateAdded`, `_bIsArchived`; plus `_bIsTrashed` / `_bIsWithheld` |
| Mirrors | `Util/Fileservers?_nPage=1` |
| Bulk update check | legacy `https://api.gamebanana.com/Core/Item/Data?itemtype[]=Mod&itemid[]=X&fields[]=…`, one `itemtype`/`itemid`/`fields` triple per item, up to 50 items, fields `mdate,Files().aFiles()` |

### Categories and heroes

- **No category ids.** DMM uses category *names*: the first non-empty one of `_aSuperCategory`, `_aRootCategory` and `_aCategory`.
- **Heroes:**
  - For mods under the "Skins" super-category, the category name is the hero. It is matched against an alias table that includes internal codenames (`atlas` = Abrams, `inferno` = Infernus).
  - For anything else, DMM guesses the hero from the mod's name.

### Downloads and updates

- **Downloading:**
  - Downloads resume through a `.partial` file and HTTP `Range`, and are checked against `_sMd5Checksum` when they finish.
  - Only HTTPS hosts under `gamebanana.com` or `deadlockmods.app` are allowed, including on redirects.
- **Mirrors.** DMM picks a mirror by rewriting the download URL to `https://{_sDomain}.gamebanana.com/mods/<_sFile>`. "Auto" picks the healthy server with the best throughput.
- **Which files are installed.** DMM saves the ids of the files the user picked (`selectedFileIds`). Internally a file is referred to as `gamebanana-file://{slug}/{fileId}`.
- **Updates.** A mod counts as updated when one of its selected files has a `_tsDateAdded` later than the install time, or when a selected file has disappeared. Without saved file ids, DMM compares `mdate` instead.
- **Local catalog.** DMM keeps a local SQLite copy of the catalogue:
  - every 6 h, an incremental sync walks `Generic_LatestModified` until it reaches the newest date it already has;
  - every 7 days, a full reconciliation.

### One-click install

GameBanana's install button opens a custom URL scheme. DMM handles `deadlock-mod-manager:`, `deadlock-modmanager:` and `dlmm:`:

```
deadlock-mod-manager:https://gamebanana.com/mmdl/<fileId>,<Mod|Sound>,<itemId>
```

The file id is read from the `/mmdl/(\d+)` part of the URL.

### Client etiquette

- User-Agent `DMM/<version>`.
- At most 10 requests per second and 4 at once.
- Timeouts: 10 s to connect, 30 s in total.
- 3 retries with jittered backoff, and 429 `Retry-After` is honoured.
- The JSON parser tolerates leading PHP warnings in GameBanana's responses.

## Other tools

| Tool | What it is |
|---|---|
| [Grimoire](https://github.com/Slush97/grimoire) | Electron mod manager; GameBanana browsing, enable/disable/reorder |
| [deadlock-modbox](https://github.com/rszyma/deadlock-modbox) | Nix-based: mods kept as patches and merged into one `pak75_dir.vpk`, so collisions are caught at build time |
| [Source 2 Viewer](https://s2v.app) | Browse, decompile and extract VPKs |
| [CSDK 12](https://deadlockmodding.pages.dev/modding-tools/csdk-12) | CS2 Workshop Tools adapted for Deadlock; compiles assets and packs VPKs |
| [vpkmerge](https://github.com/Slush97/vpkmerge) | Merges or splices VPKs |
| [Deadworks](https://github.com/Deadworks-net/deadworks), [modlock](https://github.com/paralin/modlock) | Frameworks for modding game servers, not client mod loaders |

## What this means for Lockdeader

- **gameinfo.gi:**
  - Lockdeader needs its own marked block in `gameinfo.gi`, with a one-time backup.
  - It has to re-check and re-apply that block before every launch, because patches reset the file.
- **Load order and disabling:**
  - Load order is the `pak##` number (lower wins), with a hard limit of 99 per directory.
  - The simplest way to disable a mod is to rename it out of the `pak##_dir.vpk` pattern.
- **Coexisting with DMM:**
  - Detect a `.dmm.json` file and the `// Deadlock Mod Manager` markers.
  - Don't renumber or delete VPKs that Lockdeader didn't create.
- **Conflict detection** only needs each VPK's directory tree: paths plus CRC32s.
- **API client.** `banana.Client` already covers `Mod/Index` and `ProfilePage`. Installing needs `DownloadPage` (files, MD5s, `_tsDateAdded`). Choosing a mirror would need `Util/Fileservers`, and bulk update checks would need `Core/Item/Data`.

## Sources

- Stock `gameinfo.gi`: [SteamDatabase/GameTracking-Deadlock](https://github.com/SteamDatabase/GameTracking-Deadlock/blob/master/game/citadel/gameinfo.gi)
- DMM source: [deadlock-mod-manager/deadlock-mod-manager](https://github.com/deadlock-mod-manager/deadlock-mod-manager):
  - `mod_manager/game_config_manager.rs`, `shard.rs`, `conflicts.rs`, `vpk_manifest.rs`
  - `vpk_manager/`, `providers/gamebanana/`, `deep_link.rs`
- DMM docs: [docs.deadlockmods.app](https://docs.deadlockmods.app/using-mod-manager/getting-started)
- VPK format: [ValvePak](https://github.com/ValveResourceFormat/ValvePak)
- Community guides:
  - [deadlockmodding.pages.dev](https://deadlockmodding.pages.dev/installing-mods)
  - [Complete Guide to the Modding Deadlock Workflow](https://github.com/Sqooky/Complete-Guide-to-the-Modding-Deadlock-Workflow)
