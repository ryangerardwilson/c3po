## ADDED Requirements

### Requirement: Only a safe external port is signalled
A safe external port SHALL be an integer from 8000 through 8999 inclusive. `c3po deport <port>` SHALL signal every process with a local TCP or UDP socket on that port, SIGTERM first and SIGKILL when the port is still taken, and SHALL print `killed <pid> <name>` for each process it signals, then `<port> is free`. A port outside that range SHALL be refused with no signal, SHALL exit 2, and SHALL say `<port> is not a safe external port`. The command SHALL exit 0 when the port is free. The command SHALL exit 1 when a safe port stays occupied or its process cannot be seen or signalled.

#### Scenario: An internal port is refused
- **WHEN** the operator runs `c3po deport 22`
- **THEN** the command SHALL exit 2
- **AND** no process SHALL be signalled
- **AND** the text SHALL say `22 is not a safe external port`

#### Scenario: The edges of the band are safe
- **WHEN** the operator names port 8000 or port 8999
- **THEN** the command SHALL treat that port as safe
- **AND** port 7999 and port 9000 SHALL be refused with no signal

#### Scenario: A safe port is freed
- **WHEN** a process holds local port 8082
- **AND** the operator runs `c3po deport 8082`
- **THEN** the command SHALL exit 0
- **AND** the process SHALL receive SIGTERM, then SIGKILL if it is still alive
- **AND** the text SHALL say the port is free

### Requirement: List shows occupied external ports
`c3po deport list` SHALL print one row for each process holding a safe external port, as `<port> <pid> <name>`, sorted by port and then pid. A port in that band with no visible process SHALL be printed as the port number alone. An internal port SHALL NOT appear. When no safe external port is occupied, the command SHALL print `no occupied external ports` and SHALL exit 0.

#### Scenario: Internal ports are omitted
- **WHEN** ports 22 and 8080 are occupied
- **AND** the operator runs `c3po deport list`
- **THEN** the text SHALL contain the 8080 row
- **AND** the text SHALL NOT contain a row for port 22

#### Scenario: An empty band prints one sentence
- **WHEN** no port from 8000 through 8999 is occupied
- **AND** the operator runs `c3po deport list`
- **THEN** the command SHALL exit 0
- **AND** the text SHALL be `no occupied external ports`

### Requirement: The screen lists those ports and frees one
`c3po tui deport` SHALL be the screen. It SHALL list the same occupied safe external ports. `j` SHALL move the cursor down and `k` SHALL move it up. `dd` SHALL free the selected port by the same signal sequence as `c3po deport <port>`. The first `d` SHALL arm the chord and SHALL show `d`. Esc SHALL cancel the arm. Any other key SHALL clear the arm and SHALL run as a normal motion. When the port becomes free, its row SHALL leave the list. `q` SHALL quit. Extra arguments SHALL exit 2 and SHALL print the screen help. `c3po tui deport help` SHALL name the range 8000 through 8999, SHALL say that `j` and `k` move, and SHALL say that `dd` frees the selected port.

#### Scenario: j and k move
- **WHEN** the screen lists port 8080 and port 8082
- **AND** the cursor is on 8080
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on 8082
- **AND** `k` SHALL move it back to 8080

#### Scenario: dd frees the selected port
- **WHEN** the cursor is on port 8082
- **AND** the operator presses `d` and then `d`
- **THEN** the holders of port 8082 SHALL be signalled
- **AND** port 8080 SHALL NOT be signalled
- **AND** the 8082 row SHALL leave the list when the port is free

#### Scenario: A single d does not free
- **WHEN** the operator presses `d` and then `j`
- **THEN** no process SHALL be signalled
- **AND** the cursor SHALL move down
