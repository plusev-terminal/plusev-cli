package prompt

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// Option is a selectable choice for Select.
type Option struct {
	Value string
	Label string
}

// Text prompts for a single-line string value.
func Text(title, placeholder string) (string, error) {
	var v string

	input := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&v)

	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		return "", err
	}

	return v, nil
}

// RequiredText prompts for a non-empty string value.
func RequiredText(title, placeholder string) (string, error) {
	var v string

	input := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&v).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("required")
			}

			return nil
		})

	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		return "", err
	}

	return v, nil
}

// Password prompts for a masked secret value.
func Password(title string) (string, error) {
	var v string

	input := huh.NewInput().
		Title(title).
		Value(&v).
		EchoMode(huh.EchoModePassword).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("required")
			}

			return nil
		})

	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		return "", err
	}

	return v, nil
}

// Select prompts the user to pick one option from a list.
func Select(title string, opts []Option) (Option, error) {
	if len(opts) == 0 {
		return Option{}, fmt.Errorf("no options available")
	}

	var selected *Option

	huhOpts := make([]huh.Option[*Option], 0, len(opts))

	for i := range opts {
		o := opts[i]
		huhOpts = append(huhOpts, huh.NewOption(o.Label, &o))
	}

	sel := huh.NewSelect[*Option]().
		Title(title).
		Options(huhOpts...).
		Value(&selected)

	if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
		return Option{}, err
	}

	if selected == nil {
		return Option{}, fmt.Errorf("nothing selected")
	}

	return *selected, nil
}

// Confirm asks a yes/no question.
func Confirm(title string, defaultValue bool) (bool, error) {
	v := defaultValue

	confirm := huh.NewConfirm().
		Title(title).
		Value(&v)

	if err := huh.NewForm(huh.NewGroup(confirm)).Run(); err != nil {
		return false, err
	}

	return v, nil
}
