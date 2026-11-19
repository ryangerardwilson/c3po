## ADDED Requirements

### Requirement: Tab completes a :! word
While the command line is open and its trimmed text starts with `!`, Tab SHALL complete the shell word at the cursor and SHALL NOT run a command. The listed directory SHALL be the working directory for that completion. The first word SHALL complete an executable file on `PATH`, unless the word contains `/` or starts with `~`, in which case it SHALL complete a path. A later word SHALL complete a path. A relative path SHALL resolve in the listed directory. An absolute path SHALL resolve there. A word that starts with `~/` SHALL resolve under the home directory and SHALL stay written with `~/`. A name that starts with `.` SHALL match only when the typed base starts with `.`. `.` and `..` SHALL NOT be matches.

One match that is a directory SHALL be inserted with a trailing `/` and SHALL NOT gain a space. One other match SHALL be inserted with one trailing space, unless a space already follows the word. Several matches SHALL insert their longest common prefix and SHALL NOT add `/` or a space. When that insertion does not change the word, the prompt line under the title SHALL show the match names and SHALL NOT show the command text. Any other key SHALL clear those names. Tab on a command line whose trimmed text does not start with `!` SHALL leave the line unchanged. A character that the shell would split or expand SHALL be escaped so the completed word stays one word. The shortcut list and `r2d2 o help` SHALL name Tab.

#### Scenario: A unique file completes with a space
- **GIVEN** the listed directory contains a file `notes.txt`
- **WHEN** the operator types `:!ls no` and presses Tab
- **THEN** the command line SHALL be `:!ls notes.txt `
- **AND** no command SHALL run

#### Scenario: A unique directory gains a slash
- **GIVEN** the listed directory contains a directory `notes`
- **WHEN** the operator types `:!ls no` and presses Tab
- **THEN** the command line SHALL be `:!ls notes/`

#### Scenario: Several matches fill the common prefix
- **GIVEN** the listed directory contains `notes.txt` and `notice.txt`
- **WHEN** the operator types `:!ls n` and presses Tab
- **THEN** the command line SHALL be `:!ls not`
- **AND** the prompt line SHALL NOT show those names

#### Scenario: An unchanged word lists the matches
- **GIVEN** the listed directory contains `notes.txt` and `notice.txt`
- **WHEN** the command line is `:!ls not` and the operator presses Tab
- **THEN** the command line SHALL stay `:!ls not`
- **AND** the prompt line SHALL show `notes.txt` and `notice.txt`
- **AND** the prompt line SHALL NOT show `:!`

#### Scenario: A dot name stays hidden
- **GIVEN** the listed directory contains `.secret` and `notes.txt`
- **WHEN** the operator types `:!ls n` and presses Tab
- **THEN** the command line SHALL be `:!ls notes.txt `

#### Scenario: The first word completes a command
- **GIVEN** `PATH` contains an executable named `touch`
- **WHEN** the operator types `:!tou` and presses Tab
- **THEN** the command line SHALL be `:!touch `

#### Scenario: A path command completes in the listed directory
- **GIVEN** the listed directory contains an executable `script`
- **WHEN** the operator types `:!./sc` and presses Tab
- **THEN** the command line SHALL be `:!./script `

#### Scenario: A space in a name is escaped
- **GIVEN** the listed directory contains a file `my notes.txt`
- **WHEN** the operator types `:!ls my` and presses Tab
- **THEN** the command line SHALL be `:!ls my\ notes.txt `

#### Scenario: Tab outside a shell line does nothing
- **WHEN** the operator types `:mkdir no` and presses Tab
- **THEN** the command line SHALL stay `:mkdir no`
