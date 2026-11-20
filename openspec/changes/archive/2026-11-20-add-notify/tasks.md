## 1. Command

- [x] 1.1 Add `c3po notify` with the labeled form, the positional form, and `--urgency`/`-u`
- [x] 1.2 Deliver through omarchy-shell, fall back to the quickshell bar, and name the omarchy-shell error on failure
- [x] 1.3 Copy the `ryan.notify` plugin to `notify/plugin/` and install it from `install.sh`; remove `~/.local/bin/notify`
- [x] 1.4 List `notify` in help, the usage catalog, the app docs, and the scout spec
- [x] 1.5 Make the gmail and gdrive timer units call `c3po notify` and drop the inline shell function and `notify-send`
- [x] 1.6 Show `c3po notify heading ...` in the looptab jobs template

## 2. Check

- [x] 2.1 Port the notify unit tests and test refusal, delivery fallback, the omarchy-shell error, help, and version
- [x] 2.2 Test that the gmail and gdrive units call `c3po notify` and no longer call `notify-send`
- [x] 2.3 Test that `c3po` and `c3po help` list notify and `c3po help -f notify` matches `c3po notify help`
- [x] 2.4 Validate this change with `openspec validate add-notify --strict --no-interactive` and archive it
