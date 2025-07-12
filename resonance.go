package main

import (
	"encoding/json"
	"strings"
)

func Transmute(edi string) (csv string, jsonOut string, logTrace string) {
	lines := strings.Split(edi, "\n")
	var rows [][]string
	logTrace = "Transmuting EDI into CSV and JSON...\n"

	for _, line := range lines {
		elements := strings.Split(line, "*")
		rows = append(rows, elements)
		logTrace += "Parsed: " + strings.Join(elements, ",") + "\n"
	}

	// Convert to CSV
	for _, row := range rows {
		csv += strings.Join(row, ",") + "\n"
	}

	// Convert to JSON
	jsonData, _ := json.Marshal(rows)
	jsonOut = string(jsonData)

	logTrace += "Transmutation complete.\n"
	return
}