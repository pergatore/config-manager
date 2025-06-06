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

Config Manager supports dynamic templates that automatically adapt to different environments and machines. Instead of manually editing config files on each computer, templates fill in the details automatically.

### What Templates Do

Templates let you create **one config file** that automatically adapts to different computers. For example, your `.gitconfig` can automatically use different email addresses for work vs personal machines, while keeping everything else the same.

### Complete Example: Git Configuration

Let's create a `.gitconfig` that automatically adapts based on your machine's hostname:

#### Step 1: Create the Template

```bash
# Create the templates directory
mkdir -p ~/.config/config-manager/templates

# Create a git config template that uses hostname detection
cat > ~/.config/config-manager/templates/gitconfig.tmpl << 'EOF'
[user]
    name = {{ .user }}
    email = {{ .user }}@{{ if contains .hostname "work" }}company.com{{ else }}gmail.com{{ end }}
[core]
    editor = {{ .editor }}
    autocrlf = input

# Work-specific settings
{{ if contains .hostname "work" }}
[url "git@github.com:company/"]
    insteadOf = https://github.com/company/
[commit]
    gpgsign = true
[user]
    signingkey = ABC123DEF456
{{ end }}

# Personal settings
{{ if not (contains .hostname "work") }}
[github]
    user = {{ .user }}
[pull]
    rebase = true
{{ end }}

[push]
    default = simple
[alias]
    st = status
    co = checkout
    br = branch
    # Configuration for {{ .user }} on {{ .hostname }}
EOF
```

#### Step 2: No Variables Needed!

Your `~/.config/config-manager/config.json` stays completely portable - no machine-specific variables needed:

```json
{
  "editor": "vim",
  "shell": "zsh",
  "global_variables": {}
}
```

#### Step 3: Add .gitconfig as a Managed Template

1. In config-manager, press `a` to add your existing `.gitconfig`
2. Edit the config to mark it as a template:

```json
{
  "name": ".gitconfig",
  "source": "git/gitconfig",
  "target": "/home/username/.gitconfig",
  "category": "git",
  "template": true,
  "variables": {}
}
```

#### Step 4: Link the Template

Press `l` in config-manager to link your .gitconfig.

**What happens:**
1. Config-manager reads `templates/gitconfig.tmpl`
2. Checks your hostname and applies appropriate conditionals
3. Fills in built-in variables: `{{ .user }}`, `{{ .hostname }}`, `{{ .editor }}`
4. Creates the final file and symlinks it

#### Step 5: Results on Different Machines

**On work laptop** (hostname: `work-laptop-01`):
```ini
[user]
    name = john
    email = john@company.com
[core]
    editor = vim
    autocrlf = input
[url "git@github.com:company/"]
    insteadOf = https://github.com/company/
[commit]
    gpgsign = true
[user]
    signingkey = ABC123DEF456
[push]
    default = simple
[alias]
    st = status
    co = checkout
    br = branch
    # Configuration for john on work-laptop-01
```

**On personal MacBook** (hostname: `Johns-MacBook-Pro`):
```ini
[user]
    name = john
    email = john@gmail.com
[core]
    editor = vim
    autocrlf = input
[github]
    user = john
[pull]
    rebase = true
[push]
    default = simple
[alias]
    st = status
    co = checkout
    br = branch
    # Configuration for john on Johns-MacBook-Pro
```

**The key insight:** Same template file, same config.json, but different outputs based on hostname detection!

### Advanced Example: Environment-Specific Shell Config

Create `~/.config/config-manager/templates/zshrc.tmpl`:

```bash
# {{ .user }}'s {{ .shell }} configuration on {{ .hostname }}
export EDITOR="{{ .editor }}"
export PATH="$HOME/bin:$PATH"

# Work-specific settings (based on hostname)
{{ if contains .hostname "work" }}
export COMPANY_API_KEY="work-api-key-123"
export WORK_PROJECT_DIR="$HOME/work"
alias deploy="kubectl apply -f"
alias vpn="sudo openvpn /etc/openvpn/work.conf"
{{ end }}

# Personal settings (based on hostname)
{{ if contains .hostname "home" }}
export PERSONAL_PROJECT_DIR="$HOME/projects"
alias blog="cd $PERSONAL_PROJECT_DIR/blog"
alias backup="rsync -av $HOME/Documents/ /backup/"
{{ else if contains .hostname "macbook" }}
export PERSONAL_PROJECT_DIR="$HOME/Code"
alias blog="cd $PERSONAL_PROJECT_DIR/blog"
# macOS-specific aliases
alias brew-update="brew update && brew upgrade"
{{ end }}

# Common aliases for all machines
alias ll="ls -la"
alias ..="cd .."
alias grep="grep --color=auto"
```

And `~/.config/config-manager/templates/gitconfig.tmpl`:

```bash
[user]
    name = {{ .user }}
    email = {{ .user }}@{{ if contains .hostname "work" }}company.com{{ else }}gmail.com{{ end }}
[core]
    editor = {{ .editor }}
    autocrlf = input

# Work-specific git settings
{{ if contains .hostname "work" }}
[url "git@github.com:company/"]
    insteadOf = https://github.com/company/
[commit]
    gpgsign = true
{{ end }}

# Personal git settings  
{{ if not (contains .hostname "work") }}
[github]
    user = {{ .user }}
{{ end }}

[alias]
    st = status
    co = checkout
    br = branch
```

**Why Hostname-Based Conditionals Work Best:**

- ✅ **No circular dependencies** - hostname is always available when templates are processed
- ✅ **Fully portable** - same config.json works on all machines
- ✅ **Self-contained** - no external environment setup needed
- ✅ **Immediate evaluation** - works reliably every time you link
- ✅ **Predictable** - same hostname always produces the same result

**Different machines automatically get different configs:**

**Work laptop** (`work-laptop-01`):
```bash
export COMPANY_API_KEY="work-api-key-123"
email = john@company.com
```

**Personal MacBook** (`Johns-MacBook-Pro`):
```bash
export PERSONAL_PROJECT_DIR="$HOME/Code"
email = john@gmail.com
alias brew-update="brew update && brew upgrade"
```

**Home desktop** (`home-desktop`):
```bash
export PERSONAL_PROJECT_DIR="$HOME/projects"
email = john@gmail.com
alias backup="rsync -av $HOME/Documents/ /backup/"
```

This approach keeps your `config.json` completely portable while automatically adapting to each machine!

### Available Variables

Templates automatically have access to:

**Built-in System Variables:**
- `{{ .user }}` - Current username (from `$USER` environment variable)
- `{{ .hostname }}` - Machine hostname (from `os.Hostname()`)
- `{{ .editor }}` - Your configured editor (from config-manager setup)
- `{{ .shell }}` - Your configured shell (from config-manager setup)

**Custom Variables:**
- Any variables from `global_variables` in config.json
- Any file-specific variables in the file's `variables` section

**Example of all variable types:**
```bash
# Built-in variables
User: {{ .user }}           # → john
Host: {{ .hostname }}       # → work-laptop  
Editor: {{ .editor }}       # → vim
Shell: {{ .shell }}         # → zsh

# Custom global variables (from config.json global_variables)
Email: {{ .email_domain }}  # → company.com
Environment: {{ .environment }} # → work

# File-specific variables (from file's variables section)
API Key: {{ .api_key }}     # → secret-key-123
```

### Template Functions

Use Go template functions for dynamic content:

- `{{ if eq .environment "work" }}...{{ end }}` - Conditional sections
- `{{ .user | upper }}` - Transform text to uppercase
- `{{ .hostname | lower }}` - Transform text to lowercase

### Template Workflow

1. **Create template file** in `~/.config/config-manager/templates/`
2. **Add variables** to `config.json` 
3. **Mark file as template** when adding to config-manager
4. **Link the template** - config-manager generates the final file
5. **On new machines** - same template + different variables = different output

This way, you maintain **one template** but get **machine-specific configs** automatically! Perfect for managing configurations across work laptops, personal machines, and servers.

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
