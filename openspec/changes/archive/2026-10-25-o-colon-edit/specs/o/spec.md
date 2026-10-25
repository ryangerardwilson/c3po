## MODIFIED Requirements

### Requirement: Bang runs a shell command
`!` SHALL open the colon command line on the last row with `!` already typed and the cursor after it. The line SHALL show `:` immediately followed by `!`, with no space between them, and SHALL NOT show a separate `!` prompt or a hint. Text typed after that SHALL sit against the bang. Enter SHALL run the text after `!` in the shell with the listed directory as its working directory, and SHALL hand the terminal to that command. The command SHALL NOT be limited to `mv`, `rm`, `cp`, `touch`, or `mkdir`. An empty command SHALL NOT run. Esc SHALL cancel the command line without running a command. After the command exits, the listing SHALL reload and the status line SHALL be the last line again. `!` typed into the filter SHALL stay text. Editing keys SHALL be the same as on the colon line. The shortcut list and `c3po tui o help` SHALL name `!`.

#### Scenario: A shell command runs in the listed directory
- **WHEN** the operator presses `!`, types `touch made.txt`, and presses enter
- **THEN** `made.txt` SHALL be created in the listed directory

#### Scenario: A refused file command is allowed in the shell
- **WHEN** the operator presses `!`, types `rm -rf notes`, and presses enter
- **THEN** the directory notes SHALL be gone

#### Scenario: An empty shell line does not run
- **WHEN** the operator presses `!` and then enter
- **THEN** no command SHALL run

#### Scenario: Bang sits against the colon
- **WHEN** the operator presses `!`
- **THEN** the last line SHALL show `:` immediately followed by `!` and the cursor
- **AND** the last line SHALL NOT show a hint

#### Scenario: The shell line is the last row
- **WHEN** the operator presses `!` and types `touch made.txt`
- **THEN** the last line SHALL contain `:!touch made.txt`
- **AND** the line under the title SHALL NOT contain `touch made.txt`


## REMOVED Requirements

### Requirement: Colon runs mv, rm, cp, touch, or mkdir
**Reason**: The colon line sits against the typed text, a leading bang runs a shell, and the line edits at a cursor.
**Migration**: Type mv, rm, cp, touch, or mkdir after `:`. Type `:!` or press `!` for a shell command.

## ADDED Requirements

### Requirement: Colon runs mv, rm, cp, touch, mkdir, or :!
`:` SHALL open a command line on the last row. The command line SHALL show `:` immediately followed by the text being typed, with no space after `:`, and SHALL NOT show a hint or that text on the prompt line under the title. Enter SHALL run that line in the listed directory when the command is `mv`, `rm`, `cp`, `touch`, or `mkdir`. `rm` MAY take `-rf`. `cp` and `mv` SHALL require two paths. Any other command or flag SHALL be refused and SHALL NOT run, unless the trimmed line starts with `!`. A trimmed line that starts with `!` SHALL run the rest in the shell with the listed directory as its working directory, and SHALL hand the terminal to that command. Space after the bang SHALL be ignored. An empty shell command SHALL NOT run. That shell command SHALL NOT be limited to `mv`, `rm`, `cp`, `touch`, or `mkdir`. Esc SHALL cancel the command line without running a command. After a command runs, the listing SHALL reload and the status line SHALL be the last line again. `:` typed into the filter SHALL stay text. The shortcut list and `c3po tui o help` SHALL name these commands and `:!`.

While the command line is open, the cursor SHALL move inside the typed text. Home and Ctrl-a SHALL move to the start. End and Ctrl-e SHALL move to the end. Left and Ctrl-b SHALL move back one character. Right and Ctrl-f SHALL move forward one character. Backspace SHALL delete the character before the cursor. Delete and Ctrl-d SHALL delete the character at the cursor. Ctrl-d at the end of the line, including an empty line, SHALL do nothing and SHALL NOT quit. Ctrl-k SHALL delete from the cursor to the end. Ctrl-u SHALL delete from the start to the cursor. A word is a run of non-whitespace characters. Ctrl-w and Alt-Backspace SHALL delete the whitespace immediately before the cursor and then the word before that. Alt-b SHALL move to the start of the word before the cursor. Alt-f SHALL move to the end of the word after the cursor. Alt-d SHALL delete through that end. An Alt chord other than these SHALL NOT insert a character. Ctrl-w while the command line is closed SHALL NOT delete a file.

#### Scenario: mkdir creates a directory
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses enter
- **THEN** the listed directory SHALL contain a directory named notes
- **AND** the listing SHALL show notes

#### Scenario: cp copies a file
- **WHEN** the operator submits `cp a.txt b.txt` on the `:` command line
- **THEN** the listed directory SHALL contain b.txt
- **AND** a.txt SHALL remain

#### Scenario: mv renames a file
- **WHEN** the operator submits `mv b.txt c.txt` on the `:` command line
- **THEN** the listed directory SHALL contain c.txt
- **AND** b.txt SHALL be gone

#### Scenario: rm -rf removes a directory
- **WHEN** the operator submits `rm -rf notes` on the `:` command line
- **THEN** the directory notes SHALL be gone

#### Scenario: Another command is refused
- **WHEN** the operator submits `ls` on the `:` command line
- **THEN** no command SHALL run

#### Scenario: Another flag is refused
- **WHEN** the operator submits `rm -r notes` on the `:` command line
- **THEN** no command SHALL run
- **AND** the directory notes SHALL remain

#### Scenario: A colon bang runs a shell command
- **WHEN** the operator presses `:`, types `!touch made.txt`, and presses enter
- **THEN** `made.txt` SHALL be created in the listed directory

#### Scenario: An empty colon bang does not run
- **WHEN** the operator presses `:`, types `!`, and presses enter
- **THEN** no command SHALL run

#### Scenario: The command line is the last row
- **WHEN** the operator presses `:` and types `mkdir notes`
- **THEN** the last line SHALL contain `:mkdir notes`
- **AND** the last line SHALL NOT contain `: mkdir`
- **AND** the line under the title SHALL NOT contain `mkdir notes`

#### Scenario: An empty colon line has no hint
- **WHEN** the operator presses `:`
- **THEN** the last line SHALL show `:` immediately followed by the cursor
- **AND** the last line SHALL NOT name touch, mkdir, or rm

#### Scenario: Colon in the filter is text
- **WHEN** the operator is typing a filter and presses `:`
- **THEN** the command line SHALL stay closed
- **AND** the filter text SHALL contain `:`

#### Scenario: Ctrl-a moves to the start of the command
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses Ctrl-a
- **THEN** the cursor SHALL be before `mkdir notes`

#### Scenario: Alt-b moves to the previous word
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses Alt-b
- **THEN** the cursor SHALL be before `notes`
- **AND** the command line SHALL still contain `mkdir`

#### Scenario: Ctrl-w deletes the previous word
- **WHEN** the operator presses `:`, types `mkdir notes`, and presses Ctrl-w
- **THEN** the command line SHALL show `mkdir`
- **AND** the command line SHALL NOT show `notes`
- **AND** no file SHALL be removed

#### Scenario: Ctrl-w while browsing does not delete
- **WHEN** the operator presses Ctrl-w while the command line is closed
- **THEN** no file SHALL be removed

#### Scenario: An unbound Alt chord does not insert
- **WHEN** the operator presses `:` and then Alt-x
- **THEN** the command text SHALL be empty
