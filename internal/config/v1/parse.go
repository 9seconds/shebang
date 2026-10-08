package v1

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"

	"github.com/9seconds/shebang/internal/utils"
	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

var (
	ErrUnknownNode         = errors.New("unknown node")
	ErrOneArgumentExpected = errors.New("only 1 vararg can be defined")
	ErrDefineExecute       = errors.New("execute cannot be empty")

	ReName = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

func Parse(r io.Reader) (*Config, error) {
	doc, err := kdl.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config as KDL: %w", err)
	}

	conf := &Config{
		Argv:           []string{"bash"},
		seenLongNames:  make(map[string]bool),
		seenShortNames: make(map[string]bool),
	}

	for _, node := range doc.Nodes {
		name := node.Name.NodeNameString()
		if err := processConfigNode(conf, node); err != nil {
			return nil, fmt.Errorf("cannot process node %s: %w", name, err)
		}
	}

	if len(conf.Argv) == 0 {
		return nil, ErrDefineExecute
	}

	return conf, nil
}

func processConfigNode(conf *Config, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "description":
		return setSingleArgument(&conf.Description, node)
	case "example":
		return setSingleArgument(&conf.Example, node)
	case "execute":
		return processConfigNodeExecute(conf, node)
	case "option":
		return processConfigNodeOption(conf, node)
	case "flag":
		return processConfigNodeFlag(conf, node)
	case "arg":
		return processConfigNodeArg(conf, node)
	case "vararg":
		return processConfigNodeVarArg(conf, node)
	}

	return ErrUnknownNode
}

func processConfigNodeOption(conf *Config, node *document.Node) error {
	opt := Option{}

	if err := setName(&opt.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	if _, ok := conf.seenLongNames[opt.Name]; ok {
		return fmt.Errorf("duplicate long name %s", opt.Name)
	}
	conf.seenLongNames[opt.Name] = true

	for _, chld := range node.Children {
		name := chld.Name.NodeNameString()
		if err := processOptionNode(&opt, chld); err != nil {
			return fmt.Errorf("cannot process option %s: %w", name, err)
		}
	}

	if opt.Short != "" {
		if _, ok := conf.seenShortNames[opt.Short]; ok {
			return fmt.Errorf("duplicate short name %s", opt.Short)
		}
		conf.seenShortNames[opt.Short] = true
	}

	conf.Options = append(conf.Options, opt)

	return nil
}

func processOptionNode(opt *Option, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "description":
		return setSingleArgument(&opt.Description, node)
	case "short":
		return processShort(&opt.Short, node)
	case "value":
		return processWithValue(&opt.WithValue, node)
	}

	return ErrUnknownNode
}

func processConfigNodeFlag(conf *Config, node *document.Node) error {
	flag := Flag{}

	if err := setName(&flag.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	if _, ok := conf.seenLongNames[flag.Name]; ok {
		return fmt.Errorf("duplicate long name %s", flag.Name)
	}
	conf.seenLongNames[flag.Name] = true

	for _, chld := range node.Children {
		name := chld.Name.NodeNameString()
		if err := processFlagNode(&flag, chld); err != nil {
			return fmt.Errorf("cannot process flag %s: %w", name, err)
		}
	}

	if flag.Short != "" {
		if _, ok := conf.seenShortNames[flag.Short]; ok {
			return fmt.Errorf("duplicate short name %s", flag.Short)
		}
		conf.seenShortNames[flag.Short] = true
	}

	conf.Flags = append(conf.Flags, flag)

	return nil
}

func processFlagNode(flag *Flag, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "short":
		return processShort(&flag.Short, node)
	case "description":
		return setSingleArgument(&flag.Description, node)
	}

	return ErrUnknownNode
}

func processConfigNodeArg(conf *Config, node *document.Node) error {
	arg := Arg{}

	if err := setName(&arg.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	for _, chld := range node.Children {
		name := chld.Name.NodeNameString()
		if err := processArgNode(&arg, chld); err != nil {
			return fmt.Errorf("cannot process argument %s: %w", name, err)
		}
	}

	if conf.VarArgs == nil {
		conf.FirstArgs = append(conf.FirstArgs, arg)
	} else {
		conf.LastArgs = append(conf.LastArgs, arg)
	}

	return nil
}

func processArgNode(arg *Arg, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "description":
		return setSingleArgument(&arg.Description, node)
	case "value":
		return processWithValue(&arg.WithValue, node)
	}

	return ErrUnknownNode
}

func processConfigNodeVarArg(conf *Config, node *document.Node) error {
	if conf.VarArgs != nil {
		return ErrOneArgumentExpected
	}

	arg := &VarArg{}

	if err := setName(&arg.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	for _, chld := range node.Children {
		name := chld.Name.NodeNameString()
		if err := processVarArgNode(arg, chld); err != nil {
			return fmt.Errorf("cannot process vararg %s: %w", name, err)
		}
	}

	if arg.MinCount != nil && arg.MaxCount != nil && *arg.MinCount > *arg.MaxCount {
		return fmt.Errorf(
			"min-count %d is greater than max-count %d",
			*arg.MinCount,
			*arg.MaxCount,
		)
	}

	conf.VarArgs = arg

	return nil
}

func processVarArgNode(arg *VarArg, node *document.Node) error {
	switch node.Name.NodeNameString() {
	case "description":
		return setSingleArgument(&arg.Description, node)
	case "value":
		return processWithValue(&arg.WithValue, node)
	case "min-count":
		return setPointer(&arg.MinCount, node)
	case "max-count":
		return setPointer(&arg.MaxCount, node)
	}

	return ErrUnknownNode
}

func processConfigNodeExecute(conf *Config, node *document.Node) error {
	argv, err := getNodeArguments[string](node)
	if err != nil {
		return err
	}

	conf.Argv = argv

	return nil
}

func processWithValue(data *WithValue, node *document.Node) error {
	valueType, err := getNodeArgument[string](node)
	if err != nil {
		return fmt.Errorf("cannot get a type of the value: %w", err)
	}

	if valueType == "" {
		return errors.New("please define a value type")
	}

	data.Type = valueType
	data.Properties = make(map[string][]any)

	for _, chld := range node.Children {
		propName := chld.Name.NodeNameString()

		propValues := make([]any, len(chld.Arguments))
		for idx, arg := range chld.Arguments {
			propValues[idx] = arg.Value
		}

		data.Properties[propName] = propValues
	}

	return nil
}

func processShort(data *string, node *document.Node) error {
	if err := setSingleArgument(data, node); err != nil {
		return err
	}

	if lv := utf8.RuneCountInString(*data); lv != 1 {
		return fmt.Errorf("short must contain 1 character, not %d", lv)
	}

	if !ReName.MatchString(*data) {
		return fmt.Errorf("must comply %s regexp", ReName.String())
	}

	return nil
}

func setSingleArgument[T any](target *T, node *document.Node) error {
	val, err := getNodeArgument[T](node)
	if err != nil {
		return err
	}

	*target = val

	return nil
}

func setPointer[T any](target **T, node *document.Node) error {
	val, err := getNodeArgument[T](node)
	if err != nil {
		return err
	}

	*target = &val

	return nil
}

func setName(target *string, node *document.Node) error {
	if err := setSingleArgument(target, node); err != nil {
		return err
	}

	if !ReName.MatchString(*target) {
		return fmt.Errorf(
			"value %s does not match regex %s",
			*target,
			ReName.String(),
		)
	}

	return nil
}

func getNodeArguments[T any](node *document.Node) ([]T, error) {
	return utils.All[T](convertNodeArgsToAny(node))
}

func getNodeArgument[T any](node *document.Node) (T, error) {
	return utils.One[T](convertNodeArgsToAny(node))
}

func convertNodeArgsToAny(node *document.Node) []any {
	values := make([]any, len(node.Arguments))

	for idx, arg := range node.Arguments {
		values[idx] = arg.Value
	}

	return values
}
