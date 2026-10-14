## Why

`map` is a walk, `affected` is a path prefix. Agents need typed connections:
who imports this, what is load-bearing, and which dirty files are one component.

## What Changes

- Build a heterogeneous graph: app/test/desktop packages, Go package dirs,
  Go files; edges `contains`, `imports`, `in_package`
- `rgw-ast graph`, `neighbors`, `cut`
- `affected` unions prefix mapping with reverse-import reachability
- `map` ranks files by import degree
- `neighbors --json` includes a 1-ring hash

## Capabilities

### Modified Capabilities
- `cli-surface`: graph, neighbors, cut verbs; affected reachability
- `code-intelligence`: ranked map, import graph
