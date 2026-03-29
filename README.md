# lazytf

lazygit but for Terraform :D

## Installation

### Binary releases

Download platform archives and checksums from GitHub Releases.

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

Run without installing:

```bash
nix run github:ushiradineth/lazytf
```

Build from source:

```bash
nix build github:ushiradineth/lazytf
```

## Configuration

### YAML config

Config is stored in YAML and supports a schema hint for editors.

Path resolution order:

1. `LAZYTF_CONFIG`
2. `$XDG_CONFIG_HOME/lazytf/config.yaml`
3. `~/.config/lazytf/config.yaml`
4. `/etc/lazytf/config.yaml`

Example:

```yaml
version: 1
theme:
  name: default
terraform:
  default_flags:
    - -compact-warnings
  timeout: 10m
history:
  enabled: true
  level: standard
notifications:
  enabled: true
  sink:
    protocol: cloudevents-http
    url: https://example.com/hooks/lazytf
    timeout: 3s
    source: https://github.com/ushiradineth/lazytf
```

Notification delivery uses CloudEvents 1.0 JSON over HTTP(S). `notifications.enabled` toggles delivery on or off.

### Nix

NixOS module usage:

```nix
{
  inputs.lazytf.url = "github:ushiradineth/lazytf";

  outputs = { self, nixpkgs, lazytf, ... }: {
    nixosConfigurations.host = nixpkgs.lib.nixosSystem {
      modules = [
        lazytf.nixosModules.default
        ({ pkgs, ... }: {
          nixpkgs.overlays = [ lazytf.overlays.default ];
          programs.lazytf.enable = true;
          programs.lazytf.settings.theme.name = "default";
        })
      ];
    };
  };
}
```

Home Manager module usage:

```nix
{
  inputs.lazytf.url = "github:ushiradineth/lazytf";

  outputs = { home-manager, lazytf, ... }: {
    homeConfigurations.user = home-manager.lib.homeManagerConfiguration {
      modules = [
        lazytf.homeManagerModules.default
        ({ pkgs, ... }: {
          nixpkgs.overlays = [ lazytf.overlays.default ];
          programs.lazytf.enable = true;
          programs.lazytf.settings.theme.name = "default";
        })
      ];
    };
  };
}
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
