package tui

type fittedRowTable[T any] struct {
	width   int
	header  string
	rows    []T
	limit   int
	rowLine func(T) string
}

func appendFittedRows[T any](lines []string, table fittedRowTable[T]) []string {
	lines = append(lines, fit(table.header, table.width))
	for _, row := range limitSlice(table.rows, table.limit) {
		lines = append(lines, fit(table.rowLine(row), table.width))
	}
	return lines
}
