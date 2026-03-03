# lazytf

lazygit but for Terraform :D

## Installation

### Binary releases

Download platform archives and checksums from GitHub Releases.

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

The Nix module `programs.lazytf.settings` options are generated from `internal/config/config.schema.json`.
Regenerate schema and Nix options after config model changes:

```bash
just generate-check
```

## Configuration

### YAML config (native)

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
```

### Nix config parity

Both `nixosModules.default` and `homeManagerModules.default` write the same YAML keys under `programs.lazytf.settings`.

NixOS:

```nix
programs.lazytf = {
  enable = true;
  settings = {
    theme.name = "default";
    terraform.default_flags = [ "-compact-warnings" ];
    terraform.timeout = "10m";
    history.enabled = true;
    history.level = "standard";
  };
};
```

Home Manager:

```nix
programs.lazytf = {
  enable = true;
  settings = {
    theme.name = "default";
    terraform.default_flags = [ "-compact-warnings" ];
    terraform.timeout = "10m";
    history.enabled = true;
    history.level = "standard";
  };
};
```

Notes:

- Nix modules only write keys you set. lazytf applies runtime defaults for omitted values.
- Keep YAML schema and Nix options in sync with `just generate-check` after config model changes.

### Go

```bash
go install github.com/ushiradineth/lazytf/cmd/lazytf@latest
```

## Releases

- Tagged releases (`vX.Y.Z`) run `.github/workflows/release.yml` and publish artifacts to GitHub Releases.
- Manual release flow is available via GitHub Actions `workflow_dispatch` with `patch` or `minor` bump.
- Release notes are generated from merged PRs and commits using GitHub native release notes plus `.github/release.yml` categories.
- Homebrew cask updates are published to `ushiradineth/homebrew` by GoReleaser.

### Versioning

- GitHub release artifacts inject semver via ldflags from the release tag.
- Nix flake builds inject `LAZYTF_VERSION` via ldflags when set, otherwise commit hash (`shortRev`/`dirtyShortRev`).
- `just run`, `just build`, and `just install` inject version via ldflags from an exact git tag when present, otherwise short commit SHA. Set `LAZYTF_VERSION` to override explicitly.
- Local/dev fallback version remains `0.1.0` in `internal/consts/consts.go` when ldflags are not applied.

## Prerequisites

- Go 1.25.5 or later
- [just](https://github.com/casey/just)

### With Nix

- Install [direnv](https://direnv.net/)
- `direnv allow`

OR

- `just shell`

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

### Before Committing

Always run the quality checks:

- `just check-all`

## Credits

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm.
- UI components from [Bubbles](https://github.com/charmbracelet/bubbles).
- Terminal styling from [Lipgloss](https://github.com/charmbracelet/lipgloss).
- Inspired by [Terraform Cloud](https://cloud.hashicorp.com/products/terraform) and [lazygit](https://github.com/jesseduffield/lazygit).
