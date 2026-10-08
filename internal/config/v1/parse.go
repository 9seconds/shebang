package v1

import (
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/9seconds/shebang/internal/utils"
	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

const (
	// MaxVarArgs limits each explicitly configured nonnegative vararg bound.
	MaxVarArgs = math.MaxUint16
)

var (
	// ErrDuplicateLongName indicates a repeated normalized long name.
	ErrDuplicateLongName = errors.New("duplicate long name")
	// ErrDuplicateShortName indicates a repeated normalized shorthand.
	ErrDuplicateShortName = errors.New("duplicate short name")
	// ErrUnknownNode indicates an unsupported configuration node.
	ErrUnknownNode = errors.New("unknown node")
	// ErrOneArgumentExpected indicates multiple vararg declarations.
	ErrOneArgumentExpected = errors.New("only 1 vararg can be defined")
	// ErrDefineExecute indicates an empty interpreter argument list.
	ErrDefineExecute = errors.New("execute cannot be empty")
	// ErrReservedName indicates a name reserved for built-in command behavior.
	ErrReservedName = errors.New("name is reserved")
	// ErrReservedShort indicates a reserved shorthand.
	ErrReservedShort = errors.New("shorthand is reserved")
	// ErrNoValueType indicates an explicitly empty value type.
	ErrNoValueType = errors.New("value type is not defined")

	// ReName matches ASCII-alphanumeric declaration names with internal hyphens.
	ReName = regexp.MustCompile(`^[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*$`)
	// ReShort matches an ASCII-alphanumeric shorthand.
	ReShort = regexp.MustCompile(`^[a-zA-Z0-9]$`)

	// ReservedShorts contains normalized shorthands unavailable to declarations.
	ReservedShorts = map[string]bool{
		"h": true,
	}
	// ReservedNames contains normalized names unavailable to declarations.
	ReservedNames = map[string]bool{
		"help": true,
	}
)

// Parse reads KDL configuration, validates declarations, and normalizes names
// and vararg bounds. The default interpreter is bash.
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
	case "description": //nolint:goconst // Keep KDL node names explicit in parser switches.
		return setSingleArgument(&conf.Description, node)
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

//nolint:dupl // Preserve distinct option and flag child handlers and error context.
func processConfigNodeOption(conf *Config, node *document.Node) error {
	opt := Option{}

	if err := setName(&opt.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	if _, ok := conf.seenLongNames[opt.Name]; ok {
		return fmt.Errorf("%w %s", ErrDuplicateLongName, opt.Name)
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
			return fmt.Errorf("%w %s", ErrDuplicateShortName, opt.Short)
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
	case "value": //nolint:goconst // Keep KDL node names explicit in parser switches.
		return processWithValue(&opt.WithValue, node)
	}

	return ErrUnknownNode
}

//nolint:dupl // Preserve distinct option and flag child handlers and error context.
func processConfigNodeFlag(conf *Config, node *document.Node) error {
	flag := Flag{}

	if err := setName(&flag.Name, node); err != nil {
		return fmt.Errorf("cannot set a name: %w", err)
	}

	if _, ok := conf.seenLongNames[flag.Name]; ok {
		return fmt.Errorf("%w %s", ErrDuplicateLongName, flag.Name)
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
			return fmt.Errorf("%w %s", ErrDuplicateShortName, flag.Short)
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

	arg.Name = strings.ReplaceAll(arg.Name, "-", "_")

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

	arg.Name = strings.ReplaceAll(arg.Name, "-", "_")

	for _, chld := range node.Children {
		name := chld.Name.NodeNameString()
		if err := processVarArgNode(arg, chld); err != nil {
			return fmt.Errorf("cannot process vararg %s: %w", name, err)
		}
	}

	if err := normalizeVarArgBounds(arg); err != nil {
		return err
	}

	conf.VarArgs = arg

	return nil
}

func normalizeVarArgBounds(arg *VarArg) error {
	// Negative bounds have the same meaning as omitted bounds: no required
	// items for the minimum, and no limit for the maximum. Normalize before
	// comparing so an unlimited maximum accepts any nonnegative minimum.
	arg.MinCount = normalizeVarArgBound(arg.MinCount)
	arg.MaxCount = normalizeVarArgBound(arg.MaxCount)

	if arg.MinCount != nil && arg.MaxCount != nil && *arg.MinCount > *arg.MaxCount {
		return NewVarArgBoundsError(*arg.MinCount, *arg.MaxCount)
	}

	if arg.MinCount != nil && *arg.MinCount > MaxVarArgs {
		return NewVarArgLimitError("min-count", *arg.MinCount)
	}

	if arg.MaxCount != nil && *arg.MaxCount > MaxVarArgs {
		return NewVarArgLimitError("max-count", *arg.MaxCount)
	}

	return nil
}

func normalizeVarArgBound(bound *int64) *int64 {
	if bound == nil || *bound < 0 {
		return nil
	}

	return bound
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
		return ErrNoValueType
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

	*data = strings.ToLower(*data)

	if lv := utf8.RuneCountInString(*data); lv != 1 {
		return NewShortNameLengthError(lv)
	}

	if !ReShort.MatchString(*data) {
		return NewShortNamePatternError(ReShort.String())
	}

	if ReservedShorts[*data] {
		return ErrReservedShort
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

	*target = strings.ToLower(*target)

	if !ReName.MatchString(*target) {
		return NewNamePatternError(*target, ReName.String())
	}

	if ReservedNames[*target] {
		return ErrReservedName
	}

	return nil
}

func getNodeArguments[T any](node *document.Node) ([]T, error) {
	return utils.All[T](convertNodeArgsToAny(node))
}

//nolint:ireturn // Return the caller-selected generic node argument type T.
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
