// Why this file:
// The go.mod file defines our project dependencies. Each dependency serves a specific purpose:

// fatih/color - Terminal colors for beautiful CLI output
// fsnotify - File system watching for real-time code updates
// go-sqlite3 - Local metadata storage
// qdrant/go-client - Vector database for semantic search
// sashabaranov/go-openai - Primary AI provider
// spf13/cobra - CLI framework for robust command handling
// spf13/viper - Configuration management
// AI provider SDKs for fallback system
// zap - High-performance logging

module github.com/yourusername/useq-ai-assistant

go 1.23

toolchain go1.24.7

require (
	github.com/fatih/color v1.16.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/joho/godotenv v1.5.1
	github.com/spf13/viper v1.18.2
	go.uber.org/zap v1.26.0
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/qdrant/go-client v1.15.2
	github.com/sashabaranov/go-openai v1.41.2
	github.com/schollz/progressbar/v3 v3.18.0
	google.golang.org/grpc v1.66.0
)

require (
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240827150818-7e3bb234dfed // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
