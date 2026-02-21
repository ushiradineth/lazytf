{ lib }:
{
  "general" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "default_environment" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "history" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "compression_threshold" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "level" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "max_entries" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "path" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "retention_days" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "presets" = lib.mkOption {
    type = lib.types.nullOr (lib.types.listOf (lib.types.submodule {
      options = {
        "environment" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "strategy" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "theme" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "workdir" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    }));
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "project_overrides" = lib.mkOption {
    type = lib.types.nullOr (lib.types.attrsOf (lib.types.submodule {
      options = {
        "flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "preset_name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "theme" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    }));
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "terraform" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "binary" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "default_flags" = lib.mkOption {
          type = lib.types.nullOr (lib.types.listOf (lib.types.str));
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "parallelism" = lib.mkOption {
          type = lib.types.nullOr (lib.types.int);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "timeout" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "working_dir" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "theme" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "custom" = lib.mkOption {
          type = lib.types.nullOr (lib.types.submodule {
            options = {
              "background_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "border_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "create_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "delete_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "dimmed_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "foreground_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "highlight_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "name" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "no_change_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "replace_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "selected_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
              "update_color" = lib.mkOption {
                type = lib.types.nullOr (lib.types.str);
                default = null;
                description = "Auto-generated from internal/config/config.schema.json";
              };
            };
          });
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "name" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "ui" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "animations_enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "compact_mode" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "mouse_enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "show_help" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "show_line_numbers" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "split_ratio" = lib.mkOption {
          type = lib.types.nullOr (lib.types.float);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
        "split_view_default" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "version" = lib.mkOption {
    type = lib.types.nullOr (lib.types.int);
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
}
