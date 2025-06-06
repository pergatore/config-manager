package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Find unmanaged dotfiles in home directory
func findUnmanagedDotfiles(config *Config) []string {
	homeDir, _ := os.UserHomeDir()
	var unmanaged []string
	
	// Get list of currently managed files (check both name and target path)
	managed := make(map[string]bool)
	managedPaths := make(map[string]bool)
	for _, file := range config.Files {
		managed[file.Name] = true
		managedPaths[file.Target] = true
	}
	
	// Check common dotfiles
	commonDotfiles := []string{
		".zshrc", ".bashrc", ".bash_profile", ".profile", ".bash_aliases",
		".gitconfig", ".gitignore_global", ".gitmessage",
		".vimrc", ".inputrc", ".tmux.conf", ".screenrc",
		".xinitrc", ".xprofile", ".Xresources", ".Xdefaults",
		".editorconfig", ".prettierrc", ".eslintrc",
	}
	
	for _, dotfile := range commonDotfiles {
		targetPath := filepath.Join(homeDir, dotfile)
		if !managed[dotfile] && !managedPaths[targetPath] {
			if _, err := os.Stat(targetPath); err == nil {
				unmanaged = append(unmanaged, dotfile)
			}
		}
	}
	
	// Also scan for other dotfiles in home directory
	if entries, err := os.ReadDir(homeDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			targetPath := filepath.Join(homeDir, name)
			if strings.HasPrefix(name, ".") && !entry.IsDir() && 
			   !managed[name] && !managedPaths[targetPath] {
				// Skip common non-config files
				if !isSystemFile(name) {
					unmanaged = append(unmanaged, name)
				}
			}
		}
	}
	
	return unmanaged
}

// Discover all possible configuration files and directories
func discoverAllConfigs() []string {
	homeDir, _ := os.UserHomeDir()
	var configs []string
	
	fmt.Printf("Scanning home directory: %s\n", homeDir)
	
	// Common dotfiles in home directory
	commonDotfiles := []string{
		".zshrc", ".bashrc", ".bash_profile", ".profile", ".bash_aliases",
		".gitconfig", ".gitignore_global", ".gitmessage",
		".vimrc", ".inputrc", ".tmux.conf", ".screenrc",
		".xinitrc", ".xprofile", ".Xresources", ".Xdefaults",
		".editorconfig", ".prettierrc", ".eslintrc",
	}
	
	fmt.Print("Checking common dotfiles... ")
	found := 0
	for _, dotfile := range commonDotfiles {
		path := filepath.Join(homeDir, dotfile)
		if _, err := os.Stat(path); err == nil {
			configs = append(configs, fmt.Sprintf("%s (file)", dotfile))
			found++
		}
	}
	fmt.Printf("found %d\n", found)
	
	// Check .config directory for subdirectories
	configDir := filepath.Join(homeDir, ".config")
	fmt.Printf("Checking .config directory: %s... ", configDir)
	configFound := 0
	if entries, err := os.ReadDir(configDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				// Skip some system directories
				name := entry.Name()
				if !isSystemConfigDir(name) {
					configs = append(configs, fmt.Sprintf(".config/%s (directory)", name))
					configFound++
				}
			}
		}
		fmt.Printf("found %d directories\n", configFound)
	} else {
		fmt.Printf("not accessible (%v)\n", err)
	}
	
	// Other important directories
	specialDirs := []string{
		".ssh", ".gnupg", ".local/bin", ".local/share/applications",
		".fonts", ".themes", ".icons",
	}
	
	fmt.Print("Checking special directories... ")
	specialFound := 0
	for _, dir := range specialDirs {
		path := filepath.Join(homeDir, dir)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			configs = append(configs, fmt.Sprintf("%s (directory)", dir))
			specialFound++
		}
	}
	fmt.Printf("found %d\n", specialFound)
	
	fmt.Printf("Total configurations discovered: %d\n", len(configs))
	
	// Debug: print first few found configs
	if len(configs) > 0 {
		fmt.Println("Examples found:")
		for i, config := range configs {
			if i >= 5 { // Only show first 5
				fmt.Printf("  ... and %d more\n", len(configs)-5)
				break
			}
			fmt.Printf("  - %s\n", config)
		}
	} else {
		fmt.Println("❌ No configurations found!")
		fmt.Println("This might be because:")
		fmt.Println("  - You don't have common dotfiles yet")
		fmt.Println("  - Your configs are in non-standard locations")
		fmt.Println("  - Permission issues accessing directories")
		fmt.Printf("  - Home directory: %s\n", homeDir)
		
		// Let's check what's actually in the home directory
		if entries, err := os.ReadDir(homeDir); err == nil {
			fmt.Println("Files in home directory:")
			for i, entry := range entries {
				if i >= 10 { // Only show first 10
					fmt.Printf("  ... and %d more\n", len(entries)-10)
					break
				}
				if strings.HasPrefix(entry.Name(), ".") {
					fmt.Printf("  %s\n", entry.Name())
				}
			}
		}
	}
	
	return configs
}

// Check if a config directory should be skipped (system directories)
func isSystemConfigDir(name string) bool {
	systemDirs := []string{
		"BraveSoftware", "google-chrome", "chromium", "firefox",
		"pulse", "systemd", "dconf", "gconf", "ibus", "fontconfig",
		"gtk-2.0", "gtk-3.0", "gtk-4.0", "qt5ct", "qt6ct",
		"menus", "mimeapps.list", "user-dirs.dirs", "user-dirs.locale",
		"evolution", "libreoffice", "calibre", "vlc", "mpv",
		"dbus-1", "autostart", "enchant", "epiphany", "gedit",
	}
	
	for _, sysDir := range systemDirs {
		if name == sysDir {
			return true
		}
	}
	
	return false
}
