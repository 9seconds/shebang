# Project overview

- The project is written in Golang. Check @go.mod to get dependencies and
  version number.
- @README.md must always be up to date. For any change check if the readme
  must be updated.

# Styling guide

Try to put field values for struct and maps on their own distinct lines.

Good:

```go
a := map[string]any{
    v: 1,
    x: 2,
}
b := Struct{
    b: 1,
    c: 1
}
```

Bad:

```go
a := map[string]any{v: 1, x: 2}
b := Struct{b: 1, c: 1}
```

This is also applicable to strings if they have more than 1 element.

# Execution chain

These diagrams describe the runtime flow and the boundaries that are useful
when implementing validators or choosing where to test a change.

## Startup and configuration

```text
main.go: main
  |
  +--> env.IsDebug -> log.Configure
  |
  +--> no arguments / standalone -h or --help
  |      `--> renderReadme -> exit
  |
  `--> getScript (absolute script path, exec.LookPath)
         -> parseConfig (open script)
         -> config.Parse
              -> config.NewReader
                   find #!shebang.N
                   extract consecutive # comment lines
              -> config/v1.Parse (versions 0 and 1)
                   parse KDL, validate declarations, normalize names/bounds
         -> cli.NewCommand(scriptPath, runExecve or runDebug)
         -> conf.Configure(cmd)
         -> cmd.Cmd.SetArgs(os.Args[2:])
         |
         +--> SHEBANG_COMPLETION is present
         |      `--> runCompletion -> print shell completion script
         |
         `--> cmd.Cmd.Execute -> validation/export/execution chain below
```

Without a configuration marker, the reader returns version 0 and an empty
document; version-one parsing supplies the default `bash` interpreter.
Configuration and interpreter resolution also run before completion-script
generation. Presence of `SHEBANG_COMPLETION`, including an empty value, selects
generation; empty or `auto` uses `SHELL` to select the shell.

## From KDL to a configured command

```text
value "datetime" format="unix" { date-after "0"; }
  |
  `--> v1.WithValue
         Type          = "datetime"
         Properties    = {"format": "unix"}
         Subvalidators = {"date-after": ["0"]}
            |
            `--> values.New(Type, Properties, Subvalidators)
                   -> NewDatetime -> values.Value

v1.Config.Configure
  -> configureDescription
  -> configureArgv              resolve interpreter using exec.LookPath
  -> configureUse               build usage line
  -> configureOptions           values.New -> cli.Option -> flag completion
  -> configurePositionalArgs    newArgValidators -> Args / ValidArgsFunction
  -> configureFlags             boolean flag bindings
```

Node properties (`name=value`) and child subvalidators have separate maps.
Parsing checks KDL structure, argument types, declaration names, duplicates,
reserved names, and vararg bounds. Validator constructors check the selected
value type and its supported properties/subvalidators during configuration.
Unknown child subvalidators use `ErrUnknownSubvalidator`; unknown node
properties use `ErrUnknownProperty` or `ErrNoProperties` as appropriate.

## Validation, export, and interpreter invocation

```text
Cobra Execute
  -> parse flags/options
       option value -> cli.Option.Set -> Value.Validate -> store original string
  -> Cmd.Args -> argValidators.Validate
       check count -> validate leading / trailing / variadic values
  -> Cmd.PreRun
       options -> SetEnv -> SHEBANG_OL_* and optional SHEBANG_OS_*
       enabled flags -> SetEnv -> SHEBANG_FL_* and optional SHEBANG_FS_*
  -> Cmd.RunE
       argv = [resolved interpreter, interpreter arguments..., script path,
               original positional arguments...]
       |
       +--> runExecve -> syscall.Exec(argv[0], argv, os.Environ())
       `--> runDebug  -> print argv and SHEBANG_* environment
```

Validation failures stop execution before `PreRun` exports configured values.
Internal conversion never replaces the original CLI string: option exports
and positional interpreter arguments preserve the input. Unset options and
disabled flags do not overwrite inherited environment variables. Environment
names uppercase names and replace hyphens with underscores.

## Positional argument assignment

```text
KDL declarations:  arg SOURCE -> vararg ITEM (min 1) -> arg DEST
CLI arguments:    [source, item1, item2, destination]
                  |       |      |      |
validators.first: SOURCE  ITEM1  |      |
validators.varArg:               ITEM2  |
validators.last:                       DEST
```

`newArgValidators` expands the required vararg minimum into `first`; the
remaining vararg items occupy the middle slice. Count validation runs before
slicing, and trailing positions are reserved from the end. During completion,
leading positions are filled first; then trailing validators are preferred
when optional varargs make the split ambiguous. Without trailing positions,
remaining words use the vararg validator. Finite capacity stops suggestions
once all positions are filled. See `internal/config/v1/args.go`.

## Value validation and interactive completion

```text
Value.Validate(raw string)
  -> baseValue.toValue(raw) -> typed input
  -> each configured subvalidator -> success or error

Interactive shell request (Cobra __complete)
  +--> option completion callback -> Value.Complete(toComplete)
  `--> positional ValidArgsFunction
         -> argValidators.Complete -> selected Value.Complete(toComplete)

baseValue.Complete
  +--> valid input -> return original input with NoFileComp
  `--> invalid input -> toComplete -> sort/deduplicate -> return directive

valueFilepath.Complete (override)
  -> split directory/prefix -> ReadDir -> prefix match
  +--> directories -> trailing separator + NoSpace for navigation
  `--> files -> Validate(candidate) -> accepted candidate paths
```

Completion-script generation and interactive completion requests are distinct
paths. Interactive requests select completion callbacks rather than executing
the script's `RunE`. Subvalidators execute in unspecified map order; `String()`
sorts their diagnostic details. Filepath completion overrides the base
valid-input shortcut so partial paths can still be expanded.

Filesystem validators use `internal/world/files.go`: `Stat` follows symlinks,
`Lstat` inspects the link, and access checks use OS access rules. `world.World`
caches successes and errors for the process; use fresh `world.Files` and `world.Users` instances in
internal cache tests and unique temporary paths in validator tests.
Git caches use fresh `world.Git` instances and temporary repositories. Tests that
change the working directory must remain serial. Run `go test -race ./internal/world`
when changing the cache helpers or concurrent Git operations.

## Verification points

- `internal/config/reader_test.go` and `internal/config/v1/parse_test.go`:
  configuration extraction, KDL structure, properties, and normalization.
- `internal/config/v1/config_test.go` and `args_test.go`: constructor wiring,
  argument assignment/counts, validation, and positional completion selection.
- `internal/values/*_test.go`: validator construction, errors, boundary values,
  diagnostic strings, and completion candidates/directives.
- `internal/cli/*_test.go`: option storage, environment exports, and execution
  callback argv. Tests with process environment changes must not run in parallel.
- `main_test.go`: subprocess coverage of the reachable end-to-end chain,
  including prevented script execution, preserved inputs, and Cobra completion.
- Format changed Go files with `gofumpt`; run `go test ./...`,
  `golangci-lint run`, and `git diff --check`. Use race checks for concurrent
  cache behavior. Prefer reachable inputs over invalid states rejected upstream.

# Naming convention for @internal/values

Name of the value is the name of the source file. For example, `xyz` value
must be stored in `xyz.go`.

This name is the stem. The name of the value struct in `xyz.go` must be
`valueXyz`. Please consult a linter for the correct casing.

All errors that are used only by this value types must be named as
`Err<STEM>...`. For example `PathDoesNotExist` error used by `valueXyz`
must be named as `ErrXyzPathDoesNotExist`.

All specific errors are stored in a separate `_errors.go` file: `xyz_errors.go`
for example above.

If some values share the same error, then stem must be omitted: just `ErrSomething`.
Such common errors must be put in file in @internal/values/errors.go

All data structures for value struct must have a stem prefix:

```go
var xyzHelper = []string{1, 2, 3}
```

Name subvalidator installation helpers `addSubvalidatorXXX`, with
`addSubvalidator` as the dispatcher. Name completion candidate generators
`genCompletions`; specialized generators may append a suffix such as `Glob`.
Filepath keeps its `Complete` override to bypass the base valid-input shortcut
and delegates candidate generation to `genCompletions`.

# Testing

These rules apply to every package, including `internal/world`, not only
`internal/values`. Use testify suites for new tests, with named table subtests
where appropriate. Keep suite subtests serial; tests that change the working
directory or process environment must also run in a serial suite.

Use table tests when possible.

Tests must be arranged as AAA pattern:
- https://semaphore.io/blog/aaa-pattern-test-automation
- https://www.fastruby.io/blog/testing/the-aaa-pattern-writing-robust-tests-for-any-project-with-confidence.html

Try to structure internals this way:


```go
func TestSomething(t *testing.T) {
    // Arrange
    ...

    // Act
    ...

    // Assert
    ...
}
```

In case of table tests with a common arrange, set these separators inside a
subtest.

Keep a result-producing call and its associated assertions together. Insert a
blank line before the next independent call/assertion group. When calls are
collected in Act and verified later in Assert, separate assertions for each
independent result with a blank line. Also separate setup or mutation assertions
(such as creating or removing a file) from the next lookup. Apply this spacing
inside fixtures, helpers, loops, and subtests as well as test methods.

Good:

```go
val, err := getValue()
suite.NotNil(val)
suite.Error(err)

other, err := another()
suite.NotNil(other)
suite.Error(err)
```

Bad:

```go
val, err := getValue()
suite.NotNil(val)
suite.Error(err)
other, err := another()
suite.NotNil(other)
suite.Error(err)
```

When calculating tests, use classes of equivalence:
- https://www.testbench.com/equivalence-class-partioning-and-boundary-value-analysis/
- https://katalon.com/resources-center/blog/equivalence-class-partitioning-guide

These tests should cover all equivalence classes.
