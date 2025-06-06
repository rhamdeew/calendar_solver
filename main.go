package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
