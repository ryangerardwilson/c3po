# vibetree

## Purpose

`c3po scout vibetree` shows where the size of a tree sits. A module is a file or a directory that contains one. It is not a Go package, a Python package, a Rust crate, or a TypeScript module. A directory's size is the sum of the files under it. `-m loc` counts lines. `-m cyclo` sums the cyclomatic complexity of the functions in each file. The report has no grade.

## Requirements

### Requirement: Commands and exit codes
The scout SHALL expose `c3po scout vibetree <dir> [-p N] [-m loc|cyclo] [--json]` and `c3po scout vibetree help`. `<dir>` MAY come before or after the flags and SHALL default to the working directory. `-p` SHALL be an integer from 1 to 99 and SHALL default to 95. `--percentile` SHALL be the same flag. `-m` and `--mode` SHALL be `loc` or `cyclo` and SHALL default to `loc`. The attached forms `-mcyclo` and `--mode=cyclo` SHALL be the same flag. An unknown flag, a percentile outside 1 to 99, an unknown mode, or more than one path SHALL exit 2 and SHALL print help. A missing path or a path that is not a directory SHALL exit 1. A tree that can be read SHALL exit 0, including a tree with no modules. The name `visualize` SHALL NOT select this scout.

#### Scenario: Help names the percentile and the mode
- **WHEN** the operator runs `c3po scout vibetree help`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL name `-p` and `-m`
- **AND** the text SHALL say the percentile defaults to 95 and the mode defaults to loc
- **AND** the text SHALL name `--json`

#### Scenario: No flags use p95 and loc
- **WHEN** the operator runs `c3po scout vibetree` with no flags
- **THEN** the report SHALL begin with `mode loc`
- **AND** the cutoff line SHALL name `p95`

#### Scenario: A bad percentile is rejected
- **WHEN** the operator runs `c3po scout vibetree -p 100`
- **THEN** the command SHALL exit 2

#### Scenario: A bad mode is rejected
- **WHEN** the operator runs `c3po scout vibetree -m nope`
- **THEN** the command SHALL exit 2

#### Scenario: A missing path fails
- **WHEN** the operator runs `c3po scout vibetree` on a path that does not exist
- **THEN** the command SHALL exit 1

### Requirement: Line count is the loc size
In `loc` mode a module SHALL be each text file and each directory that contains a counted file. A file's size SHALL be its number of lines, counting a final line that has no newline. A directory's size SHALL be the sum of the file sizes under it. A file with a NUL byte, an empty file, and a non-regular file SHALL NOT be a module. The scout SHALL NOT select files by language.

In a git work tree the file list SHALL be `git ls-files` for tracked and untracked files, honoring gitignore. Otherwise the scout SHALL walk the directory and SHALL skip VCS and dependency directory names, including `.git` and `node_modules`. The same file list SHALL be used in `cyclo` mode.

#### Scenario: A directory sums its files
- **GIVEN** `a.go` of 10 lines and `b/c.go` of 100 lines and `b/d.go` of 10 lines
- **WHEN** the operator runs `c3po scout vibetree -p 50`
- **THEN** `b` SHALL show 110
- **AND** the root SHALL show 120

#### Scenario: Text of any name counts
- **GIVEN** `notes.txt` of 2 lines and `src/main.go` of 3 lines
- **WHEN** the operator runs `c3po scout vibetree`
- **THEN** both files SHALL be modules

#### Scenario: A binary and a dependency directory are omitted
- **GIVEN** a file with a NUL byte and `node_modules/pkg/index.js`
- **WHEN** the operator runs `c3po scout vibetree` outside a git work tree
- **THEN** neither path SHALL be a module

#### Scenario: Gitignore drops a file
- **GIVEN** a git work tree whose `.gitignore` lists `secret/`
- **WHEN** the operator runs `c3po scout vibetree`
- **THEN** a file under `secret/` SHALL NOT be a module

### Requirement: Cyclomatic mode sums functions per file
In `cyclo` mode a file's size SHALL be the sum of the cyclomatic complexity of its functions. A directory's size SHALL be the sum of those file sizes. The module SHALL stay the file or the directory. Go, Python, Rust, TypeScript, and JavaScript SHALL be scored. JavaScript SHALL use the TypeScript reader. A file in another language, or a scored file with no function, SHALL NOT be a module.

#### Scenario: Two languages share one directory
- **GIVEN** `lib/a.go` and `lib/b.py`, each with functions, and `lib/notes.txt`
- **WHEN** the operator runs `c3po scout vibetree -m cyclo -p 1`
- **THEN** `a.go` and `b.py` SHALL be modules
- **AND** `notes.txt` SHALL NOT be a module
- **AND** `lib` SHALL show the sum of the two files

#### Scenario: An unscored tree has no modules
- **GIVEN** a directory whose only file is `notes.txt`
- **WHEN** the operator runs `c3po scout vibetree -m cyclo`
- **THEN** the text SHALL contain `No modules.`

### Requirement: The percentile cuts the tree
The report SHALL begin with `mode loc` or `mode cyclo`, then print `modules N`, then the chosen percentile and its cutoff as `pN <cutoff>`, using the nearest-rank percentile. It SHALL NOT print any other percentile. It SHALL then print a tree of each module whose size is at or above that cutoff. A child below the cutoff SHALL be omitted. Children SHALL be ordered by size descending, then by name. The root SHALL be named `.`. An empty tree SHALL print `No modules.` and SHALL NOT print a tree. When the chosen percentile is the largest module and no child of the root meets the cutoff, the report SHALL print one uncolored sentence naming that rank, stating that the root is that size because a directory sums its files, and telling the operator to try a lower `-p`. There SHALL be no grade, ratio, or score.

#### Scenario: The light files drop out
- **GIVEN** `a.go` of 10 lines, `b/c.go` of 100 lines, and `b/d.go` of 10 lines
- **WHEN** the operator runs `c3po scout vibetree -p 50`
- **THEN** the text SHALL show `c.go` under `b` under `.`
- **AND** the text SHALL NOT show `a.go` or `d.go`

#### Scenario: An empty directory prints no tree
- **GIVEN** a directory with no text files
- **WHEN** the operator runs `c3po scout vibetree`
- **THEN** the text SHALL contain `No modules.`

#### Scenario: Only the chosen percentile is printed
- **GIVEN** files of different sizes
- **WHEN** the operator runs `c3po scout vibetree -p 95`
- **THEN** the text SHALL contain `p95`
- **AND** the text SHALL NOT contain `p50`, `p90`, or `p99`

#### Scenario: A small sample explains p95
- **GIVEN** a directory of few text files
- **WHEN** the operator runs `c3po scout vibetree`
- **THEN** the text SHALL say that p95 is the largest module
- **AND** the text SHALL tell the operator to try a lower `-p`
- **AND** `--json` SHALL put that sentence in `note`

### Requirement: JSON prints the same cutoff
`--json` SHALL print one JSON object and SHALL NOT print color or the text report. The object SHALL include `mode`, `modules`, `p`, `cutoff`, and `tree`. When the text report prints the small-sample sentence, `note` SHALL be that sentence. Otherwise `note` SHALL be omitted. `tree` SHALL contain each module the text tree would print, with `path`, `name`, `size`, and `children`. An empty tree SHALL set `tree` to null and SHALL NOT print `No modules.`

#### Scenario: JSON keeps the heavy file
- **GIVEN** `a.go` of 10 lines, `b/c.go` of 100 lines, and `b/d.go` of 10 lines
- **WHEN** the operator runs `c3po scout vibetree -p 50 --json`
- **THEN** the JSON SHALL name `c.go`
- **AND** the JSON SHALL NOT name `a.go` or `d.go`

### Requirement: A directory path roots the tree
The path argument SHALL be a directory. The modules SHALL be the files under that directory and the directories that contain them. A file outside that directory SHALL NOT be a module.

#### Scenario: The path comes before the percentile
- **GIVEN** `focus/big/b.txt` and `other/c.txt`
- **WHEN** the operator runs `c3po scout vibetree focus -p 1`
- **THEN** the text SHALL show `big`
- **AND** the text SHALL NOT show `other`

### Requirement: Tree lines scale in yellow
When color is on, each tree line SHALL be yellow, `\x1b[38;2;N;N;0m`, with N from the same curve `l` uses for text files. In `loc` mode N SHALL be 70 at or under 200 lines when a larger module is shown, 180 when every shown module is at or under 200 lines, and 255 at the largest line count. In `cyclo` mode the same curve SHALL use 1 in place of 200, so a size of 1 stays dim when a larger module is shown. A larger size SHALL be at least as bright as a smaller one. The mode and percentile lines SHALL stay uncolored. Color SHALL follow stdout: on for a terminal in `auto`, on for `--color=always`, and off for `--no-color`, `--color=never`, and a non-terminal under `auto`.

#### Scenario: The heavier directory is brighter
- **GIVEN** a chosen directory with `small` of 250 lines and `big` of 5000 lines
- **WHEN** the operator runs `c3po scout vibetree <dir> -p 1 --color=always`
- **THEN** the yellow of `big` SHALL be brighter than the yellow of `small`
- **AND** `--no-color` SHALL print no escape sequence

#### Scenario: A straight function stays dim
- **GIVEN** one function with no branch and one function with a branch
- **WHEN** the operator runs `c3po scout vibetree -m cyclo -p 1 --color=always`
- **THEN** the straight function SHALL be dimmer than the branching function
- **AND** the mode line SHALL NOT be colored
