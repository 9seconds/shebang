# shebang

_When "Good Enough" is the pinnacle._

The goal of shebang is to give your ad hoc, homegrown scripts a decent
command-line interface. These scripts often have little documentation beyond
their source code and basic argument parsing that handles only the happy path.
Yet they are the workhorses that keep workflows running in companies of all
sizes, from tiny startups to giants with hundreds of thousands of employees.

They deserve a bit more love. shebang builds a CLI for them and validates input
against the rules you configure before running the script. Your script can
then focus on its job rather than repeating those validation checks.

Also, you will get a beautiful autocompletion out of the box.

## tl;dr

shebang is useful when:

1. You have a small script with a basic command-line interface.
2. You want to validate its inputs without building a full CLI by hand.
3. Your script uses a [shebang](https://en.wikipedia.org/wiki/Shebang_(Unix))
   line, such as `#!/usr/bin/env bash`, to tell the operating system which
   interpreter to run.

To use shebang:

1. Change the interpreter in the script's shebang line to `#!/usr/bin/env shebang`.
2. Add a KDL configuration block in the format shebang recognizes. It declares
   flags, options, positional arguments, and any supported value validators.
3. Run the script as usual. shebang parses the command line, validates values
   against the configured rules, and starts the original interpreter.
4. Your script could also generate a shell autocompletion that uses a given
   KDL configuration.

The interpreter receives the script path and positional arguments. shebang
exports configured flags and options as environment variables, so the script
can read their validated values without parsing those CLI options itself.
Validation only covers rules declared in the configuration; application-level
checks, such as whether a path exists or is a directory, remain the script's
responsibility unless an appropriate validator is configured.

## Install

Just go to [Releases](https://github.com/9seconds/shebang/releases) page and
pick up an archive for your architecture. If you use
[mise](https://mise.jdx.dev/), you can use its [GitHub
backend](https://mise.jdx.dev/dev-tools/backends/github.html) to download the
tool.

## Built-in help

Run `shebang` without arguments, or with only `-h` or `--help`, to display its
built-in help. No special environment variables are required.

```shell
shebang
shebang --help
```

The help displays the this README embedded in your installed binary, formatted
for the terminal. It reflects the documentation from that binary's build and
may differ from the latest version on GitHub.

To view a script's generated CLI help instead, run `your-script --help`.

## Build and develop

Just build and develop it as a generic Go project. It has no specifics. shebang
is the very conservative project.

If you use [mise](https://mise.jdx.dev/), you can use it to do a development:

* `mise install` will install all required dependencies
* `mise tasks run build` will build a project, generating `shebang` binary
* `mise tasks run lint` will run a linter
  (we use [golangci-lint](https://golangci-lint.run/))
* `mise tasks run test` will run tests (also, very generic tests with
  [testify](https://github.com/stretchr/testify))

## How to use it

Consider a script that archives a project directory and uploads it to a remote
machine named `dev`.

```bash
#!/usr/bin/env bash
set -eo pipefail

source_dir=$1
destination_dir=$2

archive_name=$(basename -- "$source_dir")
archive_path="${archive_name}.tar.gz"

tar -czf "$archive_path" -C "$(dirname -- "$source_dir")" "$archive_name"
rsync -rlpt "$archive_path" "dev:${destination_dir%/}/"
```
The script assumes its two arguments are present and meaningful. It has no
standard `--help` output or descriptions of the expected arguments.

Having a calculated `archive_name` is handy but sometimes not convenient. Let's
allow passing it to the script with an option:

Add `-h` and `--help` as well, so users can discover the arguments and options.
Options must appear before the positional arguments.

```bash
#!/usr/bin/env bash
set -eo pipefail

archive_name=
while (($#)); do
  case $1 in
    -h|--help)
      cat <<EOF
Usage: $0 [OPTIONS] SOURCE_DIR DESTINATION_DIR

Archive a directory and upload it to the remote host dev.

Arguments:
  SOURCE_DIR       Directory to archive
  DESTINATION_DIR  Destination directory on dev

Options:
  -a, --archive-name NAME  Archive filename without .tar.gz
                          (default: source directory's name)
  -h, --help               Show this help and exit

Example:
  $0 -a release /projects/my-project /backups
EOF
      exit 0
      ;;
    -a|--archive-name)
      archive_name=$2
      shift 2
      ;;
    --archive-name=*)
      archive_name=${1#*=}
      shift
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
  esac
done

source_dir=$1
destination_dir=$2

source_name=$(basename -- "$source_dir")
archive_name=${archive_name:-$source_name}
archive_path="${archive_name}.tar.gz"

tar -czf "$archive_path" -C "$(dirname -- "$source_dir")" "$source_name"
rsync -rlpt "$archive_path" "dev:${destination_dir%/}/"
```

Now check that exactly two positional arguments remain after parsing the
options.

```bash
#!/usr/bin/env bash
set -eo pipefail

archive_name=
while (($#)); do
  case $1 in
    -h|--help)
      cat <<EOF
Usage: $0 [OPTIONS] SOURCE_DIR DESTINATION_DIR

Archive a directory and upload it to the remote host dev.

Arguments:
  SOURCE_DIR       Directory to archive
  DESTINATION_DIR  Destination directory on dev

Options:
  -a, --archive-name NAME  Archive filename without .tar.gz
                          (default: source directory's name)
  -h, --help               Show this help and exit

Example:
  $0 -a release /projects/my-project /backups
EOF
      exit 0
      ;;
    -a|--archive-name)
      archive_name=$2
      shift 2
      ;;
    --archive-name=*)
      archive_name=${1#*=}
      shift
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
  esac
done

if (($# != 2)); then
  printf 'Expected 2 positional arguments, got %d.\n' "$#" >&2
  printf 'Usage: %s [OPTIONS] SOURCE_DIR DESTINATION_DIR\n' "$0" >&2
  printf 'Run %s --help for details.\n' "$0" >&2
  exit 2
fi

source_dir=$1
destination_dir=$2

source_name=$(basename -- "$source_dir")
archive_name=${archive_name:-$source_name}
archive_path="${archive_name}.tar.gz"

tar -czf "$archive_path" -C "$(dirname -- "$source_dir")" "$source_name"
rsync -rlpt "$archive_path" "dev:${destination_dir%/}/"
```

Let's also check that the source directory exists.

```bash
#!/usr/bin/env bash
set -eo pipefail

archive_name=
while (($#)); do
  case $1 in
    -h|--help)
      cat <<EOF
Usage: $0 [OPTIONS] SOURCE_DIR DESTINATION_DIR

Archive a directory and upload it to the remote host dev.

Arguments:
  SOURCE_DIR       Existing directory to archive
  DESTINATION_DIR  Destination directory on dev

Options:
  -a, --archive-name NAME  Archive filename without .tar.gz
                          (default: source directory's name)
  -h, --help               Show this help and exit

Example:
  $0 -a release /projects/my-project /backups
EOF
      exit 0
      ;;
    -a|--archive-name)
      archive_name=$2
      shift 2
      ;;
    --archive-name=*)
      archive_name=${1#*=}
      shift
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
  esac
done

if (($# != 2)); then
  printf 'Expected 2 positional arguments, got %d.\n' "$#" >&2
  printf 'Usage: %s [OPTIONS] SOURCE_DIR DESTINATION_DIR\n' "$0" >&2
  printf 'Run %s --help for details.\n' "$0" >&2
  exit 2
fi

source_dir=$1
destination_dir=$2

if [[ ! -d $source_dir ]]; then
  printf 'Source directory does not exist or is not a directory: %s\n' "$source_dir" >&2
  exit 1
fi

source_name=$(basename -- "$source_dir")
archive_name=${archive_name:-$source_name}
archive_path="${archive_name}.tar.gz"

tar -czf "$archive_path" -C "$(dirname -- "$source_dir")" "$source_name"
rsync -rlpt "$archive_path" "dev:${destination_dir%/}/"
```

The script is now useful, but its boilerplate hides the actual work. Even if
an LLM generates it for you, you still have to maintain the argument parsing,
help text, and validation.

Now let's see how shebang could help.

```bash
#!/usr/bin/env shebang

#!shebang.1
# execute "bash" "-e" "-o" "pipefail"
# description "Archive a directory and upload it to the remote host dev."
# option "archive-name" {
#   short "a"
#   description "Archive filename without .tar.gz (default: source directory's name)"
#   value "str" {
#     min-length 1
#     re "[a-z]+"
#   }
# }
# arg "source-dir" {
#   description "Existing directory to archive"
#   value "directory"
# }
# arg "destination-dir" {
#   description "Destination directory on dev"
# }

source_dir=$1
destination_dir=$2

source_name=$(basename -- "$source_dir")
archive_name=${SHEBANG_OS_A:-$source_name}
archive_path="${archive_name}.tar.gz"

tar -czf "$archive_path" -C "$(dirname -- "$source_dir")" "$source_name"
rsync -rlpt "$archive_path" "dev:${destination_dir%/}/"
```

shebang supplies `-h` and `--help`, parses `--archive-name` and `-a`, and checks
the positional argument count.

Everything is defined and described by a special comment that is a part of the
script. It generates a full CLI for you, and validates arguments and options.

The value of an archive name is propagated as `SHEBANG_OS_A` or
`SHEBANG_OL_ARCHIVE_NAME` environment variables, already preparsed and
validated.

Hyphens remain in long option and flag names, such as `--archive-name`.
Environment variable names replace hyphens with underscores and use uppercase:
`--archive-name` exports `SHEBANG_OL_ARCHIVE_NAME`. Positional names follow the
same convention in help and diagnostics: `source-dir` is displayed as
`SOURCE_DIR`. Positional argument values are passed unchanged.

## Embedding KDL configuration in a script

Keep `#!/usr/bin/env shebang` as the first line of the script. This tells the
operating system to run shebang.

Add a separate `#!shebang.1` marker to introduce the version-one configuration,
then prefix every line of the KDL document with `#`:

```bash
#!/usr/bin/env shebang

#!shebang.1
# execute "bash" "-e" "-o" "pipefail"
# description "An example script."
#
# option "archive-name" {
#   short "a"
#   description "Archive filename without .tar.gz"
# }
#
# arg "source-dir" {
#   description "Directory to archive"
# }

# The script body begins here.
printf 'Source: %s\n' "$1"
```

The two markers have different roles: `#!/usr/bin/env shebang` selects the
program that starts the script, while `#!shebang.1` selects the configuration
format. `shebang.1` means that the format version 1 is used. When we introduce
version 2, it is going to be `shebang.2`.

The configuration block must start with a marker. In that case shebang will
treat the rest of the comment block as commented KDL document. End of the
consecutive comments mean the end of the configuration block.

#### Version 1

[The full reference with many comments](https://github.com/9seconds/shebang/blob/master/scheme.v1.kdl).

This configuration is defined by [KDL](https://kdl.dev/), a tiny XML-like
configuration like that you probably seen in [Zellij](https://zellij.dev/), or
[Ferron](https://ferron.sh/). It looks and feels like
[nginx](https://nginx.org/) configuration format but more well-defined and a
bit more polished.

For a practical example, imagine `sync-files`, a wrapper around `rsync` that
copies selected paths from a source directory to a local or remote destination.
The wrapper runs `rsync` from the source directory and preserves the relative
paths of the selected files and directories.

Its command line has one required leading argument, a group of one or more
paths, and one required trailing argument:

```text
sync-files [flags] [--] SOURCE_DIR PATH1 [PATH2 ...] DESTINATION
```

Here is its KDL configuration. To embed it in a Bash script, place it after
`#!shebang.1` and prefix every line with `#`, as described above.

```kdl
execute "bash" "-eu" "-o" "pipefail"

description """
  Copy selected files and directories with rsync.

  Paths are relative to SOURCE_DIR. Preserve their directory structure at
  DESTINATION, which may be a local directory or an rsync destination such as
  backup@example.org:/srv/backups/.

  Requires rsync and, for remote copies, SSH access to the destination host.

  Examples:
  sync-files --archive /projects/my-app src assets /backups/my-app/
  sync-files -a -n -v /projects/my-app src assets backup@example.org:/srv/backups/
  sync-files --exclude '*.tmp' --bandwidth 2048 /projects/my-app logs data /backups/
  """

flag "archive" {
  description "Preserve permissions, timestamps, symlinks, and other archive attributes"
  short "a"
}

flag "dry-run" {
  description "Show what would be copied without changing the destination"
  short "n"
}

flag "verbose" {
  description "List transferred files and show a transfer summary"
  short "v"
}

option "exclude" {
  description "Exclude files matching an rsync pattern; quote shell wildcards"
  short "x"
  value "str" {
    min-length 1
  }
}

option "bandwidth" {
  description "Limit transfer bandwidth in KiB/s; 0 means unlimited"
  short "b"
  value "str" {
    re "^[0-9]+$"
  }
}

arg "source-dir" {
  description "Base directory containing the selected paths"
  value "str" {
    min-length 1
  }
}

vararg "path" {
  description "Files or directories to copy, relative to SOURCE_DIR"
  min-count 1
  value "str" {
    min-length 1
  }
}

arg "destination" {
  description "Local destination directory or USER@HOST:PATH"
  value "str" {
    min-length 1
  }
}
```

Declaration order determines how positional arguments are assigned. In the
first example, `/projects/my-app` is `SOURCE_DIR`, `src` and `assets` belong to
the `PATH` group, and `/backups/my-app/` is `DESTINATION`. One `vararg` node
handles all the selected paths; the final `arg` reserves the last argument for
the destination.

Let's look at each configuration node in more detail.

##### `execute` field

The `execute` node specifies the interpreter used to run your script. Its
first argument is the executable name or path; any remaining arguments are
passed to that interpreter before the script path. If you omit `execute`,
shebang uses `bash` without additional interpreter arguments.

For example, suppose `/usr/local/bin/example` contains:

```bash
#!/usr/bin/env shebang
#!shebang.1
# execute "bash"

echo Hello
```

Running `/usr/local/bin/example` makes shebang find `bash` on `PATH` and start
it with the script path as its argument. Conceptually, this is equivalent to:

```sh
bash /usr/local/bin/example
```

You can also configure interpreter arguments:

```kdl
execute "bash" "-eu" "-o" "pipefail"
```

In that case, the interpreter invocation is equivalent to:

```sh
bash -eu -o pipefail /usr/local/bin/example
```

This serves a purpose similar to `env -S`: it lets you specify an interpreter
and multiple arguments without relying on the operating system's handling of
arguments in a shebang line. Each KDL string is a separate argument. shebang
does not split strings on spaces or expand shell variables, so write
`"bash" "-eu"` rather than `"bash -eu"`.

An executable name such as `"bash"` is resolved through `PATH`; an explicit
path such as `"/usr/local/bin/bash"` selects that executable directly. shebang
reports an error if it cannot resolve the interpreter. Any positional script
arguments follow the script path, while parsed flags and options are exported
as environment variables.

##### `description` field

In order to provide a description to your command, you can use `description`
parameter:

```kdl
description "Some description"
```

Please keep in mind that KDL natively supports multiline strings:

```kdl
description """
  Copy selected files and directories with rsync.

  Paths are relative to SOURCE_DIR. Preserve their directory structure at
  DESTINATION, which may be a local directory or an rsync destination such as
  backup@example.org:/srv/backups/.

  Requires rsync and, for remote copies, SSH access to the destination host.

  Examples:
  sync-files --archive /projects/my-app src assets /backups/my-app/
  sync-files -a -n -v /projects/my-app src assets backup@example.org:/srv/backups/
  sync-files --exclude '*.tmp' --bandwidth 2048 /projects/my-app logs data /backups/
  """
```

Do not forget to add some usage examples. This can help AI agents to use your
ad hoc script more efficiently.

##### `flag` node

Flags are boolean command-line switches. Unlike value-taking options, they do
not require a separate value: including a flag enables its behavior. For
example:

```shell
$ cp -a one two
```

Here, `-a` enables archive mode for `cp`; `one` and `two` are positional
arguments, not values for the flag.

Declare a flag in shebang with a `flag` node:

```kdl
flag "archive" {
  description "Preserve permissions, timestamps, symlinks, and other archive attributes"
  short "a"
}
```

The node takes exactly one argument: the flag's long name. In this example,
`"archive"` defines `--archive`. The optional child nodes inside the braces
provide its help text and a short spelling:

| Property      | Example                                 | Description                            |
| ------------- | --------------------------------------- | -------------------------------------- |
| `description` | `description "Preserve file attributes"` | Help text. Defaults to an empty string. |
| `short`       | `short "a"`                             | Optional short form, such as `-a`.      |

The short name must be a single ASCII letter or digit. Omit `short` if the flag
does not need a short form.

Both `--archive` and `-a` enable the same flag. Long names and short names must
be unique across flags and value-taking options. Names are normalized to
lowercase, and `help` and `h` are reserved for built-in help.

When enabled, this flag exports `SHEBANG_FL_ARCHIVE=true` and
`SHEBANG_FS_A=true` before the interpreter starts. These variables are the same
regardless of which spelling the user chooses. The flag itself is removed from
the script's positional arguments.

The prefixes `FL` and `FS` stand for "flag long" and "flag short", respectively.
You can read either `SHEBANG_FL_ARCHIVE` or `SHEBANG_FS_A`; both represent the
same flag. If no short name is configured, only the long-name variable is
exported.

Enabled flags export the value `true`. If a flag is omitted or explicitly set
to false, shebang does not export a value for it. Any existing value inherited
from the environment remains unchanged.

##### `option` node

An option is a command-line switch that takes a value. For example:

```shell
curl -o response.json https://example.org/api
```

Here, `-o` takes the output filename, `response.json`, where curl saves the
response body.

Declare an option in shebang with an `option` node:

```kdl
option "exclude" {
  description "Exclude files matching an rsync pattern; quote shell wildcards"
  short "x"
  value "str" {
    min-length 1
  }
}
```

As with flags, the node's argument defines its long name: `"exclude"` creates
`--exclude`. The optional `short` child adds a short spelling, `-x` in this
example. Both spellings accept the same value.

Unlike a flag, an option can have a `value` child that selects its value type
and validation rules. Here, `value "str"` with `min-length 1` requires a
nonempty string. Value types also determine shell completion behavior. If
`value` is omitted, the option accepts an unconstrained string. See the value
types section below for details.

> [!IMPORTANT]
> Flags and options share their name namespaces. Each long name and each short
> name must be unique across both kinds of declarations. For example, a flag
> and an option cannot both use `--exclude` or `-x`. Names are normalized to
> lowercase, so changing capitalization does not make a name unique.

##### `arg` node

An argument is a required positional value. Unlike flags and options, it has
no command-line switch: its position determines its meaning. For example:

```shell
cp source.txt destination.txt
```

Here, `source.txt` and `destination.txt` are positional arguments. Declare each
required position with an `arg` node:

```kdl
arg "source-dir" {
  description "Base directory containing the selected paths"
  value "str" {
    min-length 1
  }
}
```

The node takes exactly one argument: the name shown in help and validation
errors. `"source-dir"` is displayed as `SOURCE_DIR`; it does not create a
`--source-dir` option. Names are normalized to lowercase internally, with
hyphens replaced by underscores.

| Property      | Example                         | Description                                      |
| ------------- | ------------------------------- | ------------------------------------------------ |
| `description` | `description "Source directory"` | Help text. Defaults to an empty string.           |
| `value`       | `value "str" { min-length 1; }`  | Optional value type, validation, and completion.  |

If `value` is omitted, the argument accepts an unconstrained string. Each
`arg` consumes exactly one positional value. Without a `vararg`, declaration
order defines the complete positional argument order, and extra or missing
values are rejected.

Arguments declared before a `vararg` are taken from the start of the positional
values. Arguments declared after it are taken from the end, in declaration
order. The interpreter receives the original positional values unchanged;
argument names are labels, not environment variable names.

##### `vararg` node

A variadic argument group consumes a variable number of positional values.
Use it when a command accepts several files, paths, or other items:

```kdl
vararg "path" {
  description "Files or directories to copy, relative to SOURCE_DIR"
  min-count 1
  max-count 10
  value "str" {
    min-length 1
  }
}
```

The node takes exactly one argument: the group's name. This example accepts
between one and ten paths and requires each path to be a nonempty string. The
name follows the same normalization rules as an `arg` name.

| Property      | Example                         | Description                                      |
| ------------- | ------------------------------- | ------------------------------------------------ |
| `description` | `description "Paths to copy"`   | Help text for the group.                         |
| `value`       | `value "str" { min-length 1; }`  | Optional validator applied to every item.         |
| `min-count`   | `min-count 1`                   | Minimum number of items; omitted or negative means zero. |
| `max-count`   | `max-count 10`                  | Maximum number of items; omitted or negative means unlimited. |

Both `min-count` and `max-count` are optional.

For example, combine a leading argument, a variadic group, and a trailing
argument:

```kdl
arg "source-dir"
vararg "path" {
  min-count 1
}
arg "destination"
```

For this invocation:

```shell
sync-files /projects/my-app src assets /backups/my-app/
```

`/projects/my-app` is `SOURCE_DIR`, `src` and `assets` belong to `PATH`, and
`/backups/my-app/` is `DESTINATION`. The trailing argument is reserved before
the remaining values are assigned to the variadic group. The group's bounds
count only its own items, not the leading or trailing arguments.

> [!IMPORTANT]
> A configuration can contain only one `vararg` node. Declare it between any
> required leading and trailing `arg` nodes. It consumes the values left after
> those fixed positions are assigned, so it can represent many items without
> requiring multiple variadic declarations.

## Shell completion

shebang can generate completion scripts for Bash, Zsh, Fish, and PowerShell.

Make your script executable and put it on `PATH` before loading its
completions. The examples below assume that `sync-files` is available on
`PATH` and uses `#!/usr/bin/env shebang`.

### Generating completions

Set `SHEBANG_COMPLETION` for a single invocation to choose the output format:

```shell
SHEBANG_COMPLETION=bash sync-files
```

This prints a completion script instead of running the script body. The
configuration is still parsed and the interpreter must be available, but you
do not need to supply the script's required positional arguments.

| Value  | Shell      |
| ------ | ---------- |
| `bash` | Bash       |
| `zsh`  | Zsh        |
| `fish` | Fish       |
| `pwsh` | PowerShell |

An absolute shell path, such as `/bin/bash`, is also accepted. Set the value
to `auto` to autodetect.

> [!IMPORTANT]
> Do not export `SHEBANG_COMPLETION` globally. Its presence selects completion
> generation, even when its value is empty. Set it only while generating the
> completion script; normal execution and interactive completion requests
> should run without it.

### Loading completions

#### Bash

Load completions into the current session:

```bash
source <(SHEBANG_COMPLETION=bash sync-files)
```

#### Zsh

Initialize Zsh's completion system if your shell configuration has not already
done so, then load the generated script:

```zsh
autoload -Uz compinit
compinit
source <(SHEBANG_COMPLETION=zsh sync-files)
```

#### Fish

```fish
env SHEBANG_COMPLETION=fish sync-files | source
```

#### PowerShell

In a PowerShell session on a system that can execute the script, temporarily
set the environment variable and restore it before loading the completions:

```powershell
$previousCompletion = $env:SHEBANG_COMPLETION
try {
    $env:SHEBANG_COMPLETION = 'pwsh'
    $completionScript = sync-files | Out-String
} finally {
    if ($null -eq $previousCompletion) {
        Remove-Item Env:SHEBANG_COMPLETION -ErrorAction SilentlyContinue
    } else {
        $env:SHEBANG_COMPLETION = $previousCompletion
    }
}
$completionScript | Invoke-Expression
```

These commands affect only the current session. To load completions on
startup, save the generated output to a file and source it from your shell's
startup configuration or PowerShell profile. Fish also loads files from
`~/.config/fish/completions/`; save this script as `sync-files.fish` there.
Regenerate the file when you change your script's CLI configuration.

## Values

A `value` node selects how shebang validates an option or positional argument
and completes its value in the shell. Validation does not change the input:
the script receives the original string, without conversion or normalization.

The `value` node is optional for `option`, `arg`, and `vararg`. If omitted,
the value is treated as an unconstrained string. For a variadic group, the
configured validator applies to each item separately.

Specify the type as the node's argument and any additional constraints as
child nodes, called _subvalidators_:

```kdl
value "str" {
  min-length 1
  max-length 20
  re "^[a-z]+$"
}
```

This definition accepts lowercase ASCII words between 1 and 20 characters
long: `"hello"` passes, while `""`, `"Hello"`, and `"hello123"` do not.

Validation checks the value type, then applies the configured subvalidators.
Every constraint must pass; their execution order is not guaranteed. For
example, `min-length` and `max-length` check the number of Unicode code points,
while `re` checks a regular expression.

Constraints are configured _independently_, so contradictory bounds can be
declared:

```kdl
value "str" {
  min-length 10
  max-length 5
}
```

The configuration is accepted, but no string can satisfy both constraints.
Make sure your constraints allow the values you intend to accept.

### `str` value

`str` accepts strings. Without subvalidators, every command-line value passes,
including an empty string. It provides no completion suggestions and disables
automatic file completion.

| Subvalidator | Example definition | Passes                    | Fails                                   | Description                                                  |
| ------------ | ------------------ | ------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| `min-length` | `min-length 6`     | `"привет"`, `"welcome"`     | `"hello"`, `""`                           | Requires at least the specified number of Unicode code points. |
| `max-length` | `max-length 6`     | `"привет"`, `"hello"`, `""` | `"welcome"`                             | Allows at most the specified number of Unicode code points.   |
| `re`         | `re "^[a-z]+$"`    | `"hello"`, `"archive"`      | `"Hello"`, `"hello123"`, `"привет"`, `""` | Requires a match for the supplied regular expression.         |

Length bounds must be nonnegative integers. Zero is allowed: `min-length 0`
places no restriction on length, while `max-length 0` accepts only an empty
string. `"привет"` contains six Unicode code points, even though its UTF-8
encoding occupies twelve bytes. Code points are not necessarily the same as
visually displayed characters, especially when combining marks are used.

The `re` subvalidator uses Go's regular-expression syntax. Matching is not
implicitly anchored: `re "hello"` also accepts `"say hello"`. Use `^` and `$`,
as in the table, when the entire value must match. Each subvalidator takes
exactly one argument; unknown subvalidators and invalid regular expressions
are rejected while configuring the command.

#### Example

```kdl
value "str" {
  min-length 5
  max-length 1024
  re "^pod-[0-9]-.*$"
}
```

This definition requires a string between 5 and 1,024 Unicode code points
long that matches the entire pattern: `pod-`, one ASCII digit, another hyphen,
and an optional suffix.

For example, `"pod-1-xx"` and `"pod-1-"` pass. `"pod-12-xx"` fails because
`[0-9]` matches exactly one digit, and `"prefix-pod-1-xx"` fails because the
pattern is anchored at the start. A matching string longer than 1,024 code
points fails the length constraint.
