package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Position struct {
	Row, Col int
}

type Piece []Position

type CalendarBoardSolver struct {
	Months         []string
	MonthPositions map[string]Position
	DayPositions   map[int]Position
	Pieces         []Piece
}

type SolveResult struct {
	Solution    []Position
	PieceMap    map[Position]int  // Maps position to piece number (1-8)
	Found       bool
	SolveTime   time.Duration
	Attempts    int64
	WorkerID    int
}

type WorkItem struct {
	Board     map[Position]bool
	PieceMap  map[Position]int  // Maps position to piece number (1-8)
	UsedPieces []bool
	Depth     int
}

func NewCalendarBoardSolver() *CalendarBoardSolver {
	solver := &CalendarBoardSolver{
		Months: []string{
			"Янв", "Фев", "Март", "Апр", "Май", "Июнь",
			"Июль", "Авг", "Сент", "Окт", "Нояб", "Дек",
		},
		MonthPositions: make(map[string]Position),
		DayPositions:   make(map[int]Position),
	}

	// Initialize month positions
	months := []string{"Янв", "Фев", "Март", "Апр", "Май", "Июнь", "Июль", "Авг", "Сент", "Окт", "Нояб", "Дек"}
	for i, month := range months {
		if i < 6 {
			solver.MonthPositions[month] = Position{0, i}
		} else {
			solver.MonthPositions[month] = Position{1, i - 6}
		}
	}

	// Initialize day positions
	day := 1
	for row := 2; row < 7; row++ {
		for col := 0; col < 7 && day <= 31; col++ {
			solver.DayPositions[day] = Position{row, col}
			day++
		}
	}

	// Initialize pieces - these are the 8 unique pieces that total 41 cells
	solver.Pieces = []Piece{
		// Piece 1: L-shape (5 cells)
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2}},
		// Piece 2: Long L (5 cells)
		{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {3, 1}},
		// Piece 3: Cut Rectangle (5 cells)
		{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {1, 2}},
		// Piece 4: Rectangle (6 cells)
		{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},
		// Piece 5: T-shape (5 cells)
		{{0, 0}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},
		// Piece 6: Z-shape (5 cells)
		{{0, 0}, {0, 1}, {1, 1}, {2, 1}, {2, 2}},
		// Piece 7: P-shape (5 cells)
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {3, 0}},
		// Piece 8: Stair Shape (5 cells)
		{{0, 0}, {1, 0}, {2, 0}, {2, 1}, {3, 1}},
	}

	return solver
}

func (s *CalendarBoardSolver) getAllOrientations(piece Piece) []Piece {
	orientations := make([]Piece, 0, 8)
	seen := make(map[string]bool)

	// Generate all 4 rotations
	current := piece
	for i := 0; i < 4; i++ {
		// Add current rotation
		normalized := s.normalizePiece(current)
		key := s.pieceToString(normalized)
		if !seen[key] {
			orientations = append(orientations, normalized)
			seen[key] = true
		}

		// Rotate 90 degrees clockwise for next iteration
		current = s.rotatePiece90(current)
	}

	// Generate all 4 rotations of the flipped piece
	flipped := s.flipHorizontal(piece)
	current = flipped
	for i := 0; i < 4; i++ {
		// Add current rotation of flipped piece
		normalized := s.normalizePiece(current)
		key := s.pieceToString(normalized)
		if !seen[key] {
			orientations = append(orientations, normalized)
			seen[key] = true
		}

		// Rotate 90 degrees clockwise for next iteration
		current = s.rotatePiece90(current)
	}

	return orientations
}

func (s *CalendarBoardSolver) rotatePiece90(piece Piece) Piece {
	rotated := make(Piece, len(piece))
	for i, pos := range piece {
		// Rotate 90 degrees clockwise: (x, y) -> (y, -x)
		rotated[i] = Position{pos.Col, -pos.Row}
	}
	return s.normalizePiece(rotated)
}

func (s *CalendarBoardSolver) flipHorizontal(piece Piece) Piece {
	flipped := make(Piece, len(piece))
	for i, pos := range piece {
		// Flip horizontally: (x, y) -> (-x, y)
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

	// Sort positions for consistent representation
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Row == normalized[j].Row {
			return normalized[i].Col < normalized[j].Col
		}
		return normalized[i].Row < normalized[j].Row
	})

	return normalized
}

func (s *CalendarBoardSolver) pieceToString(piece Piece) string {
	positions := make([]string, len(piece))
	for i, pos := range piece {
		positions[i] = fmt.Sprintf("%d,%d", pos.Row, pos.Col)
	}
	return strings.Join(positions, ";")
}

func (s *CalendarBoardSolver) isValidCalendarPosition(row, col int) bool {
	// Month positions (rows 0-1, cols 0-5)
	if row >= 0 && row <= 1 && col >= 0 && col <= 5 {
		return true
	}
	// Day positions (rows 2-6, cols 0-6, but only first 31 positions)
	if row >= 2 && row <= 6 && col >= 0 && col <= 6 {
		dayNum := (row-2)*7 + col + 1
		return dayNum <= 31
	}
	return false
}

func (s *CalendarBoardSolver) canPlacePiece(board map[Position]bool, piece Piece, startRow, startCol int, blockedCells map[Position]bool) bool {
	for _, offset := range piece {
		pos := Position{startRow + offset.Row, startCol + offset.Col}

		// Check bounds
		if pos.Row < 0 || pos.Row >= 7 || pos.Col < 0 || pos.Col >= 7 {
			return false
		}

		// Check if already occupied
		if board[pos] {
			return false
		}

		// Check if blocked (current date)
		if blockedCells[pos] {
			return false
		}

		// Check if valid calendar position
		if !s.isValidCalendarPosition(pos.Row, pos.Col) {
			return false
		}
	}
	return true
}

func (s *CalendarBoardSolver) placePiece(board map[Position]bool, pieceMap map[Position]int, piece Piece, startRow, startCol int, pieceNumber int) (map[Position]bool, map[Position]int) {
	newBoard := make(map[Position]bool)
	for pos := range board {
		newBoard[pos] = true
	}

	newPieceMap := make(map[Position]int)
	for pos, pieceNum := range pieceMap {
		newPieceMap[pos] = pieceNum
	}

	for _, offset := range piece {
		pos := Position{startRow + offset.Row, startCol + offset.Col}
		newBoard[pos] = true
		newPieceMap[pos] = pieceNumber
	}

	return newBoard, newPieceMap
}

func (s *CalendarBoardSolver) SolveParallel(currentDay int, currentMonth string) SolveResult {
	startTime := time.Now()

	// Get blocked positions
	blockedCells := make(map[Position]bool)
	blockedCells[s.MonthPositions[currentMonth]] = true
	blockedCells[s.DayPositions[currentDay]] = true

	// Get all valid positions
	allPositions := make(map[Position]bool)
	for _, pos := range s.MonthPositions {
		allPositions[pos] = true
	}
	for _, pos := range s.DayPositions {
		allPositions[pos] = true
	}

	// Target positions (all except blocked)
	targetPositions := make([]Position, 0)
	for pos := range allPositions {
		if !blockedCells[pos] {
			targetPositions = append(targetPositions, pos)
		}
	}

	targetSize := len(targetPositions)

	fmt.Printf("Target positions to fill: %d\n", targetSize)

	// Calculate total cells in all pieces
	totalPieceCells := 0
	for _, piece := range s.Pieces {
		totalPieceCells += len(piece)
	}
	fmt.Printf("Total piece cells: %d\n", totalPieceCells)

	if totalPieceCells != targetSize {
		fmt.Printf("WARNING: Piece cells (%d) != target positions (%d)\n", totalPieceCells, targetSize)
	}

	// Parallel solving setup
	numWorkers := runtime.NumCPU()
	runtime.GOMAXPROCS(numWorkers)

	resultChan := make(chan SolveResult, 1)
	workChan := make(chan WorkItem, 10000)
	doneChan := make(chan bool)

	var globalAttempts int64
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.worker(i, workChan, resultChan, doneChan, &globalAttempts, targetSize, blockedCells, &wg)
	}

	// Generate initial work items
	go func() {
		defer close(workChan)

		initialBoard := make(map[Position]bool)
		initialPieceMap := make(map[Position]int)
		usedPieces := make([]bool, len(s.Pieces))

		select {
		case workChan <- WorkItem{
			Board:      initialBoard,
			PieceMap:   initialPieceMap,
			UsedPieces: usedPieces,
			Depth:      0,
		}:
		case <-doneChan:
			return
		}
	}()

	// Wait for result or timeout
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Get result with longer timeout for complex problems
	select {
	case result := <-resultChan:
		close(doneChan)
		result.SolveTime = time.Since(startTime)
		result.Attempts = atomic.LoadInt64(&globalAttempts)
		return result
	case <-time.After(60 * time.Second): // Increased timeout to 60 seconds
		close(doneChan)
		return SolveResult{
			Found:     false,
			SolveTime: time.Since(startTime),
			Attempts:  atomic.LoadInt64(&globalAttempts),
		}
	}
}

func (s *CalendarBoardSolver) worker(workerID int, workChan <-chan WorkItem, resultChan chan<- SolveResult, doneChan <-chan bool, globalAttempts *int64, targetSize int, blockedCells map[Position]bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-doneChan:
			return
		case work, ok := <-workChan:
			if !ok {
				return
			}

			if s.backtrack(work, targetSize, blockedCells, globalAttempts, resultChan, doneChan, workerID) {
				return
			}
		}
	}
}

func (s *CalendarBoardSolver) backtrack(work WorkItem, targetSize int, blockedCells map[Position]bool, globalAttempts *int64, resultChan chan<- SolveResult, doneChan <-chan bool, workerID int) bool {
	atomic.AddInt64(globalAttempts, 1)

	// Check if we found a solution
	if len(work.Board) == targetSize {
		solution := make([]Position, 0, len(work.Board))
		for pos := range work.Board {
			solution = append(solution, pos)
		}

		select {
		case resultChan <- SolveResult{
			Solution: solution,
			PieceMap: work.PieceMap,
			Found:    true,
			WorkerID: workerID,
		}:
		case <-doneChan:
		}
		return true
	}

	// Check if we should stop
	select {
	case <-doneChan:
		return true
	default:
	}

	// Find next unused piece
	nextPieceIndex := -1
	for i, used := range work.UsedPieces {
		if !used {
			nextPieceIndex = i
			break
		}
	}

	if nextPieceIndex == -1 {
		return false // All pieces used but board not full
	}

	// Try all orientations of the next piece
	orientations := s.getAllOrientations(s.Pieces[nextPieceIndex])

	for _, orientation := range orientations {
		// Try placing this orientation at all valid positions
		for row := 0; row < 7; row++ {
			for col := 0; col < 7; col++ {
				if s.canPlacePiece(work.Board, orientation, row, col, blockedCells) {
					newBoard, newPieceMap := s.placePiece(work.Board, work.PieceMap, orientation, row, col, nextPieceIndex+1)
					newUsedPieces := make([]bool, len(work.UsedPieces))
					copy(newUsedPieces, work.UsedPieces)
					newUsedPieces[nextPieceIndex] = true

					newWork := WorkItem{
						Board:      newBoard,
						PieceMap:   newPieceMap,
						UsedPieces: newUsedPieces,
						Depth:      work.Depth + 1,
					}

					// Process all work directly to avoid channel complications
					if s.backtrack(newWork, targetSize, blockedCells, globalAttempts, resultChan, doneChan, workerID) {
						return true
					}
				}
			}
		}
	}

	return false
}

func (s *CalendarBoardSolver) VisualizeSolution(currentDay int, currentMonth string, solution []Position, pieceMap map[Position]int) {
	fmt.Printf("\nSolution for %d %s:\n", currentDay, currentMonth)
	fmt.Println("=" + strings.Repeat("=", 29))

	// Create visual board
	board := make([][]string, 7)
	for i := range board {
		board[i] = make([]string, 7)
		for j := range board[i] {
			board[i][j] = "."
		}
	}

	// Mark month positions
	for month, pos := range s.MonthPositions {
		if month == currentMonth {
			board[pos.Row][pos.Col] = "X" // Current month (blocked)
		} else if pieceNum, exists := pieceMap[pos]; exists {
			board[pos.Row][pos.Col] = fmt.Sprintf("%d", pieceNum) // Show piece number
		}
	}

	// Mark day positions
	for day, pos := range s.DayPositions {
		if day == currentDay {
			board[pos.Row][pos.Col] = "X" // Current day (blocked)
		} else if pieceNum, exists := pieceMap[pos]; exists {
			board[pos.Row][pos.Col] = fmt.Sprintf("%d", pieceNum) // Show piece number
		}
	}

	// Print board
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

	// Create board with labels
	board := make([][]string, 7)
	for i := range board {
		board[i] = make([]string, 7)
		for j := range board[i] {
			board[i][j] = "   "
		}
	}

	// Fill month positions
	for month, pos := range s.MonthPositions {
		board[pos.Row][pos.Col] = fmt.Sprintf("%3s", month)
	}

	// Fill day positions
	for day, pos := range s.DayPositions {
		board[pos.Row][pos.Col] = fmt.Sprintf("%3d", day)
	}

	// Print board with row/column indicators
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

	// Count valid cells
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

		// Find bounds
		maxRow, maxCol := 0, 0
		for _, pos := range piece {
			if pos.Row > maxRow {
				maxRow = pos.Row
			}
			if pos.Col > maxCol {
				maxCol = pos.Col
			}
		}

		// Create visual grid
		grid := make([][]string, maxRow+1)
		for j := range grid {
			grid[j] = make([]string, maxCol+1)
			for k := range grid[j] {
				grid[j][k] = "."
			}
		}

		// Fill piece positions
		for _, pos := range piece {
			grid[pos.Row][pos.Col] = "A"
		}

		// Print the piece
		for _, row := range grid {
			fmt.Print("  ")
			fmt.Println(strings.Join(row, " "))
		}

		// Print coordinates
		fmt.Printf("  Coordinates: %v\n", piece)

		// Show some orientations
		orientations := s.getAllOrientations(piece)
		fmt.Printf("  Total orientations: %d\n", len(orientations))
	}

	fmt.Printf("\nTotal pieces: %d\n", len(s.Pieces))
	fmt.Printf("Total cells in all pieces: %d\n", totalCells)
	fmt.Printf("Expected coverage: %d cells\n", totalCells)
}

func getMonthName(monthInput string, months []string) (string, error) {
	// If it's already a Russian month name
	for _, month := range months {
		if monthInput == month {
			return month, nil
		}
	}

	// If it's a number (1-12)
	if monthNum, err := strconv.Atoi(monthInput); err == nil {
		if monthNum >= 1 && monthNum <= 12 {
			return months[monthNum-1], nil
		}
	}

	// Try partial matching
	for _, month := range months {
		if strings.HasPrefix(strings.ToLower(month), strings.ToLower(monthInput)) {
			return month, nil
		}
	}

	return "", fmt.Errorf("invalid month: %s", monthInput)
}

func main() {
	var day = flag.Int("day", -1, "Day (1-31)")
	var month = flag.String("month", "", "Month (Янв, Фев, Март, etc. or 1-12)")
	var testOnly = flag.Bool("test-only", false, "Skip main solve, run only test cases")
	flag.Parse()

	solver := NewCalendarBoardSolver()

	// Print board configuration
	solver.PrintBoardConfiguration()

	// Print pieces configuration
	solver.PrintPiecesConfiguration()

	fmt.Println("\n" + strings.Repeat("=", 50))

	// Determine target date
	var currentDay int
	var currentMonth string

	if *day != -1 && *month != "" {
		currentDay = *day
		var err error
		currentMonth, err = getMonthName(*month, solver.Months)
		if err != nil {
			log.Printf("Error: %v", err)
			fmt.Printf("Available months: %s\n", strings.Join(solver.Months, ", "))
			return
		}
		fmt.Printf("Command line date: %d %s\n", currentDay, currentMonth)
	} else if !*testOnly {
		// Use current date
		now := time.Now()
		currentDay = now.Day()
		currentMonth = solver.Months[now.Month()-1]
		fmt.Printf("Using current date: %d %s\n", currentDay, currentMonth)
	}

	// Solve for main date (unless test-only mode)
	var mainSolveTime time.Duration
	var mainSolutionFound bool
	var mainAttempts int64

	if !*testOnly && currentDay != 0 {
		fmt.Printf("\nSolving calendar board for: %d %s\n", currentDay, currentMonth)
		fmt.Printf("Available pieces: %d pieces\n", len(solver.Pieces))
		fmt.Printf("Available CPU cores: %d\n", runtime.NumCPU())
		fmt.Print("Piece sizes: [")
		for i, piece := range solver.Pieces {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%d", len(piece))
		}
		fmt.Println("] cells each")

		// Solve for target date
		result := solver.SolveParallel(currentDay, currentMonth)

		if result.Found {
			fmt.Printf("\n✓ Solution found in %.4f seconds!\n", result.SolveTime.Seconds())
			fmt.Printf("Worker %d found the solution after %d attempts\n", result.WorkerID, result.Attempts)
			if result.SolveTime.Seconds() > 0 {
				fmt.Printf("Attempts per second: %.0f\n", float64(result.Attempts)/result.SolveTime.Seconds())
			}
			solver.VisualizeSolution(currentDay, currentMonth, result.Solution, result.PieceMap)
		} else {
			fmt.Printf("\n✗ No solution found for %d %s (took %.4f seconds)\n", currentDay, currentMonth, result.SolveTime.Seconds())
			fmt.Printf("Total attempts: %d\n", result.Attempts)
			fmt.Println("This might require adjustment of pieces or board layout.")
		}

		mainSolveTime = result.SolveTime
		mainSolutionFound = result.Found
		mainAttempts = result.Attempts
	}

	// Test different dates (only if no specific date was provided)
	var totalTestTime time.Duration
	successfulSolves := 0
	var testDates [][2]interface{}

	// Skip testing other dates if a specific date was provided via command line
	if *day == -1 || *month == "" {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("TESTING OTHER DATES:")

		testDates = [][2]interface{}{
			{1, "Янв"},
			{15, "Март"},
			{31, "Дек"},
			{29, "Фев"},
		}

		for _, testDate := range testDates {
			day := testDate[0].(int)
			month := testDate[1].(string)

			fmt.Printf("\nTesting %d %s...\n", day, month)
			result := solver.SolveParallel(day, month)
			totalTestTime += result.SolveTime

			if result.Found {
				fmt.Printf("✓ Solution exists for %d %s (solved in %.4fs, %d attempts)\n",
					day, month, result.SolveTime.Seconds(), result.Attempts)
				successfulSolves++
			} else {
				fmt.Printf("✗ No solution for %d %s (took %.4fs, %d attempts)\n",
					day, month, result.SolveTime.Seconds(), result.Attempts)
			}
		}
	} else {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("Skipping test dates since specific date was provided: %d %s\n", currentDay, currentMonth)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("OVERALL PERFORMANCE SUMMARY:")
	if !*testOnly {
		fmt.Printf("- Main solve time: %.4f seconds\n", mainSolveTime.Seconds())
		fmt.Printf("- Main solve attempts: %d\n", mainAttempts)
	}
	fmt.Printf("- Test cases time: %.4f seconds\n", totalTestTime.Seconds())
	fmt.Printf("- Total execution time: %.4f seconds\n", (mainSolveTime + totalTestTime).Seconds())

	totalCases := len(testDates)
	totalSuccessful := successfulSolves
	if !*testOnly {
		totalCases++
		if mainSolutionFound {
			totalSuccessful++
		}
	}

	fmt.Printf("- Successful solves: %d/%d\n", totalSuccessful, totalCases)

	if totalCases > 0 {
		avgTime := (mainSolveTime + totalTestTime).Seconds() / float64(totalCases)
		fmt.Printf("- Average solve time: %.4f seconds\n", avgTime)
	}
	fmt.Printf("- Used %d CPU cores for parallel processing\n", runtime.NumCPU())
}
