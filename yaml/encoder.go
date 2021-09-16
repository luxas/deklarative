package yaml

// the copy comments package should probably be moved here

type encoderConfig struct {
	// setIndent int -- kyaml doesn't allow us to configure this
	// seqStyleIndent
	// escapeHTML? -- seems like there's no such option
	// sortMapKeys? -- one can manipulate the list of RNodes before marshalling
}
