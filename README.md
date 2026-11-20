# c3po

c3po is the agent CLI. Agents use those commands and do not open a screen. The apps are `ast`, `clippy`, `deport`, `gcal`, `gdrive`, `gmail`, `idealake`, `looptab`, `notify`, `pass`, `slack`, and `tunnel`. `r2d2` is the terminal program. Running `r2d2` opens a screen menu. The screens are `web`, `cyclotree`, `l`, `o`, `tunnel`, `gdrive`, `pass`, `deport`, `xl`, `looptab`, and `slack`. `r2d2 web` is the ryangerardwilson.com admin screen. There is no `c3po tui` command. The neuro-linguistic programming trainer is the crusade mode `crusade -m nlp`. `c3po pass` runs `nebula crypt`. `r2d2 pass` stays the item screen and reaches 1Password through `nebula crypt`. `c3po tunnel` and `r2d2 tunnel` run `nebula tunnel`.

```sh
c3po ast [args...]
c3po clippy [text...]
c3po deport list|<port>
c3po gcal [args...]
c3po gdrive [args...]
c3po gmail [args...]
c3po idealake add|list|search|contains|show|remove|topics
c3po looptab [args...]
c3po notify [--urgency low|normal|critical] heading "<title>" body "<message>"
c3po pass <vault> <item> [args...]
c3po slack [args...]
c3po tunnel add|remove|list|start|stop|status|logs|run
c3po usage [prefix]
c3po help [-f <category> [feature]]
```

Help for every tool is `c3po help`. `c3po help -f ast` lists the ast commands, and `c3po help -f ast patch` explains one of them. `c3po help -f <category>` explains that app. Screen help is `r2d2 <screen> help`.

`c3po` keeps its records in its own sqlite file. [nebula](https://github.com/ryangerardwilson/nebula) `s3` backs that file up. `c3po usage` lists every catalog feature that has never been used, then the ones that have, with separate counts for a human, an agent, and a service. Each run adds to a daily counter in that sqlite and returns without waiting. A background worker publishes the file with nebula `s3`. The record does not store arguments.

`ast` is the AST-aware agent boundary: trace a symbol, then create, patch, and apply files by hash. `cyclotree` shows the calls of one function. With no symbol, the tree starts at the program entrypoint and includes every call reached from it. Each row is that function, its cyclomatic complexity, and how many levels of calls can be opened under it. The screen is white and updates as those files change. A nested row shows its file relative to the caller. The bottom shows the call chain from the root to the row under the cursor, then that row's full path. `enter` opens that file in vim at its line. `?` toggles the shortcut list. `j` and `k` move, `l` opens the calls, and `h` moves to the caller. A call back to a function already on the path is marked `cycle` and stays closed.

The installed names are `c3po` and `rgw-ast`. `rgw-ast` is the same binary, for tools that still launch that command. `xl` is a screen, `r2d2 xl`, and is not a separate command. `c3po idealake` stores one-line ideas of at most 300 characters by topic. `search` ranks one topic by meaning. `contains` matches the lowercase text. `c3po clippy` copies a multiline text to the Wayland clipboard and does not print it. `c3po notify` sends an omarchy-shell toast through the `ryan.notify` overlay, which `install.sh` installs from `notify/plugin/`. `c3po looptab` registers jobs files and edits them in vim. `r2d2 looptab` lists Looptab jobs. Enter runs the selected job once through `looptab now`. `r2d2 slack` shows the Slack desk. `h` and `l` switch workspace. The table is direct messages, tags, and what to know. Enter opens the row. `r` marks it read on the desk. Opening a row does not mark it read. `s` syncs the desk from Slack. The screen also syncs about every 10 minutes. The screen shows when the desk last synced. `cx` and `complexity` are not installed.

## Install

Public source is https://github.com/ryangerardwilson/c3po. Release tags are `vX.Y.Z`. The source version is 0.5.0. The published tag stays v0.4.0 until this version is tagged.

From a checkout:

```sh
./install.sh
```

The binary is installed to `$C3PO_INSTALL_DIR` or `~/.local/bin`, with an `rgw-ast` symlink beside it. An `xl` symlink that points at this binary is removed. A standalone `~/.local/bin/notify` binary or symlink is removed, and the `ryan.notify` omarchy-shell plugin is copied from `notify/plugin/` to `${XDG_CONFIG_HOME:-$HOME/.config}/omarchy/plugins/ryan.notify/`. A clean checkout reports `c3po 0.5.0+<commit>`. Uncommitted changes add `-dirty`.

Go installs the same module without the `rgw-ast` link:

```sh
GOBIN="$HOME/.local/bin" go install github.com/ryangerardwilson/c3po/cmd/c3po@v0.4.0
ln -sfn c3po "$HOME/.local/bin/rgw-ast"
```

## Develop

```sh
GOWORK=off go test ./...
```
