# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build CLI binary
make build_cli

# Run CLI with a specific date
make run_cli ARGS="--day 15 --month 3"
# or directly
./calendar_solver_cli -day 15 -month "March"  # supports numbers and Russian month names

# Run tests
make test_cli
# or
go test -v ./cli

# Run a single test
go test -v ./cli -run TestGetMonthName

# Build and run web app (WASM + HTTP server)
make web          # build + serve on http://localhost:8080
make build_wasm   # only build web/main.wasm

# Clean
make clean
```

## Architecture

This is a calendar puzzle solver: given a date (month + day), find how to place 8 puzzle pieces on a 7×7 board so all cells are covered except the current date cells.

**Packages:**
- `solver/` — Core constraint solver. `CalendarBoardSolver` holds the board layout (months in rows 0-1, days in rows 2-6, 43 valid positions) and 8 piece definitions. `SolveParallel()` spawns one goroutine per CPU core, each running independent backtracking. First solution terminates all workers via `doneChan`.
- `cli/` — CLI entry point. Parses `-day`/`-month` flags (month accepts integers, English names, Russian names with partial match). Runs the solver and prints ASCII board visualization with piece numbers and metrics.
- `web/` — Two build targets: `wasm.go` (compiled with `GOOS=js GOARCH=wasm`) exposes `solveCalendar(day, monthIndex)` to JavaScript; `server.go` is a plain HTTP file server for local development.

**Solver algorithm:** Parallel backtracking — recursively try placing unused pieces at each board position across all 8 orientations (4 rotations × 2 for horizontal flip), normalized and deduplicated. `SolveResult` returns the piece-to-position map, solve time, and attempt count.

**WASM build constraint:** `web/wasm.go` uses `//go:build js && wasm` and `web/server.go` uses `//go:build !js`. The Makefile sets `GOOS=js GOARCH=wasm` for the WASM target.
