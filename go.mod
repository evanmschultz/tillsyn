module github.com/evanmschultz/tillsyn

go 1.26.1

replace github.com/charmbracelet/x/exp/teatest/v2 => ./third_party/teatest_v2

replace charm.land/fantasy => github.com/evanmschultz/fantasy v0.0.0-20260219222711-d1be5103494b

replace charm.land/lipgloss/v2 => charm.land/lipgloss/v2 v2.0.0-beta.3.0.20260212100304-e18737634dea

replace github.com/alecthomas/chroma/v2 => github.com/alecthomas/chroma/v2 v2.14.0

replace github.com/aymanbagabas/go-udiff => github.com/aymanbagabas/go-udiff v0.3.1

replace github.com/charmbracelet/colorprofile => github.com/charmbracelet/colorprofile v0.4.2

replace github.com/charmbracelet/ultraviolet => github.com/charmbracelet/ultraviolet v0.0.0-20251205161215-1948445e3318

replace github.com/charmbracelet/x/exp/golden => github.com/charmbracelet/x/exp/golden v0.0.0-20250806222409-83e3a29d542f

replace github.com/charmbracelet/x/exp/slice => github.com/charmbracelet/x/exp/slice v0.0.0-20250904123553-b4e2667e5ad5

replace github.com/clipperhouse/displaywidth => github.com/clipperhouse/displaywidth v0.9.0

replace github.com/clipperhouse/uax29/v2 => github.com/clipperhouse/uax29/v2 v2.5.0

replace github.com/dlclark/regexp2 => github.com/dlclark/regexp2 v1.11.0

replace github.com/go-logfmt/logfmt => github.com/go-logfmt/logfmt v0.6.0

replace github.com/lucasb-eyer/go-colorful => github.com/lucasb-eyer/go-colorful v1.3.0

replace github.com/mattn/go-runewidth => github.com/mattn/go-runewidth v0.0.19

replace github.com/yuin/goldmark => github.com/yuin/goldmark v1.7.8

replace github.com/yuin/goldmark-emoji => github.com/yuin/goldmark-emoji v1.0.5

replace golang.org/x/exp => golang.org/x/exp v0.0.0-20260212183809-81e46e3db34a

replace golang.org/x/net => golang.org/x/net v0.50.0

replace golang.org/x/sync => golang.org/x/sync v0.19.0

replace golang.org/x/sys => golang.org/x/sys v0.41.0

replace golang.org/x/term => golang.org/x/term v0.40.0

replace golang.org/x/text => golang.org/x/text v0.34.0

require (
	charm.land/bubbles/v2 v2.0.0-rc.1
	charm.land/bubbletea/v2 v2.0.0-rc.2
	charm.land/fantasy v0.0.0-00010101000000-000000000000
	charm.land/lipgloss/v2 v2.0.2
	github.com/asg017/sqlite-vec-go-bindings v0.1.6
	github.com/charmbracelet/fang v0.4.4
	github.com/charmbracelet/glamour v0.10.0
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834
	github.com/charmbracelet/log v0.4.2
	github.com/charmbracelet/x/exp/teatest/v2 v2.0.0-20260216111343-536eb63c1f4c
	github.com/evanmschultz/autent v0.1.1
	github.com/evanmschultz/laslig v0.2.2
	github.com/google/uuid v1.6.0
	github.com/ncruces/go-sqlite3 v0.23.3
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/spf13/cobra v1.10.2
	github.com/tetratelabs/wazero v1.11.0
)

require (
	charm.land/glamour/v2 v2.0.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/charmbracelet/x/exp/charmtone v0.0.0-20250603201427-c31516f43444 // indirect
	github.com/charmbracelet/x/json v0.2.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/kaptinlin/go-i18n v0.2.9 // indirect
	github.com/kaptinlin/jsonpointer v0.4.16 // indirect
	github.com/kaptinlin/jsonschema v0.7.2 // indirect
	github.com/kaptinlin/messageformat-go v0.4.18 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/muesli/mango v0.1.0 // indirect
	github.com/muesli/mango-cobra v1.2.0 // indirect
	github.com/muesli/mango-pflag v0.1.0 // indirect
	github.com/muesli/roff v0.1.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/openai/openai-go/v2 v2.7.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/tools v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.46.1 // indirect
	mvdan.cc/gofumpt v0.9.2 // indirect
)

require (
	github.com/alecthomas/chroma/v2 v2.23.1 // indirect
	github.com/atotto/clipboard v0.1.4
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymanbagabas/go-udiff v0.4.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260316091819-b93f6a3b8502 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/golden v0.0.0-20260323091123-df7b1bcffcca // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20260323091123-df7b1bcffcca // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mark3labs/mcp-go v0.44.0
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	github.com/yuin/goldmark-emoji v1.0.6 // indirect
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0
	golang.org/x/text v0.35.0 // indirect
)

tool mvdan.cc/gofumpt
