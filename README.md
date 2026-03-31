![banner](./.github/banner.png)

## Installation

### Binary releases

Download platform archives and checksums from [GitHub Releases](https://github.com/ushiradineth/lazytf/releases/latest).

### Go

```bash
go install github.com/ushiradineth/lazytf/cmd/lazytf@latest
```

### Homebrew

```bash
brew tap ushiradineth/homebrew https://github.com/ushiradineth/homebrew
brew install --cask lazytf
```

### Nix

#### Run without installing

```bash
nix run github:ushiradineth/lazytf
```

#### NixOS module usage

```nix
{
  inputs.lazytf.url = "github:ushiradineth/lazytf";

  outputs = {
    nixpkgs,
    lazytf,
    ...
  }: {
    nixosConfigurations.host = nixpkgs.lib.nixosSystem {
      modules = [
        lazytf.nixosModules.default
        ({...}: {
          nixpkgs.overlays = [lazytf.overlays.default];
          programs.lazytf = {
            enable = true;
            settings.theme.name = "default";
          };
        })
      ];
    };
  };
}
```

#### Home Manager module usage

```nix
{
  inputs.lazytf.url = "github:ushiradineth/lazytf";

  outputs = {
    home-manager,
    lazytf,
    ...
  }: {
    homeConfigurations.user = home-manager.lib.homeManagerConfiguration {
      modules = [
        lazytf.homeManagerModules.default
        ({pkgs, ...}: {
          nixpkgs.overlays = [lazytf.overlays.default];
          programs.lazytf = {
            enable = true;
            settings.theme.name = "default";
          };
        })
      ];
    };
  };
}
```

## Configuration

### YAML config

Config is stored in YAML and supports a yaml schema hint for editors.

Add `# yaml-language-server: $schema=https://raw.githubusercontent.com/ushiradineth/lazytf/main/internal/config/config.schema.json` to the top of your config file.

Path resolution order:

1. `LAZYTF_CONFIG`
2. `$XDG_CONFIG_HOME/lazytf/config.yaml`
3. `~/.config/lazytf/config.yaml`
4. `/etc/lazytf/config.yaml`

Example:

```yaml
version: 1
general:
  mouse_enabled: true
theme:
  name: default
terraform:
  default_flags:
    - -compact-warnings
  timeout: 10m
history:
  enabled: true
  level: standard
```

## Usage

Execution mode with a binary plan file:

```bash
lazytf --plan plan.tfplan
```

When the plan was created in a different Terraform working directory, pass `--workdir`:

```bash
lazytf --plan ../tmpkube/plan.tfplan --workdir ../tmpkube
```

Execution mode from stdin plan text:

```bash
terraform plan -no-color | lazytf --plan -
```

Read-only mode is opt-in:

```bash
terraform plan -no-color | lazytf --plan - --readonly
```

## Development Prerequisites

- Go 1.25.8 or later
- [just](https://github.com/casey/just)

### With Nix

- Install [direnv](https://direnv.net/)
- `direnv allow` or `just shell`

### Without nix

- `just deps-tooling`

## Development

### Quick Start

```bash
# Install dependencies
just deps

# Run the application
just run # or `just dev` for live reload
```

### Tooling

Run `just` without arguments to see all available commands:

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `just check-all` to ensure quality
5. Submit a pull request

### Before Raising a Pull Request

Always run the quality checks:

- `just check-all`
- `just ci`

## Credits

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm.
- UI components from [Bubbles](https://github.com/charmbracelet/bubbles).
- Terminal styling from [Lipgloss](https://github.com/charmbracelet/lipgloss).
- Inspired by [Terraform Cloud](https://cloud.hashicorp.com/products/terraform) and [lazygit](https://github.com/jesseduffield/lazygit).
