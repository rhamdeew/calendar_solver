package solver

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Position is a (row, col) coordinate on the 7×7 board.
type Position struct {
	Row, Col int
}

// Piece is a slice of relative cell positions.
type Piece []Position

// Board is a 7×7 grid:
//
//	 0 = empty and available
//	-1 = blocked (invalid calendar cell or current date cell)
//	1–8 = placed piece number
type Board [7][7]int8

// CalendarBoardSolver holds the puzzle configuration.
type CalendarBoardSolver struct {
	Months         []string
	MonthPositions map[string]Position
	DayPositions   map[int]Position
	Pieces         []Piece
	orientations   [][]Piece // precomputed unique orientations per piece
}

// SolveResult contains the output of a solve attempt.
type SolveResult struct {
	Solution  []Position
	PieceMap  map[Position]int // position → piece number (1–8)
	Found     bool
	SolveTime time.Duration
	Attempts  int64
	WorkerID  int
}

// maxDaysInMonth[m] = maximum days in month m (1-indexed). February = 29 (leap-year inclusive).
var maxDaysInMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// NewCalendarBoardSolver constructs the solver with its fixed board layout and 8 pieces.
func NewCalendarBoardSolver() *CalendarBoardSolver {
	s := &CalendarBoardSolver{
		Months: []string{
			"Янв", "Фев", "Март", "Апр", "Май", "Июнь",
			"Июль", "Авг", "Сент", "Окт", "Нояб", "Дек",
		},
		MonthPositions: make(map[string]Position),
		DayPositions:   make(map[int]Position),
	}

	for i, month := range s.Months {
		if i < 6 {
			s.MonthPositions[month] = Position{0, i}
		} else {
			s.MonthPositions[month] = Position{1, i - 6}
		}
	}

	day := 1
	for row := 2; row < 7; row++ {
		for col := 0; col < 7 && day <= 31; col++ {
			s.DayPositions[day] = Position{row, col}
			day++
		}
	}

	s.Pieces = []Piece{
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}},        // Piece 1: L-shape
		{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {3, 1}},         // Piece 2: Long L
		{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {1, 2}},         // Piece 3: Cut Rectangle
		{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}}, // Piece 4: Rectangle
		{{0, 0}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},         // Piece 5: T-shape
		{{0, 0}, {0, 1}, {1, 1}, {2, 1}, {2, 2}},         // Piece 6: Z-shape
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {3, 0}},         // Piece 7: P-shape
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {3, 1}},         // Piece 8: Stair Shape
	}

	s.orientations = make([][]Piece, len(s.Pieces))
	for i, piece := range s.Pieces {
		s.orientations[i] = s.getAllOrientations(piece)
	}

	return s
}

// ValidateDate returns an error if day is not valid for the given month name.
// February accepts up to 29 days (the puzzle board includes a Feb 29 cell).
func (s *CalendarBoardSolver) ValidateDate(day int, month string) error {
	if day < 1 || day > 31 {
		return fmt.Errorf("day %d is out of range (must be 1–31)", day)
	}
	monthIdx := -1
	for i, m := range s.Months {
		if m == month {
			monthIdx = i + 1 // 1-indexed
			break
		}
	}
	if monthIdx == -1 {
		return fmt.Errorf("unknown month %q", month)
	}
	if day > maxDaysInMonth[monthIdx] {
		return fmt.Errorf("%s has at most %d days, got %d", month, maxDaysInMonth[monthIdx], day)
	}
	return nil
}

// ─── Piece geometry ──────────────────────────────────────────────────────────

func (s *CalendarBoardSolver) getAllOrientations(piece Piece) []Piece {
	seen := make(map[string]bool)
	var result []Piece
	add := func(p Piece) {
		n := s.normalizePiece(p)
		k := s.pieceToString(n)
		if !seen[k] {
			seen[k] = true
			result = append(result, n)
		}
	}
	cur := piece
	for i := 0; i < 4; i++ {
		add(cur)
		cur = s.rotatePiece90(cur)
	}
	flipped := s.flipHorizontal(piece)
	cur = flipped
	for i := 0; i < 4; i++ {
		add(cur)
		cur = s.rotatePiece90(cur)
	}
	return result
}

func (s *CalendarBoardSolver) rotatePiece90(piece Piece) Piece {
	rotated := make(Piece, len(piece))
	for i, pos := range piece {
		rotated[i] = Position{pos.Col, -pos.Row}
	}
	return s.normalizePiece(rotated)
}

func (s *CalendarBoardSolver) flipHorizontal(piece Piece) Piece {
	flipped := make(Piece, len(piece))
	for i, pos := range piece {
		flipped[i] = Position{-pos.Row, pos.Col}
	}
	return s.normalizePiece(flipped)
}

func (s *CalendarBoardSolver) normalizePiece(piece Piece) Piece {
	if len(piece) == 0 {
		return piece
	}
	minRow, minCol := piece[0].Row, piece[0].Col
	for _, pos := range piece {
		if pos.Row < minRow {
			minRow = pos.Row
		}
		if pos.Col < minCol {
			minCol = pos.Col
		}
	}
	normalized := make(Piece, len(piece))
	for i, pos := range piece {
		normalized[i] = Position{pos.Row - minRow, pos.Col - minCol}
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Row == normalized[j].Row {
			return normalized[i].Col < normalized[j].Col
		}
		return normalized[i].Row < normalized[j].Row
	})
	return normalized
}

func (s *CalendarBoardSolver) pieceToString(piece Piece) string {
	parts := make([]string, len(piece))
	for i, pos := range piece {
		parts[i] = fmt.Sprintf("%d,%d", pos.Row, pos.Col)
	}
	return strings.Join(parts, ";")
}

// ─── Board operations ────────────────────────────────────────────────────────

// initBoard builds the initial board with invalid/blocked cells pre-marked as -1.
// Returns the board and the number of empty cells that must be filled.
func (s *CalendarBoardSolver) initBoard(blockedCells map[Position]bool) (Board, int) {
	var board Board
	for r := range board {
		for c := range board[r] {
			board[r][c] = -1
		}
	}
	target := 0
	mark := func(pos Position) {
		if blockedCells[pos] {
			board[pos.Row][pos.Col] = -1
		} else {
			board[pos.Row][pos.Col] = 0
			target++
		}
	}
	for _, pos := range s.MonthPositions {
		mark(pos)
	}
	for _, pos := range s.DayPositions {
		mark(pos)
	}
	return board, target
}

// firstEmpty returns the first empty (value 0) cell in row-major order.
func (s *CalendarBoardSolver) firstEmpty(board *Board) (row, col int, ok bool) {
	for r := 0; r < 7; r++ {
		for c := 0; c < 7; c++ {
			if board[r][c] == 0 {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

func (s *CalendarBoardSolver) canPlace(board *Board, orient Piece, startRow, startCol int) bool {
	for _, off := range orient {
		r, c := startRow+off.Row, startCol+off.Col
		if r < 0 || r >= 7 || c < 0 || c >= 7 || board[r][c] != 0 {
			return false
		}
	}
	return true
}

func (s *CalendarBoardSolver) place(board *Board, orient Piece, startRow, startCol, pieceNum int) {
	for _, off := range orient {
		board[startRow+off.Row][startCol+off.Col] = int8(pieceNum)
	}
}

func (s *CalendarBoardSolver) unplace(board *Board, orient Piece, startRow, startCol int) {
	for _, off := range orient {
		board[startRow+off.Row][startCol+off.Col] = 0
	}
}

// ─── Parallel solver ─────────────────────────────────────────────────────────

type workItem struct {
	board      Board
	usedPieces [8]bool
	filled     int
}

// generateFirstLevel expands every valid one-piece placement that covers the
// first empty cell, returning those states as initial work items for workers.
// This gives each worker a genuinely distinct subtree to explore.
func (s *CalendarBoardSolver) generateFirstLevel(board Board) []workItem {
	tr, tc, ok := s.firstEmpty(&board)
	if !ok {
		return nil
	}
	var items []workItem
	var usedPieces [8]bool
	for pi := range s.Pieces {
		for _, orient := range s.orientations[pi] {
			for _, cell := range orient {
				startRow := tr - cell.Row
				startCol := tc - cell.Col
				if s.canPlace(&board, orient, startRow, startCol) {
					nb := board
					s.place(&nb, orient, startRow, startCol, pi+1)
					nu := usedPieces
					nu[pi] = true
					items = append(items, workItem{nb, nu, len(orient)})
				}
			}
		}
	}
	return items
}

// SolveParallel solves the puzzle for the given date using parallel backtracking.
// Returns an error if the date combination is impossible (e.g. Feb 30).
func (s *CalendarBoardSolver) SolveParallel(currentDay int, currentMonth string) (SolveResult, error) {
	if err := s.ValidateDate(currentDay, currentMonth); err != nil {
		return SolveResult{}, err
	}

	startTime := time.Now()
	blockedCells := map[Position]bool{
		s.MonthPositions[currentMonth]: true,
		s.DayPositions[currentDay]:     true,
	}

	board, target := s.initBoard(blockedCells)

	totalPieceCells := 0
	for _, p := range s.Pieces {
		totalPieceCells += len(p)
	}
	fmt.Printf("Target positions to fill: %d\n", target)
	fmt.Printf("Total piece cells: %d\n", totalPieceCells)
	if totalPieceCells != target {
		fmt.Printf("WARNING: Piece cells (%d) != target positions (%d)\n", totalPieceCells, target)
	}

	items := s.generateFirstLevel(board)
	fmt.Printf("Initial work items: %d\n", len(items))

	numWorkers := runtime.NumCPU()
	runtime.GOMAXPROCS(numWorkers)

	workChan := make(chan workItem, len(items))
	for _, item := range items {
		workChan <- item
	}
	close(workChan)

	resultChan := make(chan SolveResult, 1)
	doneChan := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(doneChan) }) }

	var globalAttempts int64
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for item := range workChan {
				select {
				case <-doneChan:
					return
				default:
				}
				b := item.board
				u := item.usedPieces
				if s.backtrack(&b, &u, item.filled, target, &globalAttempts, resultChan, doneChan, id) {
					stop()
					return
				}
			}
		}(i)
	}

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	var result SolveResult
	select {
	case r := <-resultChan:
		stop()
		result = r
	case <-allDone:
		// All workers finished; pick up a result if one was just sent.
		select {
		case r := <-resultChan:
			result = r
		default:
			result = SolveResult{Found: false}
		}
	case <-time.After(60 * time.Second):
		stop()
		result = SolveResult{Found: false}
	}

	result.SolveTime = time.Since(startTime)
	result.Attempts = atomic.LoadInt64(&globalAttempts)
	return result, nil
}

// backtrack is the core recursive search.
//
// Key optimisation — "first empty cell" strategy: instead of trying every board
// position for every piece, we find the first empty cell (row-major) and only
// try placements that cover that specific cell. Because that cell must be
// covered by some piece in any valid solution, this eliminates all placements
// that leave it uncovered and massively prunes the search tree.
func (s *CalendarBoardSolver) backtrack(
	board *Board, usedPieces *[8]bool,
	filled, target int,
	attempts *int64,
	resultChan chan<- SolveResult,
	doneChan <-chan struct{},
	workerID int,
) bool {
	atomic.AddInt64(attempts, 1)

	if filled == target {
		pieceMap := make(map[Position]int, target)
		for r := 0; r < 7; r++ {
			for c := 0; c < 7; c++ {
				if v := board[r][c]; v > 0 {
					pieceMap[Position{r, c}] = int(v)
				}
			}
		}
		solution := make([]Position, 0, len(pieceMap))
		for pos := range pieceMap {
			solution = append(solution, pos)
		}
		select {
		case resultChan <- SolveResult{
			Solution: solution,
			PieceMap: pieceMap,
			Found:    true,
			WorkerID: workerID,
		}:
		default:
		}
		return true
	}

	select {
	case <-doneChan:
		return true
	default:
	}

	tr, tc, ok := s.firstEmpty(board)
	if !ok {
		return false
	}

	for pi := range s.Pieces {
		if usedPieces[pi] {
			continue
		}
		for _, orient := range s.orientations[pi] {
			// Try each cell of this orientation as the one that covers (tr, tc).
			for _, cell := range orient {
				startRow := tr - cell.Row
				startCol := tc - cell.Col
				if s.canPlace(board, orient, startRow, startCol) {
					s.place(board, orient, startRow, startCol, pi+1)
					usedPieces[pi] = true
					if s.backtrack(board, usedPieces, filled+len(orient), target, attempts, resultChan, doneChan, workerID) {
						return true
					}
					s.unplace(board, orient, startRow, startCol)
					usedPieces[pi] = false
				}
			}
		}
	}
	return false
}

// ─── Visualization ───────────────────────────────────────────────────────────

func (s *CalendarBoardSolver) VisualizeSolution(currentDay int, currentMonth string, solution []Position, pieceMap map[Position]int) {
	fmt.Printf("\nSolution for %d %s:\n", currentDay, currentMonth)
	fmt.Println("=" + strings.Repeat("=", 29))

	board := make([][]string, 7)
	for i := range board {
		board[i] = make([]string, 7)
		for j := range board[i] {
			board[i][j] = "."
		}
	}

	for month, pos := range s.MonthPositions {
		if month == currentMonth {
			board[pos.Row][pos.Col] = "X"
		} else if pieceNum, exists := pieceMap[pos]; exists {
			board[pos.Row][pos.Col] = fmt.Sprintf("%d", pieceNum)
		}
	}

	for day, pos := range s.DayPositions {
		if day == currentDay {
			board[pos.Row][pos.Col] = "X"
		} else if pieceNum, exists := pieceMap[pos]; exists {
			board[pos.Row][pos.Col] = fmt.Sprintf("%d", pieceNum)
		}
	}

	for _, row := range board {
		fmt.Println(strings.Join(row, " "))
	}

	fmt.Printf("\nX = Current date (%d %s)\n", currentDay, currentMonth)
	fmt.Println("1-8 = Piece numbers")
	fmt.Println(". = Empty/Invalid positions")
}

func (s *CalendarBoardSolver) PrintBoardConfiguration() {
	fmt.Println("BOARD CONFIGURATION:")
	fmt.Println("=" + strings.Repeat("=", 49))

	board := make([][]string, 7)
	for i := range board {
		board[i] = make([]string, 7)
		for j := range board[i] {
			board[i][j] = "   "
		}
	}

	for month, pos := range s.MonthPositions {
		board[pos.Row][pos.Col] = fmt.Sprintf("%3s", month)
	}
	for day, pos := range s.DayPositions {
		board[pos.Row][pos.Col] = fmt.Sprintf("%3d", day)
	}

	fmt.Print("    ")
	for col := 0; col < 7; col++ {
		fmt.Printf("%4d", col)
	}
	fmt.Println()

	for row := 0; row < 7; row++ {
		fmt.Printf("%d: ", row)
		for col := 0; col < 7; col++ {
			cell := board[row][col]
			if cell == "   " {
				fmt.Print("  . ")
			} else {
				fmt.Printf("%s ", cell)
			}
		}
		fmt.Println()
	}

	validCells := len(s.MonthPositions) + len(s.DayPositions)
	fmt.Printf("\nBoard Statistics:\n")
	fmt.Printf("- Total grid size: 7x7 = 49 positions\n")
	fmt.Printf("- Valid calendar cells: %d\n", validCells)
	fmt.Printf("- Month cells: %d\n", len(s.MonthPositions))
	fmt.Printf("- Day cells: %d\n", len(s.DayPositions))
	fmt.Printf("- Empty/Invalid positions: %d\n", 49-validCells)
	fmt.Printf("- Expected filled cells per solution: %d (total - current date)\n", validCells-2)
}

func (s *CalendarBoardSolver) PrintPiecesConfiguration() {
	fmt.Println("\nBRICK PIECES CONFIGURATION:")
	fmt.Println("=" + strings.Repeat("=", 49))

	pieceNames := []string{
		"Piece 1: L-shape",
		"Piece 2: Long L",
		"Piece 3: Cut Rectangle",
		"Piece 4: Rectangle",
		"Piece 5: T-shape",
		"Piece 6: Z-shape",
		"Piece 7: P-shape",
		"Piece 8: Stair Shape",
	}

	totalCells := 0
	for i, piece := range s.Pieces {
		fmt.Printf("\n%s (%d cells):\n", pieceNames[i], len(piece))
		totalCells += len(piece)

		maxRow, maxCol := 0, 0
		for _, pos := range piece {
			if pos.Row > maxRow {
				maxRow = pos.Row
			}
			if pos.Col > maxCol {
				maxCol = pos.Col
			}
		}

		grid := make([][]string, maxRow+1)
		for j := range grid {
			grid[j] = make([]string, maxCol+1)
			for k := range grid[j] {
				grid[j][k] = "."
			}
		}
		for _, pos := range piece {
			grid[pos.Row][pos.Col] = "A"
		}
		for _, row := range grid {
			fmt.Print("  ")
			fmt.Println(strings.Join(row, " "))
		}

		fmt.Printf("  Coordinates: %v\n", piece)
		orientations := s.getAllOrientations(piece)
		fmt.Printf("  Total orientations: %d\n", len(orientations))
	}

	fmt.Printf("\nTotal pieces: %d\n", len(s.Pieces))
	fmt.Printf("Total cells in all pieces: %d\n", totalCells)
	fmt.Printf("Expected coverage: %d cells\n", totalCells)
}
