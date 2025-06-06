package main

import (
	"github.com/charmbracelet/bubbles/list"
)

// Data structures
type ConfigFile struct {
	Name        string            `json:"name"`
	Source      string            `json:"source"`      // Path in dotfiles repo
	Target      string            `json:"target"`      // Path where it should be linked
	Category    string            `json:"category"`
	Template    bool              `json:"template"`
	Variables   map[string]string `json:"variables,omitempty"`
	IsLinked    bool              `json:"-"`
	HasConflict bool              `json:"-"`
}

type Config struct {
	Files            []ConfigFile      `json:"files"`
	ConfigDir        string            `json:"config_dir"`
	DotfilesDir      string            `json:"dotfiles_dir"`
	Variables        map[string]string `json:"global_variables"`
	Categories       []string          `json:"categories"`
	TemplateExts     []string          `json:"template_extensions"`
	Editor           string            `json:"editor"`
	Shell            string            `json:"shell"`
}

// Application state
type model struct {
	config       *Config
	currentView  string
	fileList     list.Model
	selectedFile *ConfigFile
	message      string
	messageType  string // "success", "error", "warning"
	width        int
	height       int
}

// List items for bubbles/list
type fileItem struct {
	file ConfigFile
}
