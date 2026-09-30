package v1

type configItem struct {
	description string
	minCount    int
	maxCount    int
	valueType   string
	valueParams map[string]any
}

type configOption struct {
	configItem
	short string
}

type configArgument struct {
	configItem
}

type Config struct {
	options     map[string]configOption
	argument    configArgument
	execute     []string
	description string
	example     string
}

func (c *Config) ExecArgv() []string {
	return nil
}
