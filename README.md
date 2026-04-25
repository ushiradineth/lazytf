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

<details>
<summary><strong>NixOS module usage</strong></summary>

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

</details>

<details>
<summary><strong>Home Manager module usage</strong></summary>

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

</details>

## Usage

Start `lazytf` directly:

```bash
lazytf
```

Start with an existing Terraform plan file:

```bash
lazytf --plan plan.tfplan
```

If the plan is not in the current directory, pass `--workdir`:

```bash
lazytf --plan ../infrastructure/plan.tfplan --workdir ../infrastructure
```

One liner to start `lazytf` with a plan run:

```bash
terraform plan -no-color | lazytf --plan -
```

Read-only mode if you only want to visualise the diffs:

```bash
terraform plan -no-color | lazytf --plan - --readonly
```

## Configuration

Full option reference lives in [`CONFIGURATION.md`](./CONFIGURATION.md).

`lazytf` supports both Terraform and OpenTofu. By default it looks for `terraform` first and then `tofu`. If `terraform.binary` is set in config, that configured path is used consistently across the TUI.

Choose a global binary explicitly:

```yaml
terraform:
  binary: /opt/homebrew/bin/tofu   # or /opt/homebrew/bin/terraform
```

You can also add project overrides keyed by path. These apply project-scoped binary, theme, and flags:

```yaml
project_overrides:
  /Users/you/work/prod-infra:
    binary: /opt/homebrew/bin/terraform
    theme: nord
    flags:
      - -lock-timeout=60s
```

Precedence: matching `project_overrides.<path>.binary` overrides global `terraform.binary` for that project.

Path resolution order:

1. `LAZYTF_CONFIG`
2. `$XDG_CONFIG_HOME/lazytf/config.yaml`
3. `~/.config/lazytf/config.yaml`
4. `/etc/lazytf/config.yaml`

## Development Prerequisites

- Go 1.25.8
- [just](https://github.com/casey/just)

### With Nix

- Install [direnv](https://direnv.net/)
- Run `direnv allow` or `just shell`

### Without nix

- `just deps-tooling`

```bash
# Install Go dependencies
just deps

# Run the application
just run # or `just dev` for live reload
```

### Tooling

Run `just` without arguments to see all available commands.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `just check` to ensure quality
5. Submit a pull request

### Before Raising a Pull Request

Always run the quality check, `just check`.

## Credits

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm.
- UI components from [Bubbles](https://github.com/charmbracelet/bubbles).
- Terminal styling from [Lipgloss](https://github.com/charmbracelet/lipgloss).
- Inspired by [Terraform Cloud](https://cloud.hashicorp.com/products/terraform) and [lazygit](https://github.com/jesseduffield/lazygit).
