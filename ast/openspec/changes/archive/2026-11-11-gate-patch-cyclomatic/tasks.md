## 1. Contract
- [x] 1.1 Write proposal and spec deltas
- [x] 1.2 Validate strictly

## 2. Counter
- [x] 2.1 Shared cyclomatic counter and declaration name in `internal/gocyclo`
- [x] 2.2 cyclotree `scoreGo` takes `Cyclomatic` from the counter
- [x] 2.3 Parity test between cyclotree `ParseGo` and the counter

## 3. Gate
- [x] 3.1 `max_cc` config key with an in-memory default of 30
- [x] 3.2 Check old and new bytes before the plain patch write
- [x] 3.3 cc lines, unparsed line, and refused lines with exit 1 and no Error line
- [x] 3.4 CLI tests for rise, refusal, already-high rise, new high, removal, non-Go, unparsed, hash mismatch, and `--symbol`

## 4. Docs
- [x] 4.1 `c3po ast help patch`, README, and the agents block
