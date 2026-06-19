package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Bold(true)
)

// Success prints a green success line to stdout.
func Success(msg string) {
	fmt.Println(successStyle.Render("✓ " + msg))
}

// Error prints a red error line to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✖ "+msg))
}

// Info prints a blue informational line to stdout.
func Info(msg string) {
	fmt.Println(infoStyle.Render(msg))
}

// KV prints a labelled value with the label dimmed.
func KV(label, value string) {
	fmt.Printf("%s %s\n", labelStyle.Render(label+":"), value)
}

// JSON pretty-prints any value as indented JSON.
func JSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(v)
		return
	}

	fmt.Println(string(b))
}

// Table renders a compact borderless table to stdout.
func Table(header []string, rows [][]string) {
	t := tablewriter.NewWriter(os.Stdout)

	t.SetHeader(header)
	t.SetAutoWrapText(false)
	t.SetAutoFormatHeaders(true)
	t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetCenterSeparator("")
	t.SetColumnSeparator("")
	t.SetRowSeparator("")
	t.SetHeaderLine(false)
	t.SetBorder(false)
	t.SetTablePadding("\t")
	t.SetNoWhiteSpace(true)
	t.AppendBulk(rows)
	t.Render()
}

// Dim returns a dimmed version of the string.
func Dim(s string) string {
	return dimStyle.Render(s)
}

// JoinList joins string slices with ", " and renders empty as a dim dash.
func JoinList(items []string) string {
	if len(items) == 0 {
		return Dim("—")
	}

	return strings.Join(items, ", ")
}
