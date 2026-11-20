## MODIFIED Requirements

### Requirement: The screen lists those ports and frees one
`r2d2 deport` SHALL be the screen. It SHALL list the same occupied safe external ports. `j` SHALL move the cursor down and `k` SHALL move it up, and both SHALL wrap. On the last port, `j` SHALL move to the first port. On the first port, `k` SHALL move to the last port. An empty list SHALL leave the cursor in place. `g` and `G` SHALL still move to the first and last port. `dd` SHALL free the selected port by the same signal sequence as `c3po deport <port>`. The first `d` SHALL arm the chord and SHALL show `d`. Esc SHALL cancel the arm. Any other key SHALL clear the arm and SHALL run as a normal motion. When the port becomes free, its row SHALL leave the list. `q` SHALL quit. Extra arguments SHALL exit 2 and SHALL print the screen help. `r2d2 deport help` SHALL name the range 8000 through 8999, SHALL say that `j` and `k` move, and SHALL say that `dd` frees the selected port.

#### Scenario: j and k move
- **WHEN** the screen lists port 8080 and port 8082
- **AND** the cursor is on 8080
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on 8082
- **AND** `k` SHALL move it back to 8080

#### Scenario: j and k wrap
- **WHEN** the screen lists port 8080 and port 8082
- **AND** the cursor is on 8082
- **AND** the operator presses `j`
- **THEN** the cursor SHALL be on 8080
- **AND** `k` SHALL move it to 8082

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
