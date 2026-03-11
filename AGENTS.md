<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **x-hmsw** (1258 symbols, 3051 relationships, 104 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Project Overview

x-hmsw is a high-performance vector database implemented in pure Go, designed for large-scale vector similarity search.

### Key Features

- **Multiple Index Algorithms**: HNSW, IVF, Flat, ANN
- **Flexible Storage**: Badger, BBolt, Pebble, Memory, Mmap
- **Text Vectorization**: TF-IDF, BM25 (with variants), OpenAI Embeddings
- **Vector Compression**: PQ, SQ, Binary
- **High Performance**: SIMD acceleration, object pooling, concurrency control
- **Easy to Use**: QuickDB provides out-of-the-box simplified interface
- **Monitoring**: Built-in Prometheus metrics support

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
├─────────────────────────────────────────────────────────┤
│                      API Layer                          │
│                   (QuickDB 接口)                         │
├─────────────────────────────────────────────────────────┤
│                   Interface Layer                       │
│              (VectorDB, Index 接口)                      │
├─────────────────────────────────────────────────────────┤
│                 Embedding Layer                         │
│         (TF-IDF, BM25, OpenAI Embeddings)               │
├─────────────────────────────────────────────────────────┤
│                    Index Layer                          │
│         (HNSW, IVF, Flat, ANN 索引实现)                 │
├─────────────────────────────────────────────────────────┤
│                 Compression Layer                       │
│            (PQ, SQ, Binary 压缩)                        │
├─────────────────────────────────────────────────────────┤
│                   Storage Layer                         │
│      (Badger, BBolt, Pebble, Memory, Mmap)             │
├─────────────────────────────────────────────────────────┤
│                    Utils Layer                          │
│         (SIMD, Pool, Math, Concurrency)                 │
├─────────────────────────────────────────────────────────┤
│                  Infrastructure                         │
│              (Metrics, Logging, Serialization)          │
└─────────────────────────────────────────────────────────┘
```

### Directory Structure

- `api/`: QuickDB simplified API
- `embedding/`: Text vectorization (TF-IDF, BM25, OpenAI)
- `indexes/`: Index implementations (HNSW, IVF, Flat, ANN)
- `compression/`: Vector compression (PQ, SQ, Binary)
- `storage/`: Storage engines (Badger, BBolt, Pebble, Memory, Mmap)
- `interface/`: Core interfaces (VectorDB, Index, Storage, Logger)
- `types/`: Core data types
- `utils/`: Utility functions (SIMD, Pool, Math, Concurrency)
- `metrics/`: Prometheus metrics
- `serialization/`: Data serialization (Protobuf, MessagePack, Binary)
- `docs/`: Documentation
- `examples/`: Example code

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/x-hmsw/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/x-hmsw/context` | Codebase overview, check index freshness |
| `gitnexus://repo/x-hmsw/clusters` | All functional areas |
| `gitnexus://repo/x-hmsw/processes` | All execution flows |
| `gitnexus://repo/x-hmsw/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

- Re-index: `npx gitnexus analyze`
- Check freshness: `npx gitnexus status`
- Generate docs: `npx gitnexus wiki`

<!-- gitnexus:end -->
