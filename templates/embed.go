package templates

import _ "embed"

//go:embed default.drawio
var DefaultTemplate []byte

//go:embed sample-model.jsonc
var SampleModel []byte
