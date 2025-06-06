# Config Manager

A beautiful, interactive TUI application for managing your dotfiles and configuration files across multiple machines.

![Config Manager](https://img.shields.io/badge/Go-1.21+-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

## Features

- 🎨 **Beautiful TUI** - Clean, modern interface built with Charmbracelet Bubble Tea
- 📁 **Smart Discovery** - Automatically finds your dotfiles and config directories
- 🔗 **Symlink Management** - Creates and manages symlinks to keep configs in sync
- 📝 **Template Support** - Dynamic templates with variables for different environments
- 🔄 **Easy Sync** - Move your entire configuration between machines effortlessly
- ⚙️ **Smart Editor Integration** - Edit files and directories with your preferred editor
- 🗂️ **Category Organization** - Organize configs by type (shell, editor, git, etc.)
- 💾 **Backup System** - Automatic backups before making changes
- 🎯 **Selective Linking** - Link individual files or all at once

## Installation

### Prerequisites

- Go 1.21 or later
- [Gum](https://github.com/charmbracelet/gum) (optional, for enhanced UI)

### Install Gum (Recommended)

```bash
# macOS
brew install gum

# Ubuntu/Debian
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install gum

# Arch Linux
pacman -S gum
```

### Install Config Manager

```bash
go install github.com/pergatore/config-manager@latest
```

Or build from source:

```bash
git clone https://github.com/pergatore/config-manager.git
cd config-manager
go build -o config-manager
sudo mv config-manager /usr/local/bin/
```

## Quick Start

1. **Run the application**:
   ```bash
   config-manager
   ```

2. **First-time setup**: The setup wizard will guide you through:
   - Choosing your preferred editor (vim, nvim, VS Code, etc.)
   - Selecting your shell (bash, zsh, fish)
   - Discovering existing configuration files

3. **Add configurations**: Press `a` to add dotfiles and config directories

4. **Link configurations**: Press `l` to link a specific file, or `L` to link all

## Usage

### Key Bindings

- **`a`** - Add new configuration file or directory
- **`r`** - Remove configuration from management
- **`e`** - Edit configuration file (supports directories)
- **`l`** - Link selected configuration
- **`L`** - Link all configurations
- **`b`** - Create backup of current configurations
- **`q`** - Quit application

### Status Indicators

- **✓** - Configuration is properly linked
- **✗** - Configuration is not linked
- **⚠️** - Configuration has conflicts (file exists but isn't linked)

## Moving Configurations Between Machines

Config Manager makes it easy to sync your dotfiles across multiple computers.

### Method 1: Git Repository (Recommended)

1. **On your primary machine**, initialize a git repository for your entire config-manager directory:
   ```bash
   cd ~/.config/config-manager
   git init
   git add .
   git commit -m "Initial config-manager setup"
   git remote add origin https://github.com/yourusername/config-manager-dotfiles.git
   git push -u origin main
   ```

2. **On a new machine**, install config-manager and clone your complete setup:
   ```bash
   # Install config-manager first
   go install github.com/pergatore/config-manager@latest
   
   # Remove any existing config-manager directory
   rm -rf ~/.config/config-manager
   
   # Clone your complete configuration
   git clone https://github.com/yourusername/config-manager-dotfiles.git ~/.config/config-manager
   
   # Run config-manager and link everything
   config-manager
   # Press 'L' to link all configurations
   ```

   **That's it!** Your new machine now has:
   - All your dotfiles and configurations
   - Your config.json settings
   - Your custom templates
   - Your backup history
   - All categories and preferences

3. **Keep machines in sync**:
   ```bash
   # After making changes on any machine
   cd ~/.config/config-manager
   git add .
   git commit -m "Update configurations"
   git push
   
   # On other machines
   cd ~/.config/config-manager
   git pull
   config-manager  # Re-link if needed
   ```

### Method 2: Manual Sync

1. **Export your complete configuration**:
   ```bash
   tar -czf my-complete-config.tar.gz -C ~/.config config-manager
   ```

2. **On the new machine**:
   ```bash
   # Install config-manager first
   go install github.com/pergatore/config-manager@latest
   
   # Extract your complete setup
   tar -xzf my-complete-config.tar.gz -C ~/.config
   
   # Link everything
   config-manager
   # Press 'L' to link all configurations
   ```

### Why This Approach is Better

By making the entire `~/.config/config-manager` directory your repository, you get:

- ✅ **Complete portability** - Everything moves together
- ✅ **No manual config copying** - config.json is included
- ✅ **Template preservation** - All your custom templates travel with you
- ✅ **Backup history** - Your backup history is preserved (optional)
- ✅ **Category settings** - Custom categories and preferences included
- ✅ **One-command setup** - Clone and you're ready to go

## Working with Templates

Config Manager supports dynamic templates that can adapt to different environments.

### Creating Templates

1. **Create a template directory**:
   ```bash
   mkdir -p ~/.config/config-manager/templates
   ```

2. **Create template files** with `.tmpl`, `.template`, or `.tpl` extensions:
   ```bash
   # Example: gitconfig.tmpl
   [user]
       name = {{ .user }}
       email = {{ .user }}@{{ .company }}.com
   [core]
       editor = {{ .editor }}
   ```

### Available Variables

Templates have access to several built-in variables:

- `{{ .user }}` - Current username
- `{{ .hostname }}` - Machine hostname  
- `{{ .editor }}` - Configured editor
- `{{ .shell }}` - Configured shell

### Custom Variables

Add custom variables by editing `~/.config/config-manager/config.json`:

```json
{
  "global_variables": {
    "company": "mycompany",
    "email_domain": "example.com"
  }
}
```

### Using Templates

1. **Mark a file as a template** when adding it, or edit the config:
   ```json
   {
     "name": ".gitconfig",
     "template": true,
     "variables": {
       "company": "specific-company"
     }
   }
   ```

2. **Link the template** - Config Manager will generate the final file using your variables

## Configuration Structure

Config Manager stores everything in `~/.config/config-manager/`:

```
~/.config/config-manager/
├── config.json          # Main configuration file
├── dotfiles/            # Your managed dotfiles
│   ├── shell/
│   ├── editor/
│   ├── git/
│   └── misc/
├── templates/           # Template files
│   ├── gitconfig.tmpl
│   ├── zshrc.tmpl
│   └── vimrc.tmpl
└── backups/            # Automatic backups
    └── 2024-01-15_14-30-45/
```

## Example Workflows

### Managing Neovim Configuration

```bash
# Add your nvim config directory
# Press 'a', select '.config/nvim (directory)'

# Edit your config
# Select nvim, press 'e', choose which file to edit

# Link it
# Press 'l' to link just nvim, or 'L' to link everything
```

### Setting Up Shell Configuration

```bash
# Add your shell config
# Press 'a', select '.zshrc (file)' or '.bashrc (file)'

# Make it a template if you want environment-specific settings
# Edit config.json or recreate with template support

# Link across machines
# Your shell config will adapt to each environment
```

### Backing Up Before Changes

```bash
# Always backup before major changes
# Press 'b' to create a timestamped backup

# Backups are stored in ~/.config/config-manager/backups/
```

## Advanced Usage

### Custom Categories

Edit `config.json` to add your own categories:

```json
{
  "categories": ["shell", "editor", "git", "terminal", "wm", "custom"]
}
```

### Template Extensions

Customize which file extensions are treated as templates:

```json
{
  "template_extensions": [".tmpl", ".template", ".tpl", ".j2"]
}
```

### Editor Configuration

Config Manager works with any editor. Popular configurations:

```json
{
  "editor": "nvim"     // or "vim", "code", "emacs", "nano"
}
```

## Troubleshooting

### Common Issues

**Q: Config Manager doesn't find my dotfiles**
A: Make sure they're in your home directory and not hidden by `.gitignore` or similar.

**Q: Symlinks aren't working**
A: Check that the source files exist in `~/.config/config-manager/dotfiles/` and you have proper permissions.

**Q: Templates aren't rendering**
A: Verify your template syntax and check that variables are defined in your config.

**Q: Editor integration isn't working**
A: Make sure your editor is in your `$PATH` and the editor name in config matches the command.

### Getting Help

- Check the status messages in the application
- Look at backup files if something goes wrong
- File issues on [GitHub](https://github.com/pergatore/config-manager/issues)

## Contributing

We welcome contributions! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Charm's](https://charm.sh/) amazing TUI libraries
- Inspired by various dotfile management tools
- Thanks to the Go community for excellent tooling
