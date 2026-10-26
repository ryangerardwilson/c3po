# Design

Sessions use sqlx records with app `tui`. Kind `session` keeps the place. Kind `active` names the live process. A screen opens a session only for a terminal or an explicit `--resume` id, so tests that pass a buffer do not write the operator database.
