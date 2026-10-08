package v1

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

func Parse(r io.Reader) (*Config, error) {
	doc, err := kdl.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config as KDL: %w", err)
	}

	conf := &Config{
		options: map[string]*configOption{},
	}

	for _, node := range doc.Nodes {
		if err := parseConfigNode(conf, node); err != nil {
			return nil, fmt.Errorf(
				"cannot process node %s: %w",
				node.Name.NodeNameString(),
				err,
			)
		}
	}

	seenShorts := map[string]string{}
	for name, opt := range conf.options {
		if opt.short == "" {
			continue
		}

		if previous, ok := seenShorts[opt.short]; ok {
			return nil, fmt.Errorf(
				"short option %s is defined in both %s and %s",
				opt.short,
				name,
				previous,
			)
		}
		seenShorts[opt.short] = name
	}

	return conf, nil
}

func parseConfigNode(conf *Config, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "execute":
		return parseConfigNodeExecute(conf, node)
	case "example":
		return parseConfigNodeExample(conf, node)
	case "description":
		return parseConfigNodeDescription(conf, node)
	case "option":
		return parseConfigNodeOption(conf, node)
	case "argument":
		return parseConfigNodeArgument(conf, node)
	}

	return errors.New("unknown node type")
}

func parseConfigNodeExecute(conf *Config, node *document.Node) error {
	args, err := parseArguments[string](node)
	if err != nil {
		return err
	}

	conf.execute = args

	return nil
}

func parseConfigNodeExample(conf *Config, node *document.Node) error {
	val, err := parseOneArgument[string](node)
	if err != nil {
		return err
	}

	conf.example = val

	return nil
}

func parseConfigNodeDescription(conf *Config, node *document.Node) error {
	val, err := parseOneArgument[string](node)
	if err != nil {
		return err
	}

	conf.description = val

	return nil
}

func parseConfigNodeOption(conf *Config, node *document.Node) error {
	option := &configOption{
		minCount: -1,
		maxCount: int(^uint(0) >> 1),
	}

	name, err := parseOneArgument[string](node)
	if err != nil {
		return errors.New("cannot parse name")
	}

	if val, ok := node.Properties.Get("short"); ok && val != nil {
		val2, ok := val.Value.(string)
		if !ok {
			return errors.New("value of 'short' must be string")
		}
		if lv := utf8.RuneCountInString(val2); lv != 1 {
			return fmt.Errorf("length of 'short' must be 1, not %d", lv)
		}
		option.short = val2
	}

	if err := parseConfigItem(&option.configItem, node); err != nil {
		return fmt.Errorf("cannot parse option %s: %w", name, err)
	}

	conf.options[name] = option

	return nil
}

func parseConfigNodeArgument(conf *Config, node *document.Node) error {
	arg := &configArgument{}

	if err := parseConfigItem(&arg.configItem, node); err != nil {
		return fmt.Errorf("cannot parse argument: %w", err)
	}

	conf.argument = arg

	return nil
}

func parseConfigItem(item *configItem, node *document.Node) error {
	for _, child := range node.Children {
		if err := parseConfigItemChild(item, child); err != nil {
			return fmt.Errorf("cannot parse %s node: %w", child.Name.NodeNameString(), err)
		}
	}

	return nil
}

func parseConfigItemChild(item *configItem, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "description":
		return parseConfigItemChildDescription(item, node)
	case "min-count":
		return parseConfigItemChildMinCount(item, node)
	case "max-count":
		return parseConfigItemChildMaxCount(item, node)
	case "value":
		return parseConfigItemChildValue(item, node)
	}

	return errors.New("unknown node type")
}

func parseConfigItemChildDescription(item *configItem, node *document.Node) error {
	arg, err := parseOneArgument[string](node)
	if err != nil {
		return fmt.Errorf("cannot parse description: %w", err)
	}

	item.description = arg

	return nil
}

func parseConfigItemChildMinCount(item *configItem, node *document.Node) error {
	arg, err := parseOneArgument[int64](node)
	if err != nil {
		return fmt.Errorf("cannot parse min-count: %w", err)
	}

	if int64(int(arg)) != arg {
		return fmt.Errorf("min-count %d is out of range for int", arg)
	}
	item.minCount = int(arg)

	return nil
}

func parseConfigItemChildMaxCount(item *configItem, node *document.Node) error {
	arg, err := parseOneArgument[int64](node)
	if err != nil {
		return fmt.Errorf("cannot parse max-count: %w", err)
	}

	if int64(int(arg)) != arg {
		return fmt.Errorf("max-count %d is out of range for int", arg)
	}
	item.maxCount = int(arg)

	return nil
}

func parseConfigItemChildValue(item *configItem, node *document.Node) error {
	valueType, err := parseOneArgument[string](node)
	if err != nil {
		return fmt.Errorf("cannot parse value type: %w", err)
	}

	item.valueType = valueType
	item.valueParams = map[string][]any{}

	for _, child := range node.Children {
		paramName := child.Name.NodeNameString()
		paramValue, err := parseArguments[any](child)
		if err != nil {
			return fmt.Errorf("cannot parse argument of value %s: %w", paramName, err)
		}

		item.valueParams[paramName] = paramValue
	}

	return nil
}

func parseOneArgument[T any](node *document.Node) (T, error) {
	if lv := len(node.Arguments); lv != 1 {
		return *new(T), fmt.Errorf("expected 1 argument, got %d", lv)
	}

	val, ok := node.Arguments[0].Value.(T)
	if !ok {
		return *new(T), fmt.Errorf("unexpected value of type %T, expected %T", node.Arguments[0].Value, *new(T))
	}

	return val, nil
}

func parseArguments[T any](node *document.Node) ([]T, error) {
	rv := make([]T, len(node.Arguments))

	for idx, item := range node.Arguments {
		val, ok := item.Value.(T)
		if !ok {
			return nil, fmt.Errorf("unexpected value of type %T, expected %T", item.Value, *new(T))
		}

		rv[idx] = val
	}

	return rv, nil
}
