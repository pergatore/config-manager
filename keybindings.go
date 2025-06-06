package main

import "github.com/charmbracelet/bubbles/key"

// Key bindings
type keyMap struct {
	Enter   key.Binding
	Add     key.Binding
	Remove  key.Binding
	Link    key.Binding
	LinkAll key.Binding
	Edit    key.Binding
	Backup  key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Add, k.Link, k.Edit, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Enter, k.Add, k.Remove, k.Edit},
		{k.Link, k.LinkAll, k.Backup, k.Quit},
	}
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add file"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove"),
	),
	Link: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "link selected"),
	),
	LinkAll: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "link all"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Backup: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "backup configs"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
