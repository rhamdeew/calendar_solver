//go:build js
// +build js

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"puzzle_solver/solver"
)

func main() {
	fmt.Println("Hello, WebAssembly!")
	js.Global().Set("solveCalendar", js.FuncOf(solveCalendar))
	// Keep the Go program alive for JS calls
	select {}
}

// solveCalendar is a wrapper for the solver logic to be called from JS.
func solveCalendar(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return "Invalid number of arguments"
	}

	day := args[0].Int()
	monthIndex := args[1].Int() // Expecting month as 1-12

	month, err := monthFromIndex(monthIndex)
	if err != nil {
		return err.Error()
	}

	s := solver.NewCalendarBoardSolver()
	result := s.SolveParallel(day, month)

	pieceMapForJS := make(map[string]int)
	for pos, pieceNum := range result.PieceMap {
		key := fmt.Sprintf("%d,%d", pos.Row, pos.Col)
		pieceMapForJS[key] = pieceNum
	}

	resultMap := map[string]interface{}{
		"solution":    result.Solution,
		"pieceMap":    pieceMapForJS,
		"found":       result.Found,
		"solveTime":   result.SolveTime.String(),
		"attempts":    result.Attempts,
	}

	resultJSON, err := json.Marshal(resultMap)
	if err != nil {
		return "Error marshalling result to JSON"
	}

	return string(resultJSON)
}

// monthFromIndex converts a 1-based month index to its string representation.
func monthFromIndex(index int) (string, error) {
	if index < 1 || index > 12 {
		return "", fmt.Errorf("invalid month index: %d", index)
	}
	months := []string{
		"Янв", "Фев", "Март", "Апр", "Май", "Июнь",
		"Июль", "Авг", "Сент", "Окт", "Нояб", "Дек",
	}
	return months[index-1], nil
}