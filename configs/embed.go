package configs

import "embed"

// Presets contains built-in character and scenario YAML files used when
// filesystem config directories are absent.
//
//go:embed characters/*.yaml scenarios/*.yaml
var Presets embed.FS
