package tui

type fittedRowTable[T any] struct {
	width   int
	header  string
	rows    []T
	limit   int
	rowLine func(T) string
}

type fittedRowSection[T any] struct {
	title string
	empty string
	table fittedRowTable[T]
}

type fittedLineWriter struct {
	lines []string
	width int
}

func newFittedLineWriter(lines []string, width int) *fittedLineWriter {
	return &fittedLineWriter{
		lines: lines,
		width: width,
	}
}

func (w *fittedLineWriter) append(lines ...string) {
	w.lines = append(w.lines, lines...)
}

func (w *fittedLineWriter) appendFit(line string) {
	w.lines = append(w.lines, fit(line, w.width))
}

func (w *fittedLineWriter) result() []string {
	return w.lines
}

func appendFittedRows[T any](lines []string, table fittedRowTable[T]) []string {
	lines = append(lines, fit(table.header, table.width))
	for _, row := range limitSlice(table.rows, table.limit) {
		lines = append(lines, fit(table.rowLine(row), table.width))
	}
	return lines
}

func appendFittedLineRows[T any](w *fittedLineWriter, table fittedRowTable[T]) {
	table.width = w.width
	w.lines = appendFittedRows(w.lines, table)
}

func appendFittedLineSection[T any](w *fittedLineWriter, section fittedRowSection[T]) bool {
	if section.title != "" {
		w.append("", bold(section.title))
	}
	if len(section.table.rows) == 0 {
		if section.empty != "" {
			w.append(section.empty)
		}
		return false
	}
	appendFittedLineRows(w, section.table)
	return true
}
