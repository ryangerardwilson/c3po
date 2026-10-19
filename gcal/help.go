package gcal

// Help is the full gcal help text. c3po help -f gcal prints the same.
const Help = `c3po gcal

global actions:
  c3po gcal help
    show this help
  c3po gcal version
    print the installed c3po version

features:
  edit Google Calendar presets and defaults
  # config
  c3po gcal config

  list configured accounts and run a redacted setup check for agents
  # accounts list | setup check
  c3po gcal accounts list
  c3po gcal setup check

  authorize a Google account and save or refresh its preset
  # auth <client_secret_path>
  c3po gcal auth ~/Documents/credentials/client_secret.json

  list upcoming non-recurring events
  # <preset> list limit <n> [output json]
  c3po gcal 1 list limit 5
  c3po gcal 1 list limit 5 output json

  create a calendar event with invitees and a Google Meet link
  # <preset> create event <title> from <start> to <end> invite <csv>
  c3po gcal 1 preview create event "Interview" from "2026-03-10 14:00:00" to "2026-03-10 15:00:00" invite a@example.com,b@example.com
  c3po gcal 1 create event "Interview" from "2026-03-10 14:00:00" to "2026-03-10 15:00:00" invite a@example.com,b@example.com

  delete, reschedule, or export an event transcript artifact
  # <preset> delete event <id> | reschedule event <id> from <start> to <end> | export event <id>
  c3po gcal 1 delete event abc123
  c3po gcal 1 preview reschedule event abc123 from "2026-03-11 10:00:00" to "2026-03-11 11:00:00"
  c3po gcal 1 reschedule event abc123 from "2026-03-11 10:00:00" to "2026-03-11 11:00:00"
  c3po gcal 1 export event abc123

agent-safe workflow:
  list accounts first, list events before mutating an unknown event id, preview
  create or reschedule before writing, and use output json when another agent
  will parse rows

paths:
  config  $XDG_CONFIG_HOME/c3po/gcal/config.json (default ~/.config/c3po/gcal/config.json)
  tokens  $XDG_DATA_HOME/c3po/gcal/tokens/<email>.json (default ~/.local/share/c3po/gcal/tokens)
  times   YYYY-MM-DD HH:MM:SS in defaults.timezone

exit: 0 ok, 1 unexpected failure, 2 usage, config, or Google API error
`
