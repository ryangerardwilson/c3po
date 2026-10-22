package o

// Help is the full help for c3po tui o, shown by c3po help -f tui o.
const Help = `c3po tui o is a Vim-inspired terminal file navigator.

global actions:
  c3po tui o help
    show this help
  c3po tui o version
    print the installed c3po version

features:
  launch the file navigator from the current directory
  # c3po tui o
  c3po tui o

  start from a directory or open one or more files
  # <path> [<path> ...]
  c3po tui o ~/Downloads
  c3po tui o ~/notes/todo.txt
  c3po tui o a.txt b.txt

  reveal a path in its parent directory
  # reveal <path>
  c3po tui o reveal ~/Downloads/file.txt

  choose files or directories for another app
  # pick from <dir> [files <exts>|directories] [multiple]
  c3po tui o pick from ~/src files py,md
  c3po tui o pick from ~/Downloads directories
  c3po tui o pick from ~/Pictures files jpg,png multiple

  choose a save destination
  # save from <dir> [extension <ext>]
  c3po tui o save from ~/Downloads extension txt

  edit the user config in $VISUAL, $EDITOR, or vim
  # config
  c3po tui o config

A directory starts the navigator at that directory. Files open in
$VISUAL, $EDITOR, or vim when every one is a text file, and each goes
to xdg-open otherwise. Several targets must all be files.
reveal <path> starts in the parent directory with the target selected.
pick and save print the chosen path(s) to stdout and write them to
${XDG_CACHE_HOME:-~/.cache}/c3po/o/picker-selection.txt.
config opens ${XDG_CONFIG_HOME:-~/.config}/c3po/o/config.json, creating it
first when it is missing.

keys:
  j/k or arrows    move; ctrl+j/ctrl+k jump a tenth of the listing
  l or right       enter a directory, open a file, or confirm a pick
  enter            browse: cd the invoking shell into the listed directory
                   and exit; pick/save: same as l
  h or left        parent directory
  /                enter or clear the filter
  r                reload the listing now
  .                show or hide dot entries (hidden by default)
  = / -            increase or decrease the nesting preview depth
  m                mark or unmark the current item
  y                copy marked paths, or the current path, with wl-copy
  n                cd the invoking shell into the current directory and exit
  ~                home directory
  ?                toggle the key help
  q                quit

Listings reveal in a staged cascade. Text-file labels brighten with their
line count relative to the largest text file shown. The listing is polled
every 500ms, so outside creations, deletions, renames, and edits appear on
their own.

Shell cd handoff: when O_SHELL_CD_FILE names a file, enter and n write the
chosen directory to it, so a shell wrapper can cd there after exit.

Exit 0 on success, 1 on bad usage, a missing path, or a terminal error.
A path and config exit with the editor's status.
`

// Usage is printed after a malformed command.
const Usage = "Use: c3po tui o | c3po tui o <path> [<path> ...] | c3po tui o reveal <path> | c3po tui o pick from <dir> [files <exts>|directories] [multiple] | c3po tui o save from <dir> [extension <ext>] | c3po tui o config"
