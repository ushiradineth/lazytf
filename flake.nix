{
  description = "lazytf: terminal UI for Terraform plans";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      flake-utils,
      nixpkgs,
    }:
    (flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        lazytf = pkgs.buildGoModule {
          pname = "lazytf";
          version = self.shortRev or self.dirtyShortRev or "dev";

          src = ./.;
          vendorHash = "sha256-twmrMrtvUVzDiB8FHgDiAf9gbsCR+/mCZfmMucXWTcs=";
          proxyVendor = true;
          go = pkgs.go_1_25;
          doCheck = false;

          env = {
            GOTOOLCHAIN = "local";
          };

          ldflags = [
            "-s"
            "-w"
            "-X github.com/ushiradineth/lazytf/internal/consts.Version=${self.shortRev or self.dirtyShortRev or "dev"}"
          ];

          meta = with pkgs.lib; {
            description = "Terminal UI for reviewing Terraform plans";
            homepage = "https://github.com/ushiradineth/lazytf";
            platforms = platforms.unix;
            mainProgram = "lazytf";
          };
        };
      in
      {
        packages.default = lazytf;
        packages.lazytf = lazytf;

        apps.default = flake-utils.lib.mkApp {
          drv = lazytf;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go

            pkgs.gopls
            pkgs.gofumpt
            pkgs.goimports-reviser
            pkgs.golines
            pkgs.golangci-lint
            pkgs.govulncheck
            pkgs.go-tools
            pkgs.gow

            pkgs.vimPlugins.nvim-treesitter-parsers.go
            pkgs.vimPlugins.nvim-treesitter-parsers.gomod
          ];
        };
      }
    ))
    // {
      overlays.default = final: prev: {
        lazytf = self.packages.${final.system}.default;
      };

      nixosModules.default = import ./nix/modules/nixos.nix;
      homeManagerModules.default = import ./nix/modules/home-manager.nix;
    };
}
