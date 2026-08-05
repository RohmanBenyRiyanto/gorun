package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// Alignment is a table cell's text alignment.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// ColumnConfig controls one column's alignment and width bounds. MaxWidth
// of 0 means unbounded.
type ColumnConfig struct {
	HeaderAlign  Alignment
	ContentAlign Alignment
	MinWidth     int
	MaxWidth     int
}

// Table is a terminal-rendered table with per-column alignment/width
// control. Build one with NewTable or ParseTable, add rows, then call
// DrawHorizontal (boxed grid) or DrawVertical (field: value list, for
// Headers of exactly two columns).
type Table struct {
	Headers      []string
	Rows         [][]string
	ColumnConfig []ColumnConfig
}

// NewTable builds an empty Table with every column left-aligned and a
// MinWidth of 8, ready for SetColumnConfig overrides and AddRow calls.
func NewTable(headers []string) *Table {
	config := make([]ColumnConfig, len(headers))
	for i := range config {
		config[i] = ColumnConfig{
			HeaderAlign:  AlignLeft,
			ContentAlign: AlignLeft,
			MinWidth:     8,
		}
	}
	return &Table{
		Headers:      headers,
		Rows:         make([][]string, 0),
		ColumnConfig: config,
	}
}

// SetColumnConfig overrides column col's alignment/width. Out-of-range
// col is silently ignored.
func (t *Table) SetColumnConfig(col int, config ColumnConfig) {
	if col >= 0 && col < len(t.ColumnConfig) {
		t.ColumnConfig[col] = config
	}
}

func (t *Table) formatCell(text string, width int, align Alignment) string {
	visibleLen := visibleWidth(text)
	if visibleLen >= width {
		return text
	}

	padding := width - visibleLen
	switch align {
	case AlignLeft:
		return text + repeatChar(" ", padding)
	case AlignRight:
		return repeatChar(" ", padding) + text
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return repeatChar(" ", left) + text + repeatChar(" ", right)
	default:
		return text + repeatChar(" ", padding)
	}
}

// DrawHorizontal renders the table as a bordered grid (headers + rows),
// wrapping cell text and scaling column widths down to fit the terminal
// width when the natural content width would overflow it.
func (t *Table) DrawHorizontal() {
	maxWidth := GetTerminalWidth()
	colCount := len(t.Headers)
	if colCount == 0 {
		return
	}

	borderSize := (colCount + 1) * 3
	maxContentWidth := maxWidth - borderSize

	colMax := make([]int, colCount)
	for i, header := range t.Headers {
		colMax[i] = visibleWidth(header)
		if t.ColumnConfig[i].MinWidth > colMax[i] {
			colMax[i] = t.ColumnConfig[i].MinWidth
		}
	}

	for _, row := range t.Rows {
		for i := 0; i < colCount && i < len(row); i++ {
			width := visibleWidth(row[i])
			if width > colMax[i] {
				colMax[i] = width
			}
		}
	}

	totalContentWidth := 0
	for _, width := range colMax {
		totalContentWidth += width
	}

	colWidths := make([]int, colCount)
	if totalContentWidth > maxContentWidth {
		scaleFactor := float64(maxContentWidth) / float64(totalContentWidth)
		for i, width := range colMax {
			newWidth := int(float64(width) * scaleFactor)
			if newWidth < t.ColumnConfig[i].MinWidth {
				newWidth = t.ColumnConfig[i].MinWidth
			}
			if t.ColumnConfig[i].MaxWidth > 0 && newWidth > t.ColumnConfig[i].MaxWidth {
				newWidth = t.ColumnConfig[i].MaxWidth
			}
			colWidths[i] = newWidth
		}
	} else {
		copy(colWidths, colMax)
	}

	fmt.Print(ColorForeground + "┌")
	for i := 0; i < colCount; i++ {
		fmt.Print(repeatChar("─", colWidths[i]+2))
		if i < colCount-1 {
			fmt.Print("┬")
		} else {
			fmt.Print("┐")
		}
	}
	fmt.Println(ColorReset)

	fmt.Print(ColorForeground + "│")
	for i, header := range t.Headers {
		fmt.Print(" " + ColorCyan)
		fmt.Print(t.formatCell(header, colWidths[i], t.ColumnConfig[i].HeaderAlign))
		fmt.Print(" " + ColorForeground + "│")
	}
	fmt.Println(ColorReset)

	fmt.Print(ColorForeground + "├")
	for i := 0; i < colCount; i++ {
		fmt.Print(repeatChar("─", colWidths[i]+2))
		if i < colCount-1 {
			fmt.Print("┼")
		} else {
			fmt.Print("┤")
		}
	}
	fmt.Println(ColorReset)

	for _, row := range t.Rows {
		wrappedCols := make([][]string, colCount)
		maxLines := 1

		for i := 0; i < colCount; i++ {
			cellValue := ""
			if i < len(row) {
				cellValue = row[i]
			}
			wrappedCols[i] = wordWrap(cellValue, colWidths[i])
			if len(wrappedCols[i]) > maxLines {
				maxLines = len(wrappedCols[i])
			}
		}

		for line := 0; line < maxLines; line++ {
			fmt.Print(ColorForeground + "│")
			for i := 0; i < colCount; i++ {
				fmt.Print(" ")
				if line < len(wrappedCols[i]) {
					cellText := wrappedCols[i][line]
					fmt.Print(t.formatCell(cellText, colWidths[i], t.ColumnConfig[i].ContentAlign))
				} else {
					fmt.Print(repeatChar(" ", colWidths[i]))
				}
				fmt.Print(" " + ColorForeground + "│")
			}
			fmt.Println(ColorReset)
		}
	}

	fmt.Print(ColorForeground + "└")
	for i := 0; i < colCount; i++ {
		fmt.Print(repeatChar("─", colWidths[i]+2))
		if i < colCount-1 {
			fmt.Print("┴")
		} else {
			fmt.Print("┘")
		}
	}
	fmt.Println(ColorReset)
}

// DrawVertical renders each row as a "field │ value" line instead of a
// grid - meant for a two-column Table (property/value listings like `db
// status`), using row[0]/row[1] regardless of how many headers were set.
func (t *Table) DrawVertical() {
	maxWidth := GetTerminalWidth()
	rowCount := len(t.Rows)
	if rowCount == 0 {
		return
	}

	maxFieldWidth := 0
	maxValueWidth := 0

	for _, row := range t.Rows {
		if len(row) >= 2 {
			field := strings.TrimSpace(row[0])
			value := strings.TrimSpace(row[1])

			fieldWidth := visibleWidth(field)
			if fieldWidth > maxFieldWidth {
				maxFieldWidth = fieldWidth
			}

			valueLines := wordWrap(value, maxWidth-maxFieldWidth-5)
			for _, line := range valueLines {
				lineWidth := visibleWidth(line)
				if lineWidth > maxValueWidth {
					maxValueWidth = lineWidth
				}
			}
		}
	}

	totalWidth := maxFieldWidth + maxValueWidth + 5
	if totalWidth > maxWidth {
		excess := totalWidth - maxWidth
		if maxValueWidth > excess {
			maxValueWidth -= excess
		} else {
			maxValueWidth = 2
			maxFieldWidth = maxWidth - maxValueWidth - 5
		}
	}

	fmt.Print(ColorForeground + "┌")
	fmt.Print(repeatChar("─", maxFieldWidth+2))
	fmt.Print("┬")
	fmt.Print(repeatChar("─", maxValueWidth+2))
	fmt.Println("┐" + ColorReset)

	for i, row := range t.Rows {
		field := ""
		value := ""
		if len(row) >= 1 {
			field = strings.TrimSpace(row[0])
		}
		if len(row) >= 2 {
			value = strings.TrimSpace(row[1])
		}

		fmt.Print(ColorForeground + "│ " + ColorCyan)
		fmt.Print(t.formatCell(field, maxFieldWidth, AlignLeft))
		fmt.Print(" " + ColorForeground + "│ " + ColorForeground)

		valueLines := wordWrap(value, maxValueWidth)
		if len(valueLines) == 0 {
			fmt.Printf("%-*s │\n", maxValueWidth, " ")
		} else {
			firstLine := valueLines[0]
			fmt.Print(t.formatCell(firstLine, maxValueWidth, AlignLeft))
			fmt.Println(" │" + ColorReset)

			for j := 1; j < len(valueLines); j++ {
				fmt.Print(ColorForeground + "│")
				fmt.Print(repeatChar(" ", maxFieldWidth+2))
				fmt.Print("│ " + ColorForeground)
				line := valueLines[j]
				fmt.Print(t.formatCell(line, maxValueWidth, AlignLeft))
				fmt.Println(" │" + ColorReset)
			}
		}

		if i < rowCount-1 {
			fmt.Print(ColorForeground + "├")
			fmt.Print(repeatChar("─", maxFieldWidth+2))
			fmt.Print("┼")
			fmt.Print(repeatChar("─", maxValueWidth+2))
			fmt.Println("┤" + ColorReset)
		}
	}

	fmt.Print(ColorForeground + "└")
	fmt.Print(repeatChar("─", maxFieldWidth+2))
	fmt.Print("┴")
	fmt.Print(repeatChar("─", maxValueWidth+2))
	fmt.Println("┘" + ColorReset)
}

// ParseTable builds a Table from pipe-delimited strings: headers is
// "Col1|Col2|...", rows is one row per line in the same format. Convenient
// for building a table from a static []string{"a|b|c", ...} literal.
func ParseTable(headers, rows string) *Table {
	headerList := strings.Split(headers, "|")
	for i, header := range headerList {
		headerList[i] = strings.TrimSpace(header)
	}

	table := NewTable(headerList)

	if rows != "" {
		rowLines := strings.Split(strings.TrimSpace(rows), "\n")
		for _, line := range rowLines {
			if line != "" {
				cols := strings.Split(line, "|")
				for i, col := range cols {
					cols[i] = strings.TrimSpace(col)
				}
				table.AddRow(cols)
			}
		}
	}

	return table
}

// GetTerminalWidth returns stdout's current terminal width, or 80 if it
// can't be determined (e.g. stdout isn't a terminal).
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // fallback default
	}
	return width
}

// AddRow appends one row of cell values.
func (t *Table) AddRow(row []string) {
	t.Rows = append(t.Rows, row)
}

func stripColors(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func visibleWidth(s string) int {
	return utf8.RuneCountInString(stripColors(s))
}

func repeatChar(char string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(char, count)
}

func wordWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if visibleWidth(word) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			for len(word) > 0 {
				if visibleWidth(word) <= width {
					currentLine = word
					break
				}
				breakPoint := width
				if breakPoint > len(word) {
					breakPoint = len(word)
				}
				lines = append(lines, word[:breakPoint])
				word = word[breakPoint:]
			}
		} else {
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			if visibleWidth(testLine) <= width {
				currentLine = testLine
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
