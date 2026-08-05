package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// ─────────────────────────────────────────────────────────────
//                  TOKYO NIGHT COLOR PALETTE (ANSI)
// ─────────────────────────────────────────────────────────────

// ColorXxx are raw ANSI escape codes for this package's CLI output. Prefer
// the Xxx(text) wrapper functions below (Red, Green, ...) unless you need
// the raw code itself, e.g. to build up a colored string across several
// fmt.Sprintf calls.
const (
	ColorReset      = "\033[0m"
	ColorWhite      = "\033[38;5;15m"  // FFFFFF
	ColorBlue       = "\033[38;5;69m"  // 7AA2F7
	ColorGreen      = "\033[38;5;77m"  // 9ECE6A
	ColorYellow     = "\033[38;5;186m" // E0AF68
	ColorOrange     = "\033[38;5;209m" // FF9E64
	ColorRed        = "\033[38;5;204m" // F7768E
	ColorPurple     = "\033[38;5;141m" // BB9AF7
	ColorCyan       = "\033[38;5;81m"  // 7DCFFF
	ColorGray       = "\033[38;5;246m" // 565F89
	ColorLightGray  = "\033[38;5;249m" // A9B1D6
	ColorSoftPurple = "\033[38;5;183m" // C0CAF5

	ColorBackground = "\033[48;5;24m"  // 1A1B26
	ColorForeground = "\033[38;5;252m" // C0CAF5
	ColorComment    = "\033[38;5;59m"  // 545C7E
	ColorString     = "\033[38;5;108m" // 9ECE6A
	ColorKeyword    = "\033[38;5;81m"  // 7DCFFF
	ColorFunction   = "\033[38;5;69m"  // 7AA2F7
	ColorFile       = "\033[38;5;108m" // 9ECE6A
)

// Reset, White, Green, Red, ... each wrap text in one ANSI color plus a
// trailing reset code, for coloring individual words/phrases inline
// without leaking color into whatever follows.
func Reset(text string) string {
	return fmt.Sprintf("%s%s%s", ColorReset, text, ColorReset)
}

func White(text string) string {
	return fmt.Sprintf("%s%s%s", ColorWhite, text, ColorReset)
}

func Green(text string) string {
	return fmt.Sprintf("%s%s%s", ColorGreen, text, ColorReset)
}

func Red(text string) string {
	return fmt.Sprintf("%s%s%s", ColorRed, text, ColorReset)
}

func Yellow(text string) string {
	return fmt.Sprintf("%s%s%s", ColorYellow, text, ColorReset)
}

func Blue(text string) string {
	return fmt.Sprintf("%s%s%s", ColorBlue, text, ColorReset)
}

func Orange(text string) string {
	return fmt.Sprintf("%s%s%s", ColorOrange, text, ColorReset)
}

func Purple(text string) string {
	return fmt.Sprintf("%s%s%s", ColorPurple, text, ColorReset)
}

func Cyan(text string) string {
	return fmt.Sprintf("%s%s%s", ColorCyan, text, ColorReset)
}

func Gray(text string) string {
	return fmt.Sprintf("%s%s%s", ColorGray, text, ColorReset)
}

func LightGray(text string) string {
	return fmt.Sprintf("%s%s%s", ColorLightGray, text, ColorReset)
}

func SoftPurple(text string) string {
	return fmt.Sprintf("%s%s%s", ColorSoftPurple, text, ColorReset)
}

func Comment(text string) string {
	return fmt.Sprintf("%s%s%s", ColorComment, text, ColorReset)
}

func StringColor(text string) string {
	return fmt.Sprintf("%s%s%s", ColorString, text, ColorReset)
}

func Keyword(text string) string {
	return fmt.Sprintf("%s%s%s", ColorKeyword, text, ColorReset)
}

func Function(text string) string {
	return fmt.Sprintf("%s%s%s", ColorFunction, text, ColorReset)
}

func File(text string) string {
	return fmt.Sprintf("%s%s%s", ColorFile, text, ColorReset)
}

// PrintSuccess prints a green checkmark line, Printf-style.
func PrintSuccess(format string, a ...any) {
	fmt.Printf("%s✓ %s%s%s\n", ColorGreen, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintError prints a red cross-mark line, Printf-style. Terminal output
// only - it does not itself return or wrap a Go error.
func PrintError(format string, a ...any) {
	fmt.Printf("%s✗ %s%s%s\n", ColorRed, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintWarning prints a yellow warning-triangle line, Printf-style.
func PrintWarning(format string, a ...any) {
	fmt.Printf("%s⚠ %s%s%s\n", ColorYellow, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintInfo prints a cyan info-marker line, Printf-style.
func PrintInfo(format string, a ...any) {
	fmt.Printf("%s𝕚 %s%s%s\n", ColorCyan, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintDebug prints a plain foreground-colored line, Printf-style, with no
// marker glyph.
func PrintDebug(format string, a ...any) {
	fmt.Printf("%s%s%s\n", ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintNormal prints a cyan-then-foreground line, Printf-style - used for
// body text that should stand out slightly from PrintDebug's plain output.
func PrintNormal(format string, a ...any) {
	fmt.Printf("%s%s%s%s\n", ColorCyan, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintReset prints a line with no color applied, Printf-style.
func PrintReset(format string, a ...any) {
	fmt.Printf("%s%s%s\n", ColorReset, fmt.Sprintf(format, a...), ColorReset)
}

// PrintColoredNumFileDesc prints one "N. name: description" line, used by
// migration/seeder listings.
func PrintColoredNumFileDesc(number int, name string, description string) {
	fmt.Printf(
		"%s%2d.%s %s%s%s: %s%s%s\n",
		ColorForeground, number, ColorReset,
		ColorFile, name, ColorReset,
		ColorComment, description, ColorReset,
	)
}

// PrintTextNote prints a lightbulb-marked note line, Printf-style.
func PrintTextNote(format string, a ...any) {
	fmt.Printf("%s💡 %s%s%s\n", ColorPurple, ColorForeground, fmt.Sprintf(format, a...), ColorReset)
}

// PrintWCmdInfo, PrintWCmdSuccess, PrintWCmdWarning, PrintWCmdError, and
// PrintWCmdDebug are the PrintInfo/PrintSuccess/... family with a
// "[command]" tag prefixed - use these when the message needs to identify
// which subcommand/step it came from (e.g. per-seeder output in a batch
// run).
func PrintWCmdInfo(command string, format string, a ...any) {
	fmt.Printf("%s[%s] %s%s%s\n", ColorForeground, command, ColorCyan, fmt.Sprintf(format, a...), ColorReset)
}

func PrintWCmdSuccess(command string, format string, a ...any) {
	fmt.Printf("%s✓ %s[%s] %s%s%s\n", ColorGreen, ColorForeground, command, ColorGreen, fmt.Sprintf(format, a...), ColorReset)
}

func PrintWCmdWarning(command string, format string, a ...any) {
	fmt.Printf("%s⚠ %s[%s] %s%s%s\n", ColorYellow, ColorForeground, command, ColorYellow, fmt.Sprintf(format, a...), ColorReset)
}

func PrintWCmdError(command string, format string, a ...any) {
	fmt.Printf("%s✗ %s[%s] %s%s%s\n", ColorRed, ColorForeground, command, ColorRed, fmt.Sprintf(format, a...), ColorReset)
}

func PrintWCmdDebug(command string, format string, a ...any) {
	fmt.Printf("%sDEBUG: %s[%s] %s%s\n", ColorComment, ColorForeground, command, fmt.Sprintf(format, a...), ColorReset)
}

// PrintOption prints one "[N] label" line of a numbered selection menu.
func PrintOption(index int, label string) {
	fmt.Printf("  %s[%s%d%s]%s %s%s%s\n",
		ColorLightGray,    // opening bracket color
		ColorGreen, index, // number color
		ColorLightGray, // closing bracket color
		ColorReset,
		ColorReset, label, // label color
		ColorReset)
}

// PrintOptionPrompt prints a "› label (hint): " prompt line, without
// reading input itself - pair with a bufio.Reader at the call site.
func PrintOptionPrompt(label string, hintText string) {
	fmt.Printf("%s›%s %s%s", ColorYellow, ColorReset, ColorForeground, label)

	if hintText != "" {
		fmt.Printf(" %s(%s%s%s)%s", ColorReset, ColorOrange, hintText, ColorReset, ColorReset)
	}

	fmt.Printf("%s: ", ColorLightGray)
}

// PrintInputPrompt is PrintOptionPrompt with slightly different spacing
// around the hint - used ahead of free-text input rather than a numbered
// selection.
func PrintInputPrompt(label string, hintText string) {
	fmt.Printf("%s›%s %s%s%s", ColorYellow, ColorReset, ColorForeground, label, ColorReset)
	if hintText != "" {
		fmt.Printf(" %s(%s)%s", ColorOrange, hintText, ColorReset)
	}
	fmt.Print(": ")
}

// ConfirmPrompt asks a y/N question on stdin, defaulting to false (only
// "y"/"yes", case-insensitive, count as confirmed).
func ConfirmPrompt(question string) bool {
	PrintInputPrompt(question, "y/N")

	reader := stdinReader
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

// PromptInput prints label/hint and reads one line of free-text input from
// stdin, trimmed of surrounding whitespace.
func PromptInput(label string, hint string) string {
	PrintInputPrompt(label, hint)

	reader := stdinReader
	input, _ := reader.ReadString('\n')

	return strings.TrimSpace(input)
}

// PromptNumberSelection lists options as a numbered menu and reads a
// selection from stdin - accepts the option's number, its exact text
// (case-insensitive), or nothing (returns defaultValue). Reprompts on an
// invalid entry.
func PromptNumberSelection(label string, options []string, defaultValue string) string {
	PrintTextH1(label)
	fmt.Println()
	for i, option := range options {
		fmt.Printf("  %s[%s%d%s]%s %s%s%s\n",
			ColorLightGray, ColorGreen, i+1, ColorLightGray,
			ColorReset, ColorReset, option, ColorReset)
	}

	fmt.Println()
	prompt := fmt.Sprintf("Select an option %s(1-%d)%s", ColorOrange, len(options), ColorReset)
	if defaultValue != "" {
		prompt += fmt.Sprintf(" default: [%s%s%s]", ColorGreen, defaultValue, ColorReset)
	}

	for {
		input := PromptInput(prompt, "")

		if input == "" && defaultValue != "" {
			if num, err := strconv.Atoi(defaultValue); err == nil && num > 0 && num <= len(options) {
				return options[num-1]
			}
			return defaultValue
		}

		if num, err := strconv.Atoi(input); err == nil {
			if num > 0 && num <= len(options) {
				return options[num-1]
			}
			fmt.Printf("%sInvalid selection. Please enter a number between 1 and %d%s\n",
				ColorRed, len(options), ColorReset)
			continue
		}

		for _, option := range options {
			if strings.EqualFold(option, input) {
				return option
			}
		}

		fmt.Printf("%sInvalid selection. Please try again.%s\n", ColorRed, ColorReset)
	}
}

// PromptMultiLineInput reads lines from stdin until two consecutive blank
// lines (or EOF), and joins them with newlines.
func PromptMultiLineInput(prompt string) string {
	fmt.Printf("%s%s%s\n", ColorCyan, prompt, ColorReset)
	fmt.Printf("%s(Press Enter twice to finish)%s\n", ColorComment, ColorReset)

	var lines []string
	reader := stdinReader
	emptyCount := 0

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		if line == "" {
			emptyCount++
			if emptyCount >= 2 {
				break
			}
			continue
		}
		emptyCount = 0

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
