{ lib }:
{
  "default_environment" = lib.mkOption {
    type = lib.types.nullOr (lib.types.str);
    default = null;
    description = "Default workspace or folder environment to select when lazytf starts.";
  };
  "history" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "compression_threshold" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Compress stored output larger than this many bytes.";
        };
        "enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Enable persistent operation history.";
        };
        "level" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "History detail level. Supported values are minimal, standard, and full.";
        };
        "path" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Path to the history database file.";
        };
      };
    });
    default = null;
    description = "History storage and retention settings.";
  };
  "mouse" = lib.mkOption {
    type = lib.types.nullOr (lib.types.bool);
    default = null;
    description = "Enable mouse navigation in the UI. By default lazytf enables mouse outside tmux and disables it inside tmux to respect tmux mouse settings. Set this explicitly to override that behavior.";
  };
  "notification" = lib.mkOption {
    type = lib.types.nullOr (lib.types.bool);
    default = null;
    description = "Enable user notifications for important UI events. Set this explicitly to override the default behavior.";
  };
  "presets" = lib.mkOption {
    type = lib.types.nullOr (lib.types.listOf (lib.types.submodule {
      options = {
        "environment" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Workspace or folder environment selected when this preset is used.";
        };
        "flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Additional Terraform flags appended when this preset is selected.";
        };
        "name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Preset name used with --preset.";
        };
        "theme" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Built-in UI theme to apply when this preset is selected.";
        };
        "workdir" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Working directory to use when this preset is selected.";
        };
      };
    }));
    default = null;
    description = "Named presets that bundle environment selection, workdir, theme, and default Terraform flags.";
  };
  "project_overrides" = lib.mkOption {
    type = lib.types.nullOr (lib.types.attrsOf (lib.types.submodule {
      options = {
        "flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Additional Terraform flags for this project.";
        };
        "preset_name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Preset name to apply for this project before project-specific overrides.";
        };
        "theme" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Built-in UI theme override for this project.";
        };
      };
    }));
    default = null;
    description = "Per-project overrides keyed by project path.";
  };
  "terraform" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "binary" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Path to the Terraform binary to run.";
        };
        "default_flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Default flags appended to Terraform commands run by lazytf.";
        };
        "parallelism" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Default Terraform parallelism value used when no explicit -parallelism flag is provided.";
        };
        "timeout" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Maximum time allowed for Terraform commands before lazytf cancels them.";
        };
        "working_dir" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Default working directory used when no folder or preset overrides it.";
        };
      };
    });
    default = null;
    description = "Terraform execution settings used by lazytf.";
  };
  "theme" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Built-in UI theme name.";
        };
      };
    });
    default = null;
    description = "Theme settings for the lazytf UI.";
  };
}
