package configs

import _ "embed"

//go:embed config.json
var DefaultConfig []byte
