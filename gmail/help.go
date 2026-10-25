package gmail

// Help is the full help for c3po gmail.
const Help = `c3po gmail is the Gmail desk.

global actions:
  c3po gmail help
    show this help
  c3po gmail version
    print the installed version
  c3po gmail upgrade
    upgrade c3po instead of this command

features:
  authorize a Google account and save or refresh its preset
  # auth <client_secret_path>
  c3po gmail auth ~/Documents/credentials/client_secret.json

  edit account config, including signature_file paths used for automatic send/reply signatures
  # config
  c3po gmail config

  list configured accounts and run a redacted setup check for agents
  # accounts list | setup check
  c3po gmail accounts list
  c3po gmail setup check

  send a new email, with optional editor mode, cc, bcc, and attachments
  # <preset> send to <email|alias> subject <subject> body <body> [cc <emails>] [bcc <emails>] [attach <path> ...]
  c3po gmail 1 send in editor
  c3po gmail 1 send to person@example.com subject "Hello" body "Body"
  c3po gmail 1 send to boss subject "Notes" body "Draft is attached" attach ~/notes.txt
  c3po gmail 1 preview send to boss subject "Notes" body "Draft is attached" attach ~/notes.txt

  list and search messages using explicit filters
  # <preset> list [unread|read|sent|starred|external] [from <sender>] [containing <text>] [since <window>] [limit <count>] [with attachments] [open] [output json]
  c3po gmail 1 list unread from geeta since 2w limit 10
  c3po gmail 1 list sent containing proposal since 14d limit 10
  c3po gmail 1 list with attachments from geeta limit 10
  c3po gmail 1 list unread from geeta limit 10 output json

  open messages and reply to messages or threads
  # <preset> inspect message <id> | inspect thread <id> | open message <id> | open thread <id>
  c3po gmail 1 inspect message 19caef2cd6494116
  c3po gmail 1 open message 19caef2cd6494116
  # <preset> reply to <message_id> [all] [in editor|body <body>]
  c3po gmail 1 preview reply to 19caef2cd6494116 body "Thanks for the update."
  c3po gmail 1 reply to 19caef2cd6494116 body "Thanks for the update."
  c3po gmail 1 reply to thread 19ca756c06a7ebcd all body "Thanks everyone."

  trash one message, preset unread, or unread across all presets
  # delete unread | <preset> delete message <id> | <preset> delete unread
  c3po gmail delete unread
  c3po gmail 1 delete message 19caef2cd6494116
  c3po gmail 1 delete unread

  clean spam, inspect spam candidates, and control the hourly timer
  # spam clean | timer install|disable|status | <preset> spam inspect|clean|add|allow
  c3po gmail spam clean
  c3po gmail timer install
  c3po gmail 1 spam inspect
  c3po gmail 1 spam add unread
  c3po gmail 1 spam allow trusted@example.com

  delete configured spam, and list frequent unread senders
  # tidy | tidy audit
  c3po gmail tidy
  c3po gmail tidy audit

  manage saved contacts for a preset
  # <preset> contacts list|add|delete|edit
  c3po gmail 1 contacts list
  c3po gmail 1 contacts add boss boss@example.com
  c3po gmail 1 contacts edit

agent-safe workflow:
  list accounts first, inspect before open when read-state matters, preview
  before send or reply, and use output json when another agent will parse rows
`
