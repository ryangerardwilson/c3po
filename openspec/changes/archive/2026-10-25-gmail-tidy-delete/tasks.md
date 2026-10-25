## 1. Spec

- [x] 1.1 Delete configured spam from tidy and list frequent unread senders from tidy audit
- [x] 1.2 Validate the change with `openspec validate gmail-tidy-delete --strict --no-interactive`

## 2. Command

- [x] 2.1 Trash only the configured spam list, with no hardcoded sender
- [x] 2.2 Add `c3po gmail tidy audit` and name both commands in help

## 3. Check

- [x] 3.1 Run `GOWORK=off go test ./gmail/...`
