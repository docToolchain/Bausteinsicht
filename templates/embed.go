package templates

import _ "embed" // enables the //go:embed directives below; targets are []byte, not embed.FS, so no named import is used

//go:embed default.drawio
var DefaultTemplate []byte

//go:embed sample-model.jsonc
var SampleModel []byte
