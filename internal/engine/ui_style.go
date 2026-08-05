package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FancyProgressBar blocks for duration, animating a spinner next to
// action, then prints a checkmark. Purely cosmetic - it does not track
// real progress of anything, just fills wall-clock time.
func FancyProgressBar(action string, duration time.Duration) {
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	colors := []string{ColorBlue, ColorPurple, ColorCyan, ColorGreen}

	fmt.Printf("%s→ %s... %s", ColorComment, action, ColorReset)

	start := time.Now()
	i := 0

	for time.Since(start) < duration {
		fmt.Printf("\r%s→ %s... %s%s%s ", ColorComment, action, colors[i%len(colors)], spin[i%len(spin)], ColorReset)
		time.Sleep(100 * time.Millisecond)
		i++
	}
	fmt.Printf("\r%s→ %s... %s✓%s\n", ColorComment, action, ColorGreen, ColorReset)
}

// FancyProgressBarContext animates a spinner next to action until ctx is
// cancelled, then prints a checkmark plus completeText and the elapsed
// duration. Meant to run in its own goroutine alongside the real work that
// cancels ctx when done (see RunSingleSeeder for the pattern).
func FancyProgressBarContext(ctx context.Context, action string, completeText string) {
	spin := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	colors := []string{ColorBlue, ColorPurple, ColorCyan, ColorGreen}

	start := time.Now()

	fmt.Printf("%s→ %s... %s", ColorComment, action, ColorReset)

	i := 0
	for {
		select {
		case <-ctx.Done():
			duration := time.Since(start).Round(time.Millisecond)

			completionMsg := fmt.Sprintf("\r%s→ %s... %s✓%s", ColorComment, action, ColorGreen, ColorReset)

			if completeText != "" {
				completionMsg += fmt.Sprintf(" %s%s%s", ColorComment, completeText, ColorReset)
			}

			completionMsg += fmt.Sprintf(" %s(%s)%s\n", ColorComment, duration, ColorReset)

			fmt.Print(completionMsg)
			return
		default:
			fmt.Printf("\r%s→ %s... %s%s%s ", ColorComment, action, colors[i%len(colors)], spin[i%len(spin)], ColorReset)
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}

// PrintSectionHeader prints title centered inside an 80-column horizontal
// rule, wrapping onto multiple centered lines if title itself is long.
func PrintSectionHeader(title string) {
	const maxWidth = 80
	separator := "─"

	lines := wrapText(stripANSI(title), maxWidth-4)
	fmt.Println()

	for _, line := range lines {
		padding := maxWidth - len(line) - 4
		left := strings.Repeat(separator, padding/2)
		right := strings.Repeat(separator, padding-padding/2)
		fmt.Printf("%s%s  %s%s  %s%s\n", ColorPurple, left, ColorBlue, line, ColorPurple, right+ColorReset)
	}
	fmt.Println()
}

// PrintThinCard prints text centered inside a single-line-bordered box
// (╭─╮/╰─╯). The variadic color parameter is accepted for call-site
// compatibility but unused - box color is always ColorPurple.
func PrintThinCard(text string, color ...string) {
	const width = 80
	contentWidth := width - 2

	lines := wrapText(text, contentWidth-8)

	maxVisibleLen := 0
	for _, line := range lines {
		if l := visibleLength(line); l > maxVisibleLen {
			maxVisibleLen = l
		}
	}

	padding := (contentWidth - maxVisibleLen) / 2
	if padding < 0 {
		padding = 0
	}

	fmt.Printf("%s╭%s╮%s\n", ColorPurple, strings.Repeat("─", width-2), ColorReset)

	for _, line := range lines {
		visibleLen := visibleLength(line)
		linePadding := padding + (maxVisibleLen-visibleLen)/2
		rightPadding := contentWidth - visibleLen - linePadding

		fmt.Printf("%s│%s%s%s%s%s│%s\n",
			ColorPurple,
			strings.Repeat(" ", linePadding),
			ColorFunction, line, ColorPurple,
			strings.Repeat(" ", rightPadding),
			ColorReset,
		)
	}

	fmt.Printf("%s╰%s╯%s\n", ColorPurple, strings.Repeat("─", width-2), ColorReset)
}

// PrintBoldCard is PrintThinCard with heavier box-drawing characters
// (┏━┓/┗━┛) - used for the "APPLICATION COMMANDS:BUILD"-style banners each
// command constructor prints on startup.
func PrintBoldCard(text string) {
	const width = 80
	contentWidth := width - 2

	lines := wrapText(text, contentWidth-8)

	maxVisibleLen := 0
	for _, line := range lines {
		if l := visibleLength(line); l > maxVisibleLen {
			maxVisibleLen = l
		}
	}

	padding := (contentWidth - maxVisibleLen) / 2
	if padding < 0 {
		padding = 0
	}

	fmt.Printf("%s┏%s┓%s\n", ColorPurple, strings.Repeat("━", width-2), ColorReset)

	for _, line := range lines {
		visibleLen := visibleLength(line)
		linePadding := padding + (maxVisibleLen-visibleLen)/2
		rightPadding := contentWidth - visibleLen - linePadding

		fmt.Printf("%s┃%s%s%s%s%s┃%s\n",
			ColorPurple,
			strings.Repeat(" ", linePadding),
			ColorFunction, line, ColorPurple,
			strings.Repeat(" ", rightPadding),
			ColorReset,
		)
	}

	fmt.Printf("%s┗%s┛%s\n", ColorPurple, strings.Repeat("━", width-2), ColorReset)
}

// PrintTextH1 prints text as a "┃ " prefixed sub-heading line.
func PrintTextH1(text string) {
	fmt.Printf("%s%s%s\n", ColorBlue, "┃ ", text)
	fmt.Print(ColorReset)
}

// PrintItemHeader opens a "┌─ title (N items)" block, paired with
// PrintItemFooter.
func PrintItemHeader(title string, count int) {
	fmt.Printf("%s┌─ %s%s (%d items)%s%s\n",
		ColorLightGray, ColorPurple, title, count, ColorLightGray, ColorReset)
}

// PrintItemFooter closes a block opened by PrintItemHeader.
func PrintItemFooter() {
	fmt.Printf("%s└─%s\n", ColorLightGray, ColorReset)
}

// PrintCode prints code as plain indented lines (no border) - see
// PrintCodeBlock for the bordered version with an optional title.
func PrintCode(code string) {
	lines := strings.Split(strings.TrimSpace(code), "\n")

	fmt.Println()
	for _, line := range lines {
		fmt.Printf("%s  %s%s\n", ColorComment, ColorForeground, line)
	}
	fmt.Println()
}

// PrintCodeBlock prints code inside a bordered box (╭─╮/╰─╯), sized to
// its longest line, with an optional uppercased title printed above it.
func PrintCodeBlock(code string, title ...string) {
	lines := strings.Split(strings.TrimSpace(code), "\n")

	maxLen := 0
	for _, line := range lines {
		if l := visibleLength(line); l > maxLen {
			maxLen = l
		}
	}

	if len(title) > 0 {
		clean := strings.ToUpper(title[0])
		fmt.Printf("\n%s↳ %s%s\n", ColorGreen, clean, ColorReset)
		fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("·", visibleLength("↳ "+clean)), ColorReset)
	}

	fmt.Printf("%s╭%s%s%s╮\n", ColorComment, ColorGray, strings.Repeat("─", maxLen+2), ColorComment)

	for _, line := range lines {
		padding := maxLen - visibleLength(line)
		fmt.Printf("%s│%s %s%s%s%s │\n", ColorComment, ColorForeground, line, strings.Repeat(" ", padding), ColorReset, ColorComment)
	}

	fmt.Printf("%s╰%s%s%s╯\n", ColorComment, ColorGray, strings.Repeat("─", maxLen+2), ColorComment)
}

// PrintDivider prints an 80-column horizontal rule.
func PrintDivider() {
	fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 80), ColorReset)
}

// PrintKeyValueTable prints data as aligned two-column rows. Each entry is
// "key|value" (see splitUnescapedPipe for how to include a literal "|" in
// either side); long keys/values word-wrap within their column rather than
// truncating.
func PrintKeyValueTable(data []string, keyColor, valueColor string) {
	totalWidth := 80

	keyWidth := int(float64(totalWidth) * 0.5)
	valWidth := totalWidth - keyWidth

	for _, item := range data {
		key, val := splitUnescapedPipe(item)

		keyLines := wrapText(key, keyWidth)
		valLines := wrapText(val, valWidth)
		maxLines := max(len(keyLines), len(valLines))

		for i := 0; i < maxLines; i++ {
			k := ""
			v := ""
			if i < len(keyLines) {
				k = keyLines[i]
			}
			if i < len(valLines) {
				v = valLines[i]
			}
			fmt.Printf("%s%-*s%s %-*s%s\n", keyColor, keyWidth, k, valueColor, valWidth, v, ColorReset)
		}
	}
}

// PrintKeyValueTableWithHyperlink is PrintKeyValueTable, but a key
// containing "\a" is split into actualKey\adisplayKey and rendered as an
// OSC-8 terminal hyperlink (actualKey as the target, displayKey as the
// visible text) in terminals that support it.
func PrintKeyValueTableWithHyperlink(data []string, keyColor, valueColor string) {
	const keyWidth, valWidth = 60, 40

	for _, item := range data {
		keyRaw, val := splitUnescapedPipe(item)

		displayKey := keyRaw
		actualKey := ""
		useHyperlink := false

		if strings.Contains(keyRaw, "\a") {
			split := strings.SplitN(keyRaw, "\a", 2)
			actualKey = split[0]
			displayKey = split[1]
			useHyperlink = true
		}

		keyLines := wrapText(displayKey, keyWidth)
		valLines := wrapText(val, valWidth)
		maxLines := max(len(keyLines), len(valLines))

		for i := 0; i < maxLines; i++ {
			k := ""
			v := ""
			if i < len(keyLines) {
				k = keyLines[i]
			}
			if i < len(valLines) {
				v = valLines[i]
			}

			printKey := k
			if useHyperlink && i == 0 {
				printKey = fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", actualKey, k)
			}

			space := keyWidth - visibleLength(k)
			if space < 0 {
				space = 0
			}

			fmt.Printf("%s%s%s%s%s%s\n",
				keyColor,
				printKey,
				strings.Repeat(" ", space),
				valueColor,
				v,
				ColorReset,
			)
		}
	}
}

func visibleLength(s string) int {
	// Strip ANSI color codes
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mK]`)
	clean := ansiRegex.ReplaceAllString(s, "")
	return len(clean)
}

// Helper function to strip ANSI codes
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mK]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// Helper function to wrap text while preserving ANSI codes
func wrapText(text string, maxWidth int) []string {
	var result []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	line := words[0]
	for _, word := range words[1:] {
		if visibleLength(line+" "+word) <= maxWidth {
			line += " " + word
		} else {
			result = append(result, line)
			line = word
		}
	}
	result = append(result, line)
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func splitUnescapedPipe(input string) (string, string) {
	inEscape := false
	for i := 0; i < len(input); i++ {
		if input[i] == '\\' {
			inEscape = !inEscape
			continue
		}
		if input[i] == '|' && !inEscape {
			key := strings.ReplaceAll(input[:i], `\|`, "|")
			val := strings.ReplaceAll(input[i+1:], `\|`, "|")
			return key, val
		}
		inEscape = false
	}
	return strings.ReplaceAll(input, `\|`, "|"), ""
}

// VisibleLength returns s's length with ANSI color/erase codes stripped -
// what you actually want for column-width math on colored strings.
func VisibleLength(s string) int {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mK]`)
	clean := ansiRegex.ReplaceAllString(s, "")
	return len(clean)
}

// PrintCommandList prints a simple numbered "name: description" list,
// name-padded to align the descriptions.
func PrintCommandList(commands []struct {
	Name        string
	Description string
}) {
	PrintTextH1("Available Commands")
	fmt.Println()

	maxNameLen := 0
	for _, cmd := range commands {
		if l := VisibleLength(cmd.Name); l > maxNameLen {
			maxNameLen = l
		}
	}

	for i, cmd := range commands {
		paddedName := fmt.Sprintf("%-*s", maxNameLen, cmd.Name)
		fmt.Printf(" %s%2d.%s %s%s%s: %s%s%s\n",
			ColorLightGray, i+1, ColorReset,
			ColorFile, paddedName, ColorReset,
			ColorComment, cmd.Description, ColorReset,
		)
	}
}
