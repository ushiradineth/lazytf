{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.lazytf;
  yamlFormat = pkgs.formats.yaml { };
  generatedOptions = import ./generated/config-options.nix { inherit lib; };
  pruneNulls =
    value:
    if builtins.isAttrs value then
      lib.filterAttrs (_: v: v != null) (lib.mapAttrs (_: v: pruneNulls v) value)
    else if builtins.isList value then
      builtins.filter (v: v != null) (map pruneNulls value)
    else
      value;
in
{
  options.programs.lazytf = {
    enable = lib.mkEnableOption "lazytf";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.lazytf;
      defaultText = lib.literalExpression "pkgs.lazytf";
      description = "Package to install for lazytf.";
    };

    settings = lib.mkOption {
      type = lib.types.submodule {
        options = generatedOptions;
      };
      default = { };
      description = "Settings to write to ~/.config/lazytf/config.yaml.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];
    xdg.configFile."lazytf/config.yaml".source = yamlFormat.generate "lazytf-config.yaml" (pruneNulls cfg.settings);
  };
}
