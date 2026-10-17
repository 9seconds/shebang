# shebang

[![codecov](https://codecov.io/gh/9seconds/shebang/graph/badge.svg?token=6LUVKG9X2L)](https://codecov.io/gh/9seconds/shebang)

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
#   value "filepath" {
#     type "directory"
#   }
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
  value "int" {
    min 0
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

### `int` value

`int` accepts signed decimal integers within the int64 range. Without
subvalidators, any 64 bit integer passes. It provides no completion suggestions
and disables automatic file completion.

An optional `+` or `-` sign and leading zeros are accepted: `"12"`, `"+12"`,
`"0012"`, and `"-12"` are valid integers. Leading zeros do not select octal
notation.

| Subvalidator | Example definition | Passes                | Fails          | Description                            |
| ------------ | ------------------ | --------------------- | -------------- | -------------------------------------- |
| `min`        | `min 0`            | `"0"`, `"12"`, `"+12"` | `"-1"`, `"-12"` | Requires a value at least this minimum. |
| `max`        | `max 10`           | `"10"`, `"0"`, `"-12"` | `"11"`, `"12"`  | Allows a value at most this maximum.    |

Both bounds are inclusive and optional. Each takes exactly one integer within
the int64 range; negative bounds and zero are allowed. For example, `min -10`
is valid, but `min "-10"` and `min 1.5` are not. Omitting a bound leaves that
side unrestricted within the int64 range. Equal bounds accept only that numeric
value; reversed bounds are accepted as configuration, but no input can satisfy
both.

Validation parses the number internally, but preserves the original string
passed to the script. For example, `"+0012"` is validated as 12 and remains
`"+0012"` in positional arguments or an option's environment variables.

> [!IMPORTANT]
> Negative positional integers can look like command-line flags. Place `--`
> before positional arguments to end flag parsing, for example
> `your-script -- -12`. For a value-taking option, use `--offset=-12` or
> `--offset -12`.

#### Example

```kdl
option "retries" {
  description "Number of retry attempts, from 0 to 10"
  short "r"
  value "int" {
    min 0
    max 10
  }
}
```

This definition accepts `--retries 0`, `--retries 10`, and `-r +03`.
`--retries 11` and `--retries -1` fail the bounds, while `--retries 1.5`
fails integer parsing. The script reads the original value from
`SHEBANG_OL_RETRIES` or `SHEBANG_OS_R`.

### `float` value

`float` accepts numbers float numbers in [IEEE
754](https://en.wikipedia.org/wiki/IEEE_754) notation: this is a general way
of specifying such numbers almost everywhere.

It is very similar to `int`, even the meaning of subvalidators is the same.

#### Example

```kdl
option "ratio" {
  description "Sampling ratio, from 0.0 to 1.0"
  short "r"
  value "float" {
    min 0.0
    max 1.0
  }
}
```

This definition accepts `--ratio 0`, `--ratio 1.0`, and `-r 5e-1`.
`--ratio -0.1` and `--ratio 1.1` fail the bounds; `--ratio NaN` is rejected
even though comparisons with NaN would otherwise bypass those bounds. The
script reads the original value from `SHEBANG_OL_RATIO` or `SHEBANG_OS_R`.

### `ip` value

`ip` accepts literal IPv4 and IPv6 addresses.
Examples include `192.0.2.1`, `2001:db8::1`, and `::ffff:192.0.2.1`.
IPv6 zone identifiers, such as `fe80::1%eth0`, are also accepted. IPv6 address
syntax is described in [RFC 4291, section 2.2](https://www.rfc-editor.org/rfc/rfc4291#section-2.2).

| Subvalidator          | Example definition          | Passes         | Fails          | Description |
| --------------------- | --------------------------- | -------------- | -------------- | ----------- |
| `type`                | `type "v4"`                 | `192.0.2.1`    | `2001:db8::1`  | Restricts the address family. |
| `subnets`             | `subnets "192.0.2.0/24"`     | `192.0.2.42`   | `198.51.100.1` | Requires membership in at least one supplied CIDR prefix. |
| `global-unicast`      | `global-unicast #true`      | `192.0.2.1`    | `127.0.0.1`    | Requires global-unicast classification. |
| `iflocal-multicast`   | `iflocal-multicast #true`   | `ff01::1`      | `ff02::1`      | Requires IPv6 interface-local multicast scope. |
| `linklocal-unicast`   | `linklocal-unicast #true`   | `fe80::1`      | `2001:db8::1`  | Requires a link-local unicast address. |
| `linklocal-multicast` | `linklocal-multicast #true` | `224.0.0.1`    | `239.1.1.1`    | Requires link-local multicast scope. |
| `loopback`            | `loopback #true`            | `::1`          | `2001:db8::1`  | Requires a loopback address. |
| `multicast`           | `multicast #true`           | `ff02::1`      | `2001:db8::1`  | Requires a multicast address. |
| `private`             | `private #true`             | `10.1.2.3`     | `192.0.2.1`    | Requires RFC 1918 IPv4 or RFC 4193 IPv6 private addressing. |
| `unspecified`         | `unspecified #true`         | `0.0.0.0`      | `192.0.2.1`    | Requires an unspecified address (`0.0.0.0` or `::`). |

Each classifier takes exactly one boolean. Use `#false` to exclude that
classification, `#true` - to include, or omit the child to impose no
restriction.

`global-unicast` is an address classification, not a public-address or
reachability check: private addresses can also be global unicast.

The `type` subvalidator accepts one of these strings:

| Type         | Accepted addresses |
| ------------ | ------------------ |
| `v4`         | IPv4 only. |
| `v6`         | IPv6, including IPv4-mapped IPv6. |
| `v4-in-v6`   | IPv4-mapped IPv6 only, such as `::ffff:192.0.2.1`. |
| `v6-only`    | IPv6 excluding IPv4-mapped IPv6. |

`subnets` takes one or more quoted CIDR prefixes. The prefixes are alternatives,
so an address needs to match only one. Host bits are masked: `192.0.2.19/24`
represents `192.0.2.0/24`. If `subnets` is omitted, there is no subnet restriction;
an explicitly empty subnet list is rejected.

> [!IMPORTANT]
> IPv4 and IPv4-mapped IPv6 are not automatically converted into one another
> for subnet matching. `::ffff:192.0.2.1` does not match `192.0.2.0/24`; use a
> mapped prefix such as `::ffff:192.0.2.0/120` instead. Scoped IPv6 addresses
> containing a zone identifier do not match CIDR prefixes.

Completion suggests the base addresses of configured subnets, in compressed
and expanded forms where applicable. Suggestions are filtered by the remaining
constraints and the text already typed, deduplicated, and sorted. It does not
enumerate all addresses in a subnet. Without subnets, it provides no address
suggestions and disables file completion.

#### Example

```kdl
option "peer" {
  description "Private IPv4 peer in an approved network"
  short "p"
  value "ip" {
    type "v4"
    private #true
    loopback #false
    subnets "10.0.0.0/8" "192.168.0.0/16"
  }
}
```

This accepts `--peer 10.1.2.3` and `-p 192.168.1.10`. `172.16.1.10` is private
but fails the subnet restriction; `2001:db8::1` fails the family restriction.
The script reads the original value from `SHEBANG_OL_PEER` or `SHEBANG_OS_P`.

### `subnet` value

`subnet` accepts IPv4 and IPv6 CIDR prefixes, such as `192.0.2.0/24` or
`2001:db8::/32`. Prefix lengths range from 0 to 32 for IPv4 and 0 to 128 for
IPv6. Bare addresses, dotted netmasks, zone identifiers, and surrounding
whitespace are rejected.

Host bits are accepted: `192.0.2.19/24` represents the same network as
`192.0.2.0/24` for overlap checks.

| Subvalidator | Example definition | Description |
| ------------ | ------------------ | ----------- |
| `type` | `type "v4"` | Restricts the input address family. |
| `overlaps` | `overlaps "10.0.0.0/8" "192.168.0.0/16"` | Requires overlap with at least one listed subnet. |
| `not-overlaps` | `not-overlaps "10.1.0.0/16"` | Rejects overlap with any listed subnet. |

`type` takes one of `"v4"`, `"v6"`, `"v4-in-v6"`, or `"v6-only"`, with the same
address-family meanings as the `ip` value. Both overlap children take one or
more quoted CIDR prefixes; empty lists and malformed prefixes are rejected
during configuration. Node properties are not supported.

Overlap means sharing any address, not necessarily containment: both a smaller
subnet inside a listed subnet and a larger subnet containing it overlap. For
example, `0.0.0.0/0` overlaps every IPv4 subnet. _All configured constraints
must pass._ IPv4 and IPv4-mapped IPv6 prefixes are separate address families
for overlap checks; they are not automatically converted into one another.

#### Example

```kdl
option "network" {
  description "IPv4 subnet overlapping an approved network"
  short "n"
  value "subnet" {
    type "v4"
    overlaps "10.0.0.0/8" "192.168.0.0/16"
    not-overlaps "10.1.0.0/16"
  }
}
```

This accepts `--network 192.168.1.19/24` and `--network 10.2.0.0/16`, but
rejects `10.0.0.0/8` because it also overlaps the excluded subnet. The script
reads the unchanged value from `SHEBANG_OL_NETWORK` or `SHEBANG_OS_N`.

### `port` value

`port` accepts unsigned decimal port numbers from 0 through 65,535.

The range classifications follow
[RFC 6335, section 6](https://www.rfc-editor.org/rfc/rfc6335#section-6):

| Subvalidator | Example definition | Passes             | Fails              | Description |
| ------------ | ------------------ | ------------------ | ------------------ | ----------- |
| `well-known` | `well-known #true` | `"0"`, `"80"`       | `"1024"`, `"8080"`   | System (well-known) ports: 0–1,023. |
| `registered` | `registered #true` | `"1024"`, `"8080"`  | `"80"`, `"49152"`    | User (registered) ports: 1,024–49,151. |
| `ephemeral`  | `ephemeral #true`  | `"49152"`, `"65535"` | `"80"`, `"49151"`   | Dynamic/private ports: 49,152–65,535. |

Each subvalidator takes exactly one boolean. `#true` requires membership
in the range; `#false` excludes it. Omit a subvalidator to impose no restriction
on that range. All configured constraints must pass. For example, combining
`well-known #false` and `ephemeral #false` allows only registered-range ports.
Requiring two distinct ranges with `#true` accepts no value.

#### Example

```kdl
option "port" {
  description "Listening port in the registered range"
  short "p"
  value "port" {
    registered #true
  }
}
```

This accepts `--port 8080` and `-p 08080`. `--port 80` fails the range
constraint, and `--port 65536` fails parsing. The script reads the original
value from `SHEBANG_OL_PORT` or `SHEBANG_OS_P`.

### `enum` value

`enum` accepts one of a configured set of strings. Matching is exact and
case-sensitive: a choice of `"prod"` accepts `"prod"`, but not `"Prod"`,
`"pro"`, or `" prod "`.

| Subvalidator | Example definition           | Passes          | Fails               | Description |
| ------------ | ---------------------------- | --------------- | ------------------- | ----------- |
| `choices`    | `choices "dev" "stage" "prod"` | `"dev"`, `"prod"` | `"preview"`, `"Prod"` | Lists the permitted strings. |

Each choice must be a quoted string.

#### Example

```kdl
option "environment" {
  description "Deployment environment"
  short "e"
  value "enum" {
    choices "dev" "stage" "prod"
  }
}
```

The script reads the selected value from `SHEBANG_OL_ENVIRONMENT` or
`SHEBANG_OS_E`.





### `datetime` value

`datetime` accepts dates and times. By default, it parses ISO 8601 strings,
such as `2026-10-12`, `2026-10-12T13:14:15Z`, or
`2026-10-12T13:14:15.123+02:00`.

Set the optional `format` [property](https://kdl.dev/spec/#name-property) on
the `value` node to select a different parser. Properties use `name=value`
syntax.

```kdl
value "datetime" format="unix" {
  date-after "0"
}
```

The format names are case-sensitive, and `format` must be a string.

| Format | Example input | Description |
| ------ | ------------- | ----------- |
| `iso8601` | `2026-10-12T13:14:15Z` | [ISO 8601](https://en.wikipedia.org/wiki/ISO_8601); the default. |
| `unix` | `1791810855` | Signed decimal [seconds since the Unix epoch](https://en.wikipedia.org/wiki/Unix_time). |
| `unix-ms` | `1791810855123` | Signed decimal milliseconds since the Unix epoch. |
| `rfc822` | `12 Oct 26 13:14 UTC` | [RFC 822](https://www.rfc-editor.org/info/rfc822/#section-5) with a named timezone. |
| `rfc822z` | `12 Oct 26 13:14 +0200` | RFC 822 with a numeric timezone. |
| `rfc850` | `Monday, 12-Oct-26 13:14:15 UTC` | [RFC 850](https://www.rfc-editor.org/info/rfc850/#section-2.1.4). |
| `rfc1123` | `Mon, 12 Oct 2026 13:14:15 UTC` | [RFC 1123](https://www.rfc-editor.org/info/rfc1123/#section-5.2.14) with a named timezone. |
| `rfc1123z` | `Mon, 12 Oct 2026 13:14:15 +0200` | RFC 1123 with a numeric timezone. |
| `rfc3339` | `2026-10-12T13:14:15Z` | [RFC 3339](https://www.rfc-editor.org/info/rfc3339/). |
| `rfc3339nano` | `2026-10-12T13:14:15.123456789Z` | RFC 3339 with fractional seconds. |
| `Y-m-d` | `2026-10-12` | Year-month-day. |
| `d.m.Y` | `12.10.2026` | Day.month.year. |
| `H:M:S` | `13:14:15` | Hours:minutes:seconds. |
| `H:M` | `13:14` | Hours:minutes. |
| `M:S` | `14:15` | Minutes:seconds. |
| `custom:2006/01/02 15:04` | `2026/10/12 13:14` | A custom Go time layout after `custom:`. |

[Custom layouts](https://pkg.go.dev/time#Layout) use Go's reference time, `Mon
Jan 2 15:04:05 MST 2006`: `2006` represents the year, `01` the month, `02` the
day, `15` the hour, `04` the minute, and `05` the second. Quote command-line
values containing spaces. Prefer numeric timezone offsets or `Z` when comparing
timestamps from different timezones. Layouts without a timezone are parsed in
UTC; time-only layouts use year 0, January 1 for their date.

| Subvalidator | Example definition | Passes | Fails | Description |
| ------------ | ------------------ | ------ | ----- | ----------- |
| `before` | `before "1h"` | A timestamp 30 minutes from now. | A timestamp 2 hours from now. | Requires a time before now plus the duration. |
| `after` | `after "-1h"` | A timestamp 30 minutes ago. | A timestamp 2 hours ago. | Requires a time after now plus the duration. |
| `date-before` | `date-before "2026-11-01T00:00:00Z"` | `2026-10-12T00:00:00Z` | `2026-11-01T00:00:00Z` | Requires a time before the supplied timestamp. |
| `date-after` | `date-after "2026-10-01T00:00:00Z"` | `2026-10-12T00:00:00Z` | `2026-10-01T00:00:00Z` | Requires a time after the supplied timestamp. |

Each subvalidator takes exactly one quoted string. All bounds are strict:
a timestamp equal to a bound fails. Absolute bounds (`date-before` and
`date-after`) use the selected `format`, so with `format="unix"` write
`date-after "0"` rather than an ISO 8601 date.

Relative bounds (`before` and `after`) use durations:

> A duration string is a possibly signed sequence of decimal numbers, each with
> optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
> Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d", "w",
> "y". Units "day", "week" and "year" are all approximate units with the values
> of "24h" in a day, "7d" in a week, "365d" in a year.
>
> `"-1h"`, or `"30m"`. The cutoff is calculated when the validator is
> constructed: positive durations move it into the future and negative
> durations into the past. It is not recalculated for each input. All
> configured bounds must pass.

Completion suggests the current time and configured cutoff timestamps in
the selected format.

#### Example

```kdl
option "scheduled-at" {
  description "Schedule a job within the next 24 hours"
  short "s"
  value "datetime" format="rfc3339" {
    after "0s"
    before "24h"
  }
}
```

This accepts a timestamp strictly after command configuration and less than
24 hours ahead, for example `--scheduled-at 2026-10-12T14:00:00Z` if the
command is configured at `2026-10-12T13:00:00Z`. The script reads the original
value from `SHEBANG_OL_SCHEDULED_AT` or `SHEBANG_OS_S`.

### `filepath` value

`filepath` accepts nonempty filesystem paths. Relative paths are checked from
the command's working directory; the script receives the original string.
Spaces and Unicode in filenames are preserved.

The `glob` child accepts a filesystem glob, including recursive `**` patterns.
For example, `glob "**/*.json"` restricts paths to matching JSON filenames.
With a glob configured, completion returns matching paths relative to the
current directory, filtered by the typed prefix and all configured validators.
Both relative and absolute glob patterns produce relative completion paths.

| Subvalidator | Example definition | Passes | Fails | Description |
| ------------ | ------------------ | ------ | ----- | ----------- |
| `glob` | `glob "**/*.json"` | `config.json` | `config.txt` | Requires the path to match a filesystem glob. |
| `type` | `type "regular" "directory"` | An existing file or directory. | A missing path. | Accepts any of the listed filesystem types. |
| `permissions` | `permissions 0o600` | A file with mode `0600`. | A file with mode `0644`. | Requires exactly these permission bits. |
| `owner-name` | `owner-name "alice"` | A file owned by Alice. | A file owned by another user. | Looks up a user by name. |
| `owner-id` | `owner-id 1000` | A file with UID 1000. | A file with UID 1001. | Requires the numeric owner UID. |
| `owner-me` | `owner-me #true` | A file owned by the current user. | A file owned by another user. | Checks ownership against the current user. |
| `group-name` | `group-name "staff"` | A file owned by group staff. | A file owned by another group. | Looks up a group by name. |
| `group-id` | `group-id 1000` | A file with GID 1000. | A file with GID 1001. | Requires the numeric owner GID. |
| `group-me` | `group-me #true` | A file whose group is one of the current user's groups. | A file owned by a different group. | Checks group membership. |
| `readable` | `readable #true` | A readable file. | A file denied read access. | Checks OS read access. |
| `writable` | `writable #true` | A writable file. | A file denied write access. | Checks OS write access. |
| `executable` | `executable #true` | An executable file or searchable directory. | A file denied execute access. | Checks OS execute/search access. |

#### `type` subvalidator

`type` selects the permitted filesystem object types. It takes quoted strings
from the list below. A path needs to match only one listed type; duplicate
types have no additional effect. Omitting `type` imposes no existence or type
restriction, while an explicitly empty `type` list accepts no path.

- `"exists"`: The path resolves to an existing filesystem object of any type.
  A symlink passes if its target exists; a dangling symlink fails. A filesystem
  lookup error, such as denied access to a parent directory, also fails.
- `"absent"`: Looking up the path reports that it does not exist. This can be
  used for a new output filename. A missing parent directory or a dangling
  symlink also passes; this does not establish that the path can be created.
  Other lookup errors, such as permission denied, fail.
- `"regular"`: The path resolves to a regular file, such as a text document,
  image, or executable file. Directories and special filesystem objects fail.
  A symlink to a regular file passes. Readability and executability are separate
  checks, so add their subvalidators if needed.
- `"directory"`: The path resolves to a directory. A symlink to a directory
  also passes. This does not check whether its contents can be listed or whether
  the directory can be searched; use `readable` and `executable` for those checks.
- `"symlink"`: The path itself is a symbolic link. This check inspects the link
  rather than following its target, so both working and dangling symlinks pass.
  An ordinary file or directory fails.
- `"pipe"`: The path resolves to a named pipe (FIFO), used for communication
  between processes. Regular files and unnamed pipes without a filesystem path
  do not satisfy this check.
- `"socket"`: The path resolves to a filesystem socket, such as a Unix-domain
  socket used by a local service. A network address or port number is not a
  filesystem socket path.
- `"device"`: The path resolves to a device node, either a block device or a
  character device. Examples include disk device nodes and `/dev/null`.
- `"char-device"`: The path resolves specifically to a character device, such
  as `/dev/null` or a terminal device. Block devices fail this check.

Multiple types are alternatives: `type "regular" "directory"` accepts either
a regular file or a directory. Other subvalidators are combined, and all must
pass. Except for `"symlink"`, type checks follow symbolic links; permissions
and ownership checks also inspect the target. Consequently, a dangling symlink
passes both `type "symlink"` and `type "absent"`, but fails `type "exists"`.

#### `glob` subvalidator

`glob` takes exactly one quoted filesystem pattern. Matching is case-sensitive;
`*` matches within a path component, `**` can span directories, and braces
allow alternatives. Relative patterns are interpreted from the current
directory. For example, `glob "/srv/data/**"` restricts paths to that directory
tree, while `glob "**/*.{json,yaml,yml}"` selects suffixes anywhere below the
current directory. Validation matches path names without resolving symlinks or
requiring existence; add `type "regular"` or another type constraint when
needed.

```kdl
value "filepath" {
  type "regular"
  glob "**/*.{json,yaml,yml}"
  readable #true
}
```

This accepts a readable regular file named `config.json`, `config.yaml`, or
`config.yml`. Renaming another kind of file to one of those extensions is
enough to pass the extension check; parsing the contents remains the script's
responsibility. Glob completion returns existing matches relative to the current
directory, filtered by all configured constraints.

#### `permissions` subvalidator

`permissions` takes exactly one integer from `0` through `0o777` and requires
an exact match for the path's nine owner/group/other permission bits. Octal
notation makes the usual Unix modes easy to express: `permissions 0o600`
requires owner read/write and no group or other permissions. A mode of `0o640`
fails that constraint even though the owner still has read/write access.
Setuid, setgid, and sticky bits are excluded from the comparison. Symlinks are
followed, and a missing path fails. This checks mode bits; use the access
subvalidators to check what the running process is allowed to do.

#### `owner-name` subvalidator

`owner-name` takes exactly one quoted username, such as `owner-name "alice"`.
The username is resolved to a UID when the validator is configured; an unknown
user causes a configuration error. Validation requires an existing path owned
by that UID. Symlinks are followed, so the target's owner is checked. This
does not imply read or write access and does not check group membership.

#### `owner-id` subvalidator

`owner-id` takes exactly one integer. For example, `owner-id 1000` requires the
path's owner UID to be 1000. No user lookup is performed, so the UID need not
have a named account on the machine. Negative values and values above the range
are configuration errors. The path must exist, and symlinks are followed to
inspect the target's owner.

#### `owner-me` subvalidator

`owner-me` takes exactly one boolean. `owner-me #true` requires the path to be
owned by the current user; `owner-me #false` requires a different owner. The
current user's UID is determined when the validator is configured. Both forms
require an existing path and inspect the target of a symlink. Ownership alone
does not establish whether the current user can read, write, or execute the
path.

#### `group-name` subvalidator

`group-name` takes exactly one quoted group name, such as `group-name "staff"`.
The group is resolved to a GID when the validator is configured; an unknown
group causes a configuration error. The path must exist and have that owning
GID. This compares the file's owning group, not the memberships of its owner or
the running user. Symlinks are followed to inspect the target's group.

#### `group-id` subvalidator

`group-id` takes exactly one integer. For example, `group-id 1000` requires the
path's owning GID to be 1000. No group-name lookup is performed, so the GID
need not have a named group on the machine. Out-of-range values are
configuration errors. Missing paths fail, and symlinks are followed to inspect
the target's group.

#### `group-me` subvalidator

`group-me` takes exactly one boolean. `group-me #true` requires the path's
owning GID to be among the current user's groups; `group-me #false` requires it
to be outside that set. The group IDs are obtained from the current user's
account when the validator is configured. This checks the owning group rather
than ownership by the current user, and does not by itself establish access.
The path must exist, and symlinks are followed.

#### `readable` subvalidator

`readable` takes one boolean and checks OS read access for the running process.
`readable #true` requires allowed read access; `readable #false` requires a
permission-denied result. For a directory, read access concerns listing its
entries; searching it is a separate execute check. Access is determined by OS
rules rather than just the owner mode bits. Missing paths and other lookup
errors fail both forms, and symlinks are followed. Passing does not guarantee
that a later read by the script succeeds.

#### `writable` subvalidator

The same as `readable` but for writes.

#### `executable` subvalidator

The same as `readable` but for executablity.

#### Completion

Completion lists entries matching the typed filename prefix and validates
file candidates against the configured constraints. Directories remain
available for navigation even under file-only constraints, are suffixed with
`/`, and suppress a trailing space. Candidate paths retain their directory
prefix. Completion reports an error if the directory cannot be read.

#### Example

```kdl
option "input" {
  description "Readable JSON input file"
  short "i"
  value "filepath" {
    type "regular"
    glob "**/*.json"
    readable #true
  }
}
```

This accepts `--input ./report.json` when it is a readable regular file,
and rejects a directory, a missing file, or `report.txt`. The script reads
the original path from `SHEBANG_OL_INPUT` or `SHEBANG_OS_I`.

### `base64` value

`base64` accepts Base64-encoded text. Validation decodes the input internally,
but options and positional arguments retain their original encoded strings.
Decoded data may contain arbitrary binary bytes.

The optional `format` property selects the alphabet and padding rules.

| Format | Example input | Description |
| ------ | ------------- | ----------- |
| `std` | `+/8=` | [Standard alphabet (`+`, `/`) with padding](https://www.rfc-editor.org/rfc/rfc4648.html#section-4); the default. |
| `url` | `-_8=` | [URL-safe alphabet (`-`, `_`)](https://www.rfc-editor.org/rfc/rfc4648.html#section-5) with padding. |
| `raw` | `+/8` | [Standard alphabet without padding](https://www.rfc-editor.org/rfc/rfc4648.html#section-3.2). |
| `raw-url` | `-_8` | URL-safe alphabet without padding. |

There are 2 subvalidators that validate the length of the decoded payload in
bytes.

| Subvalidator | Example definition | Passes | Fails | Description |
| ------------ | ------------------ | ------ | ----- | ----------- |
| `min-bytes` | `min-bytes 2` | `Zm8=` (2 bytes) | `Zg==` (1 byte) | Requires at least this many decoded bytes. |
| `max-bytes` | `max-bytes 2` | `Zm8=` (2 bytes) | `Zm9v` (3 bytes) | Allows at most this many decoded bytes. |

#### Example

```kdl
option "payload" {
  description "URL-safe Base64 payload containing 2 to 1024 decoded bytes"
  short "p"
  value "base64" format="raw-url" {
    min-bytes 2
    max-bytes 1024
  }
}
```

This accepts `--payload -_8` and rejects `--payload Zg` (only one decoded
byte) and `--payload -_8=` (padding is not allowed). The script reads the
original encoded string from `SHEBANG_OL_PAYLOAD` or `SHEBANG_OS_P`.

### `uuid` value

`uuid` accepts UUID strings as described in
[RFC 9562](https://www.rfc-editor.org/rfc/rfc9562.html).

| Form | Example input |
| ---- | ------------- |
| Hyphenated | `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` |
| Compact hexadecimal | `f81d4fae7dec11d0a76500a0c91e6bf6` |
| Braced | `{f81d4fae-7dec-11d0-a765-00a0c91e6bf6}` |
| URN | `urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6` |

Hexadecimal digits may be lowercase, uppercase, or mixed case. The URN prefix
must be exactly lowercase `urn:uuid:`. Braced input requires both braces, and
the hyphens must appear in the positions shown. Empty strings, partial UUIDs,
non-hexadecimal digits, and leading or trailing whitespace are rejected.

Validation checks the representation without restricting UUID version or
variant bits. The Nil UUID (`00000000-0000-0000-0000-000000000000`) and Max UUID
(`ffffffff-ffff-ffff-ffff-ffffffffffff`) are accepted.

This value supports no node properties or child subvalidators.

#### Example

```kdl
option "request-id" {
  description "UUID identifying the request"
  short "r"
  value "uuid"
}
```

This accepts `--request-id F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6` followed by
a valid resource UUID. The script reads the unchanged request identifier from
`SHEBANG_OL_REQUEST_ID` or `SHEBANG_OS_R`, and the resource identifier from its
first positional argument.

### `mac-addr` value

`mac-addr` validates MAC addresses. It supports 6-byte MAC-48/EUI-48 addresses,
8-byte EUI-64 addresses, and 20-byte IP-over-InfiniBand link-layer addresses.

Each supported length can use any of these notations:

| Notation | 6-byte example | Description |
| -------- | -------------- | ----------- |
| Colon-separated | `00:00:5e:00:53:01` | Two hexadecimal digits per byte, separated by `:`. |
| Hyphen-separated | `00-00-5e-00-53-01` | Two hexadecimal digits per byte, separated by `-`. |
| Dotted | `0000.5e00.5301` | Four hexadecimal digits per group, separated by `.`. |
| Compact hexadecimal | `00005e005301` | Two hexadecimal digits per byte, without separators. |

#### Example

```kdl
option "source" {
  description "Source hardware address"
  short "s"
  value "mac-addr"
}
arg "destination" {
  description "Destination hardware address"
  value "mac-addr"
}
```

This accepts `--source AA-BB-CC-DD-EE-FF 0000.5e00.5301`. The script reads
the unchanged source address from `SHEBANG_OL_SOURCE` or `SHEBANG_OS_S`, and
the destination address from its first positional argument.

### `email` value

`email` validates email addresses using that [famous RFC3522
regexp](https://stackoverflow.com/a/201378/5405453). This has **a lot** of
flaws but practically is doing a pretty good job. The only correct way to
validate email is trying sending there something but this is an overkill for
such tool. You are free doing that on your own, considering at least the
format.

| Subvalidator | Example definition | Description |
| ------------ | ------------------ | ----------- |
| `domain` | `domain "example.com"` | Requires an exact, case-sensitive domain match. |
| `domain-re` | `domain-re "^example[.]com$"` | Matches a Go regular expression against the domain. |
| `user-re` | `user-re "^alice(\\+.*)?$"` | Matches a Go regular expression against the local part before the final `@`. |
| `icann` | `icann #true` | Requires an ICANN-managed public suffix; `#false` excludes those suffixes. |

`domain`, `domain-re`, and `user-re` each take exactly one string;
`icann` takes exactly one boolean.

`domain` requires a literal match. `domain-re` validates by regular expression.
Same for `user-re` that works on "local part" of the email.

The `icann` check uses the embedded [public-suffix
list](https://en.wikipedia.org/wiki/Public_Suffix_List), without DNS lookups or
mail-server connections. For example, `alice@example.com` passes `icann #true`,
while `alice@example.invalid` passes `icann #false`.

#### Example

```kdl
option "recipient" {
  description "Recipient in the example.com domain"
  short "r"
  value "email" {
    domain "example.com"
    user-re "^alice(\\+.*)?$"
    icann #true
  }
}
```

This accepts `--recipient alice+tag@example.com`, but rejects
`bob@example.com` and `alice@EXAMPLE.COM`. The script reads the unchanged
address from `SHEBANG_OL_RECIPIENT` or `SHEBANG_OS_R`.

### `json` value

`json` validates one complete [JSON](https://www.rfc-editor.org/info/rfc8259/)
value without requiring a schema. The only reason why schema is not required is
because I haven't found a good way of representing it in KDL without feeling
myself like an idiot.

So, it just validates a correct syntax, but not the contents.

#### Example

```kdl
option "payload" {
  description "JSON value to process"
  short "p"
  value "json"
}
```

Quote JSON appropriately for your shell so it remains a single argument:

```shell
your-script --payload '{"message":"привет","items":[1,null]}'
your-script --payload 'null'
your-script --payload '1'
```

The script reads the unchanged JSON text from `SHEBANG_OL_PAYLOAD` or
`SHEBANG_OS_P`.

### `semver` value

`semver` parses semantic versions based on [Semantic Versions
2.0](https://semver.org/spec/v2.0.0.html) specifications.

It accepts full versions such as `1.2.3`, an optional lowercase `v` prefix, and
abbreviated versions such as `1` or `v1.2`. Missing minor and patch components
are treated as zero for comparisons. Leading zeros in major, minor, and patch
components are accepted. Prerelease and build metadata are supported, for
example `1.2.3-beta.1+build.123`.

The optional `constraint` child takes exactly one quoted constraint string:

| Constraint | Meaning |
| ---------- | ------- |
| `>= 1.2.3, < 2.0.0` | Both comparisons must pass; spaces can also separate comparisons. |
| `< 1.0.0 || >= 2.0.0` | Either comparison group may pass. |
| `= 1.2.3` | Exact version precedence; build metadata is ignored. |
| `!= 1.2.3` | Excludes that version. |
| `1.2.x` | Allows versions from `1.2.0` up to, but not including, `1.3.0`. |
| `~1.2.3` | Allows versions from `1.2.3` up to, but not including, `1.3.0`. |
| `^1.2.3` | Allows versions from `1.2.3` up to, but not including, `2.0.0`. |
| `^0.2.3` | Allows versions from `0.2.3` up to, but not including, `0.3.0`. |
| `^0.0.3` | Allows versions from `0.0.3` up to, but not including, `0.0.4`. |
| `1.2.3 - 2.0.0` | Inclusive range; spaces around the hyphen are significant. |

This is a common notation from
[node.js](https://flaviocopes.com/npm-semantic-versioning/) or
[composer](https://getcomposer.org/doc/articles/versions.md#writing-version-constraints).

#### Example

```kdl
option "release" {
  description "Release in the supported major version"
  short "r"
  value "semver" {
    constraint ">= 1.2.0 < 2.0.0"
  }
}
```

This accepts `--release v1.2` and `--release 1.9.0+build.1`, but rejects
`2.0.0` and prereleases such as `1.3.0-beta.1`. The script reads the unchanged
version from `SHEBANG_OL_RELEASE` or `SHEBANG_OS_R`.

### `url` value

`url` accepts URL references as they are defined by
[RFC 3986](https://www.rfc-editor.org/rfc/rfc3986.html).

| Subvalidator | Example definition | Description |
| ------------ | ------------------ | ----------- |
| `scheme` | `scheme "https"` | Exact match against the parsed, lowercased scheme. |
| `user` | `user "alice"` | Exact match against the decoded username. |
| `user-re` | `user-re "^alice$"` | Regex match against the decoded username. |
| `password` | `password #false` | Requires (`#true`) or excludes (`#false`) password userinfo. |
| `domain` | `domain "example.com"` | Exact, case-sensitive hostname match, excluding port and IPv6 brackets. |
| `domain-re` | `domain-re "(^|[.])example[.]com$"` | Regex match against the hostname. |
| `path-re` | `path-re "^/api/"` | Regex match against the decoded path. |
| `path-glob` | `path-glob "/api/**/*.json"` | Whole-path glob match, using `/` separators and recursive `**`. |
| `query` | `query "mode" "safe" "format" "json"` | Requires each key; every occurrence must equal its configured value. |
| `query-re` | `query-re "id" "^[0-9]+$"` | Requires each key; every occurrence must match its configured regex. |
| `fragment-re` | `fragment-re "^section-"` | Regex match against the decoded fragment. |

Except for `query` and `query-re`, each child takes exactly one argument:
a boolean for `password`, a quoted string for every other child. Query
constraints take one or more key/value pairs of quoted strings. Odd argument
counts and duplicate keys within a query constraint are rejected. Properties
on the `value` node are not supported. Invalid regexes and globs fail during
configuration. All configured constraints must pass.

Regexes are case-sensitive and unanchored unless the expression specifies
otherwise. Usernames, paths, fragments, and query keys/values are percent-decoded
before matching. For example, `/a%2Fb` matches the path `/a/b`, and query
`q=a+b` matches the value `a b`. Extra query parameters are allowed. When a
query constraint is configured, malformed query escapes or unescaped semicolons
fail validation, even in extra parameters.

Missing userinfo behaves as an empty username and absent password. An explicitly
empty password (`https://alice:@example.com`) counts as present. Domain names
are not lowercased; use a case-insensitive regex such as `(?i)^example[.]com$`
if needed. A scheme constraint alone does not require a host:
`https:opaque` also has scheme `https`. Opaque URLs generally have an empty path.

#### Example

```kdl
option "endpoint" {
  description "HTTPS API endpoint in the expected domain"
  short "e"
  value "url" {
    scheme "https"
    domain "example.com"
    password #false
    path-glob "/api/**"
    query "mode" "safe"
    query-re "id" "^[0-9]+$"
  }
}
```

This accepts `--endpoint 'https://example.com/api/items?mode=safe&id=01&id=02'`.
The script reads the unchanged URL from `SHEBANG_OL_ENDPOINT` or `SHEBANG_OS_E`.

### `git` value

`git` accepts Git reference names and hexadecimal commit hashes. Without a
`type` child it checks naming syntax only: names need not exist, and no
repository is required. Short names such as `main`, `origin/main`, and `v1.2`
and full references such as `refs/heads/main` are supported. `HEAD` is also
accepted. Revision expressions such as `HEAD~1` and `main^` are not supported.

shebang uses a library to access git objects, so no command line _git_ is
required to be installed. The repository is discovered from the current working
directory or its parents.

To require local existence, specify one or more categories:

| Type | Accepted input |
| ---- | -------------- |
| `head` | `HEAD`, when it resolves to an existing commit, attached or detached. |
| `commit` | Full commit hashes or unambiguous hexadecimal object prefixes of at least four characters. |
| `local-branch` | `main` or `refs/heads/main`. |
| `remote-branch` | `origin/main` or `refs/remotes/origin/main`. |
| `tag` | Lightweight tag names such as `v1.2` or `refs/tags/v1.2`. |
| `annotated-tag` | Annotated tag names, short or under `refs/tags/`. |
| `any` | Any of the categories above. |

`type` takes one or more quoted strings; categories are alternatives. Explicit
namespaces are respected: a `refs/heads/release` input cannot match a tag named
`release`. Bare remote branch names are not searched across remotes; include
the remote name. `head` is separate from `commit`.

Commit prefixes may have [odd or even
lengths](https://git-scm.com/book/en/v2/Git-Tools-Revision-Selection#Short-SHA-1)
and either hexadecimal case. They must identify one distinct object across all
object types, and that object must be a commit. An annotated-tag object's hash
is not a commit hash. Branch and tag categories check reference presence rather
than whether their targets are checkout-compatible commits. Remote references
include locally stored symbolic aliases such as `origin/HEAD`. No network
requests or fetches are performed.

Completion filters the selected categories by the text typed, sorts and
deduplicates suggestions, and preserves full namespaces when supplied. Commit
suggestions require at least four hexadecimal characters. Already valid input
completes to its original spelling, including valid abbreviated hashes.

```kdl
option "revision" {
  description "Existing commit, branch, tag, or HEAD in the current repository"
  short "r"
  value "git" {
    type "any"
  }
}
```

The script reads the original revision from `SHEBANG_OL_REVISION` or
`SHEBANG_OS_R`.
