package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// MultiSelectItem is anything PromptMultiSelect/PromptSingleSelect can
// list and let the user pick - a display name plus an optional
// description shown alongside it.
type MultiSelectItem interface {
	GetDisplayName() string
	GetDescription() string
}

// SimpleSelectItem is the plain-string MultiSelectItem implementation -
// what PromptMultiSelectStrings/PromptSingleSelectString build internally,
// and a reasonable default when you don't need a richer item type.
type SimpleSelectItem struct {
	Name        string
	Description string
}

func (s SimpleSelectItem) GetDisplayName() string {
	return s.Name
}

func (s SimpleSelectItem) GetDescription() string {
	return s.Description
}

// NewSimpleSelectItem builds a SimpleSelectItem.
func NewSimpleSelectItem(name, description string) SimpleSelectItem {
	return SimpleSelectItem{
		Name:        name,
		Description: description,
	}
}

// MultiSelectConfig configures a PromptMultiSelect call - see
// DefaultMultiSelectConfig for the baseline every command starts from.
type MultiSelectConfig struct {
	Title            string
	Prompt           string
	ShowNumbers      bool
	ShowDescriptions bool
	PageSize         int
	AllowEmpty       bool
}

// DefaultMultiSelectConfig returns sane defaults: numbered items with
// descriptions shown, 20 per page, empty selection allowed. Override
// individual fields on the returned value rather than building
// MultiSelectConfig{} from scratch.
func DefaultMultiSelectConfig() MultiSelectConfig {
	return MultiSelectConfig{
		Title:            "Select Items",
		Prompt:           "Select items (e.g., 1, 3-5, * for all, or leave blank)",
		ShowNumbers:      true,
		ShowDescriptions: true,
		PageSize:         20,
		AllowEmpty:       true,
	}
}

// PromptMultiSelect displays items per config and reads a selection from
// stdin: comma-separated numbers, "N-M" ranges, "*" for all, or blank
// (allowed unless config.AllowEmpty is false). Invalid input returns an
// error rather than reprompting.
func PromptMultiSelect[T MultiSelectItem](items []T, config MultiSelectConfig) ([]T, error) {
	if len(items) == 0 {
		PrintInfo("No items available for selection")
		return nil, nil
	}

	displayItemsWithStyling(items, config)
	return promptItemSelectionWithStyling(items, config)
}

// PromptMultiSelectSimple is PromptMultiSelect with DefaultMultiSelectConfig
// and just a custom title.
func PromptMultiSelectSimple[T MultiSelectItem](items []T, title string) ([]T, error) {
	config := DefaultMultiSelectConfig()
	config.Title = title
	return PromptMultiSelect(items, config)
}

// PromptMultiSelectStrings is PromptMultiSelect for plain strings (no
// descriptions), returning the selected strings directly.
func PromptMultiSelectStrings(items []string, title string) ([]string, error) {
	if len(items) == 0 {
		PrintInfo("No items available for selection")
		return nil, nil
	}

	selectItems := make([]SimpleSelectItem, len(items))
	for i, item := range items {
		selectItems[i] = NewSimpleSelectItem(item, "")
	}

	config := DefaultMultiSelectConfig()
	config.Title = title
	config.ShowDescriptions = false

	selected, err := PromptMultiSelect(selectItems, config)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(selected))
	for i, item := range selected {
		result[i] = item.GetDisplayName()
	}

	return result, nil
}

func displayItemsWithStyling[T MultiSelectItem](items []T, config MultiSelectConfig) {
	fmt.Println()
	PrintTextH1(config.Title)
	fmt.Println()

	if len(items) == 0 {
		PrintWarning("No items found")
		return
	}

	PrintItemHeader(config.Title, len(items))
	fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)

	var displayRows []string

	maxNum := len(items)
	maxBracketWidth := len(fmt.Sprintf("[%d]", maxNum))

	for i, item := range items {
		var row string

		if config.ShowNumbers {
			bracketDisplay := fmt.Sprintf("[%d]", i+1)

			spacesNeeded := maxBracketWidth - len(bracketDisplay)
			spaces := strings.Repeat(" ", spacesNeeded)

			row = fmt.Sprintf("%s│%s   %s[%s%s%d%s]%s%s  %s%s%s",
				ColorLightGray, ColorReset,
				ColorLightGray, ColorReset, ColorGreen, i+1, ColorReset, ColorLightGray, spaces,
				ColorWhite, item.GetDisplayName(), ColorReset)
		} else {
			row = fmt.Sprintf("%s│%s %s◦%s %s%s%s",
				ColorLightGray, ColorReset,
				ColorCyan, ColorReset,
				ColorWhite, item.GetDisplayName(), ColorReset)
		}

		if config.ShowDescriptions && item.GetDescription() != "" {
			row += fmt.Sprintf(" %s─ %s%s",
				ColorComment, item.GetDescription(), ColorReset)
		}

		displayRows = append(displayRows, row)
	}

	if config.PageSize > 0 && len(displayRows) > config.PageSize {
		displayWithPagination(displayRows, config.PageSize, len(items))
	} else {
		for _, row := range displayRows {
			fmt.Println(row)
		}
	}

	fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)
	PrintItemFooter()
	fmt.Println()
}

func displayWithPagination(rows []string, pageSize int, totalCount int) {
	reader := stdinReader

	for i := 0; i < len(rows); i += pageSize {
		end := i + pageSize
		if end > len(rows) {
			end = len(rows)
		}

		for j := i; j < end; j++ {
			fmt.Println(rows[j])
		}

		if end < len(rows) {
			fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)
			PrintItemFooter()
			fmt.Println()

			PrintTextNote("Showing items %d-%d of %d", i+1, end, totalCount)
			PrintOptionPrompt("Press Enter to continue or 'q' to quit", "")

			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(input)) == "q" {
				fmt.Println()
				PrintWarning("Stopping display...")
				break
			}
			fmt.Println()
			PrintItemHeader("Continued", totalCount)
			fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)
		}
	}
}

func promptItemSelectionWithStyling[T MultiSelectItem](items []T, config MultiSelectConfig) ([]T, error) {
	reader := stdinReader

	fmt.Printf("%s┌─ Selection Options%s\n", ColorLightGray, ColorReset)
	fmt.Printf("%s│%s %s◦%s Enter numbers: %s1, 3-5, 7%s\n",
		ColorLightGray, ColorReset, ColorCyan, ColorReset, ColorOrange, ColorReset)
	fmt.Printf("%s│%s %s◦%s Select all: %s*%s\n",
		ColorLightGray, ColorReset, ColorCyan, ColorReset, ColorOrange, ColorReset)
	if config.AllowEmpty {
		fmt.Printf("%s│%s %s◦%s Leave blank: %sselect none%s\n",
			ColorLightGray, ColorReset, ColorCyan, ColorReset, ColorOrange, ColorReset)
	}
	fmt.Printf("%s└─%s\n", ColorLightGray, ColorReset)
	fmt.Println()

	PrintOptionPrompt(config.Prompt, "")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		if config.AllowEmpty {
			PrintInfo("No items selected")
			return nil, nil
		}
		return nil, fmt.Errorf("selection cannot be empty")
	}

	if input == "*" {
		PrintSuccess("Selected all items (%d)", len(items))
		displaySelectedItemsWithStyling(items, config.ShowDescriptions)
		return items, nil
	}

	selected, err := parseSelectionWithValidation(input, items)
	if err != nil {
		PrintError("Selection error: %s", err.Error())
		return nil, err
	}

	if len(selected) > 0 {
		PrintSuccess("Selected %d item(s)", len(selected))
		displaySelectedItemsWithStyling(selected, config.ShowDescriptions)
	}

	return selected, nil
}

func parseSelectionWithValidation[T MultiSelectItem](input string, items []T) ([]T, error) {
	var selected []T
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s (use format: start-end)", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start of range: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end of range: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start
			}

			if start < 1 || end > len(items) {
				return nil, fmt.Errorf("range out of bounds: %d-%d (valid: 1-%d)", start, end, len(items))
			}

			for i := start; i <= end; i++ {
				selected = append(selected, items[i-1])
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}

			if num < 1 || num > len(items) {
				return nil, fmt.Errorf("number out of range: %d (valid: 1-%d)", num, len(items))
			}

			selected = append(selected, items[num-1])
		}
	}

	return removeDuplicatesWithOrder(selected), nil
}

func removeDuplicatesWithOrder[T MultiSelectItem](items []T) []T {
	seen := make(map[string]bool)
	var result []T

	for _, item := range items {
		name := item.GetDisplayName()
		if !seen[name] {
			seen[name] = true
			result = append(result, item)
		}
	}

	return result
}

func displaySelectedItemsWithStyling[T MultiSelectItem](items []T, showDescriptions bool) {
	if len(items) == 0 {
		return
	}

	fmt.Println()
	PrintDivider()

	fmt.Printf("%s✓ Selected Items %s(%d):%s\n",
		ColorGreen, ColorLightGray, len(items), ColorReset)
	maxNum := len(items)
	numWidth := len(strconv.Itoa(maxNum))

	for i, item := range items {
		fmt.Printf("  %s[%s%*d%s]%s %s%s%s",
			ColorLightGray, ColorGreen, numWidth, i+1, ColorLightGray, ColorReset,
			ColorWhite, item.GetDisplayName(), ColorReset)

		if showDescriptions && item.GetDescription() != "" {
			fmt.Printf(" %s─ %s%s", ColorComment, item.GetDescription(), ColorReset)
		}
		fmt.Println()
	}

	PrintDivider()
	fmt.Println()
}

// PromptSingleSelect displays items and reads exactly one number
// selection from stdin - unlike PromptMultiSelect, an empty or
// out-of-range entry is always an error, never "select nothing".
func PromptSingleSelect[T MultiSelectItem](items []T, title string) (T, error) {
	var zero T

	if len(items) == 0 {
		PrintWarning("No items available for selection")
		return zero, fmt.Errorf("no items available")
	}

	config := DefaultMultiSelectConfig()
	config.Title = title
	config.Prompt = "Select one item (enter number)"
	config.AllowEmpty = false

	displayItemsWithStyling(items, config)

	reader := stdinReader

	PrintOptionPrompt(config.Prompt, fmt.Sprintf("1-%d", len(items)))

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	num, err := strconv.Atoi(input)
	if err != nil {
		PrintError("Invalid selection: %s", input)
		return zero, fmt.Errorf("invalid selection: %s", input)
	}

	if num < 1 || num > len(items) {
		PrintError("Selection out of range: %d (valid: 1-%d)", num, len(items))
		return zero, fmt.Errorf("selection out of range: %d (valid: 1-%d)", num, len(items))
	}

	selected := items[num-1]

	fmt.Println()
	PrintDivider()
	fmt.Printf("%s✓ Selected:%s %s%s%s\n",
		ColorGreen, ColorReset, ColorForeground, selected.GetDisplayName(), ColorReset)
	if selected.GetDescription() != "" {
		fmt.Printf("  %s%s%s\n", ColorComment, selected.GetDescription(), ColorReset)
	}
	PrintDivider()
	fmt.Println()

	return selected, nil
}

// PromptSingleSelectString is PromptSingleSelect for plain strings,
// returning the selected string directly.
func PromptSingleSelectString(items []string, title string) (string, error) {
	if len(items) == 0 {
		PrintWarning("No items available")
		return "", fmt.Errorf("no items available")
	}

	selectItems := make([]SimpleSelectItem, len(items))
	for i, item := range items {
		selectItems[i] = NewSimpleSelectItem(item, "")
	}

	selected, err := PromptSingleSelect(selectItems, title)
	if err != nil {
		return "", err
	}

	return selected.GetDisplayName(), nil
}

// PromptConfirmSelection lists (up to 3, then "...and N more") the items
// about to be affected by action and asks for a y/N confirmation. Returns
// false immediately, with no prompt, if items is empty.
func PromptConfirmSelection[T MultiSelectItem](items []T, action string) bool {
	if len(items) == 0 {
		return false
	}

	fmt.Println()
	PrintWarning("Confirm %s for %d item(s):", action, len(items))

	for i, item := range items {
		if i < 3 {
			fmt.Printf("  %s•%s %s%s%s\n",
				ColorYellow, ColorReset, ColorForeground, item.GetDisplayName(), ColorReset)
		} else if i == 3 && len(items) > 3 {
			fmt.Printf("  %s•%s %s... and %d more%s\n",
				ColorYellow, ColorReset, ColorComment, len(items)-3, ColorReset)
			break
		}
	}

	fmt.Println()
	return ConfirmPrompt(fmt.Sprintf("Proceed with %s", action))
}

// DisplayItemSummary prints a numbered, non-interactive listing of items -
// the read-only counterpart to PromptMultiSelect's display step.
func DisplayItemSummary[T MultiSelectItem](items []T, title string) {
	if len(items) == 0 {
		PrintInfo("No %s to display", strings.ToLower(title))
		return
	}

	fmt.Println()
	PrintTextH1(fmt.Sprintf("%s Summary", title))
	fmt.Println()

	PrintItemHeader(title, len(items))
	fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)

	for i, item := range items {
		fmt.Printf("%s│%s %s%3d.%s %s%s%s",
			ColorLightGray, ColorReset,
			ColorGreen, i+1, ColorReset,
			ColorForeground, item.GetDisplayName(), ColorReset)

		if item.GetDescription() != "" {
			fmt.Printf(" %s─ %s%s", ColorComment, item.GetDescription(), ColorReset)
		}
		fmt.Println()
	}

	fmt.Printf("%s│%s\n", ColorLightGray, ColorReset)
	PrintItemFooter()
	fmt.Println()
}
