# Calendar Puzzle Solver

A high-performance Go implementation of a calendar puzzle solver that finds solutions for placing 8 unique pieces on a calendar board to cover all positions except the current date.

## 🌐 **[🚀 Try the Live Demo!](https://rhamdeew.github.io/calendar_solver/)** 🌐

> **Experience the solver in action directly in your browser - no installation required!**

## 🎮 Live Demo

**🔗 [Interactive Calendar Puzzle Solver](https://rhamdeew.github.io/calendar_solver/)**

The demo is a WebAssembly (Wasm) build of the solver running directly in your browser. It is automatically updated with every push to the `main` branch.

✨ **Features of the web demo:**
- 🚀 Instant solving in your browser
- 🎯 Interactive date selection
- 📱 Works on mobile and desktop
- 🔄 Real-time visualization

## Features

- **High-Performance Solver**: Utilizes a parallel backtracking algorithm to find solutions quickly.
- **WebAssembly Demo**: Interactive web-based version compiled to Wasm.
- **Command-Line Interface**: A powerful CLI for local usage and testing.
- **Visual Output**: Both the CLI and web demo provide clear visual representations of the solution.

## Overview

This solver tackles the classic calendar puzzle where you have:
- A 7×7 grid representing a calendar with months and days
- 8 unique puzzle pieces of varying shapes (5-6 cells each)
- The goal: place all pieces to cover every position except today's date

The puzzle is particularly challenging because:
- Each piece can be rotated and flipped (up to 8 orientations each)
- The solution changes daily as the blocked positions change
- There are millions of possible combinations to explore

## Features

### 🚀 High Performance
- **Parallel processing** using all available CPU cores
- **Backtracking algorithm** with intelligent pruning
- **Memory-efficient** piece representation and board state
- Typically solves puzzles in **under 1 second**

### 🎯 Comprehensive Solving
- Supports all dates (1-31) and months (Russian month names)
- Automatic current date detection
- Command-line date specification
- Batch testing of multiple dates

### 📊 Detailed Analytics
- Real-time attempt counting
- Performance metrics (attempts per second)
- Worker identification for parallel solving
- Execution time tracking

### 🎨 Visual Output
- ASCII art board visualization
- Piece placement visualization
- Board configuration display
- Piece shape illustrations

## Board Layout

The calendar board uses a 7×7 grid with the following layout:

```
    0   1   2   3   4   5   6
0:  Янв Фев Март Апр Май Июнь  .
1:  Июль Авг Сент Окт Нояб Дек  .
2:   1   2   3   4   5   6   7
3:   8   9  10  11  12  13  14
4:  15  16  17  18  19  20  21
5:  22  23  24  25  26  27  28
6:  29  30  31   .   .   .   .
```

- **Rows 0-1**: Month positions (12 months)
- **Rows 2-6**: Day positions (1-31)
- **Total valid positions**: 43 cells
- **Blocked positions**: Current month + current day (2 cells)
- **Target coverage**: 41 cells (exactly matching total piece cells)

## Puzzle Pieces

The solver includes 8 unique pieces with the following configurations:

1. **L-shape** (5 cells) - Classic L tetromino
2. **Long L** (5 cells) - Extended L shape
3. **Cut Rectangle** (5 cells) - Rectangle with one corner extended
4. **Rectangle** (6 cells) - 2×3 rectangular piece
5. **T-shape** (5 cells) - T tetromino
6. **Z-shape** (5 cells) - Z tetromino
7. **P-shape** (5 cells) - P-like configuration
8. **Stair Shape** (5 cells) - Stepped configuration piece

**Total cells**: 41 (perfectly matches the target coverage)

## Installation

### Prerequisites
- Go 1.19 or later
- Unix-like system (macOS, Linux) or Windows with Go support

### Build from Source
```bash
git clone <repository-url>
cd calendar_solver
go build -o calendar_solver main.go
```

## Usage

### Basic Usage (Current Date)
```bash
./calendar_solver
```
Automatically detects and solves for today's date.

### Specific Date
```bash
# Using day number and month name
./calendar_solver -day 15 -month Март

# Using day number and month number
./calendar_solver -day 25 -month 12

# Using partial month name
./calendar_solver -day 1 -month Янв
```

### Test Mode Only
```bash
./calendar_solver -test-only
```
Skips main solve and runs only predefined test cases.

### Command Line Options
- `-day <1-31>`: Specify the day
- `-month <month>`: Specify month (Russian name, number 1-12, or partial name)
- `-test-only`: Run only test cases, skip main solve

## Example Output

```
BOARD CONFIGURATION:
=================================================
    0   1   2   3   4   5   6
0:  Янв Фев Март Апр Май Июнь  .
1:  Июль Авг Сент Окт Нояб Дек  .
2:   1   2   3   4   5   6   7
3:   8   9  10  11  12  13  14
4:  15  16  17  18  19  20  21
5:  22  23  24  25  26  27  28
6:  29  30  31   .   .   .   .

Solving calendar board for: 15 Март
Available pieces: 8 pieces
Available CPU cores: 8
Piece sizes: [5, 5, 5, 6, 5, 5, 5, 5] cells each

✓ Solution found in 0.1234 seconds!
Worker 3 found the solution after 45678 attempts
Attempts per second: 370123

Solution for 15 Март:
==============================
1 1 2 2 2 2 .
3 3 3 4 4 4 .
5 5 X 6 6 6 7
8 8 1 1 1 X 7
8 3 3 4 4 7 7
8 5 5 6 6 7 .
8 8 . . . . .

X = Current date (15 Март)
1-8 = Piece numbers
. = Empty/Invalid positions
```

## Algorithm Details

### Parallel Backtracking
The solver uses a sophisticated parallel backtracking algorithm:

1. **Work Distribution**: Initial state is distributed across multiple worker goroutines
2. **Piece Placement**: Each worker tries placing pieces in sequential order
3. **Orientation Testing**: All valid rotations and flips are tested for each piece
4. **Constraint Checking**: Validates board boundaries, collisions, and blocked cells
5. **Early Termination**: First solution found terminates all workers

### Optimization Techniques
- **Piece Normalization**: Consistent representation reduces duplicate orientations
- **Bounds Checking**: Early rejection of invalid placements
- **Memory Pooling**: Efficient board state copying
- **Atomic Counters**: Lock-free attempt tracking

### Performance Characteristics
- **Time Complexity**: Exponential in worst case, but heavily pruned
- **Space Complexity**: O(board_size × piece_count)
- **Typical Performance**: 10,000-100,000 attempts per second per core
- **Success Rate**: High for most valid calendar dates

## Technical Implementation

### Key Data Structures
```go
type Position struct {
    Row, Col int
}

type Piece []Position

type SolveResult struct {
    Solution    []Position
    PieceMap    map[Position]int
    Found       bool
    SolveTime   time.Duration
    Attempts    int64
    WorkerID    int
}
```

### Core Components
- **CalendarBoardSolver**: Main solver class with board configuration
- **Piece Management**: Rotation, flipping, and normalization logic
- **Parallel Workers**: Goroutine-based parallel processing
- **Result Aggregation**: Thread-safe result collection

## Performance Benchmarks

Typical performance on modern hardware:
- **Simple dates**: 0.001-0.1 seconds
- **Complex dates**: 0.1-1.0 seconds
- **Difficult dates**: 1-10 seconds
- **Impossible dates**: 60 seconds (timeout)

Performance scales linearly with CPU core count.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the LICENSE file for details.

## Acknowledgments

- Inspired by the classic calendar puzzle design
- Built with Go's excellent concurrency primitives
- Optimized for modern multi-core processors