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
        "mouse_enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
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
        "path" = lib.mkOption {
          type = lib.types.nullOr (lib.types.str);
          default = null;
          description = "Auto-generated from internal/config/config.schema.json";
        };
      };
    });
    default = null;
    description = "Auto-generated from internal/config/config.schema.json";
  };
  "notifications" = lib.mkOption {
    type = lib.types.nullOr (lib.types.submodule {
      options = {
        "enabled" = lib.mkOption {
          type = lib.types.nullOr (lib.types.bool);
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
}
