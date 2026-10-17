package values

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/world"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

// GitAnyType selects every supported Git reference category.
const GitAnyType = "any"

const gitHead = "HEAD"

func gitRefLookup(namespace string, lookup func(string) bool) func(string) bool {
	return func(input string) bool {
		if strings.HasPrefix(input, "refs/") {
			if !strings.HasPrefix(input, namespace) {
				return false
			}

			input = strings.TrimPrefix(input, namespace)
		}

		return lookup(input)
	}
}

var (
	gitLookups = map[string]func(string) bool{
		"head": func(input string) bool {
			return input == gitHead && world.World.Git.HasHead()
		},
		"commit": world.World.Git.HasCommit,
		"local-branch": gitRefLookup(
			"refs/heads/",
			world.World.Git.HasLocalBranch,
		),
		"remote-branch": gitRefLookup(
			"refs/remotes/",
			world.World.Git.HasRemoteBranch,
		),
		"tag": gitRefLookup(
			"refs/tags/",
			world.World.Git.HasTag,
		),
		"annotated-tag": gitRefLookup(
			"refs/tags/",
			world.World.Git.HasAnnotatedTag,
		),
	}
	gitCollectors = map[string]func(string) ([]string, error){
		"head": func(input string) ([]string, error) {
			if strings.HasPrefix(gitHead, input) && world.World.Git.HasHead() {
				return []string{gitHead}, nil
			}

			return nil, nil
		},
		"commit":        world.World.Git.FilterCommits,
		"local-branch":  gitRefFilter("refs/heads/", world.World.Git.FilterLocalBranches),
		"remote-branch": gitRefFilter("refs/remotes/", world.World.Git.FilterRemoteBranches),
		"tag":           gitRefFilter("refs/tags/", world.World.Git.FilterTags),
		"annotated-tag": gitRefFilter("refs/tags/", world.World.Git.FilterAnnotatedTags),
	}
)

type valueGit struct {
	baseValue[string]

	types []string
}

// NewGit constructs a Git reference-name validator with optional existence categories.
//
//nolint:ireturn // Git validators expose the shared Value interface.
func NewGit(properties map[string]any, subvalidators map[string][]any) (Value, error) {
	val := &valueGit{
		baseValue: newBase[string]("git", len(subvalidators)),
	}
	val.toValue = func(input string) (string, error) {
		if input == gitHead {
			return input, nil
		}

		name := plumbing.ReferenceName(input)
		if !strings.HasPrefix(input, "refs/") {
			name = plumbing.NewBranchReferenceName(input)
		}

		if err := name.Validate(); err != nil {
			return "", err
		}

		return input, nil
	}
	val.toComplete = val.genCompletions

	if len(properties) > 0 {
		return nil, ErrNoProperties
	}

	for k, v := range subvalidators {
		if err := val.addSubvalidator(k, v); err != nil {
			return nil, fmt.Errorf("cannot add subvalidator %s: %w", k, err)
		}
	}

	return val, nil
}

func (v *valueGit) addSubvalidator(name string, properties []any) error {
	if name == "type" { //nolint:goconst // Keep configuration names explicit.
		return v.addSubvalidatorType(properties)
	}

	return ErrUnknownSubvalidator
}

//nolint:cyclop // Validate and normalize categories before installing their combined lookup.
func (v *valueGit) addSubvalidatorType(properties []any) error {
	types, err := utils.All[string](properties)
	if err != nil {
		return err
	}

	for _, typ := range types {
		if _, ok := gitLookups[typ]; !ok && typ != GitAnyType {
			return fmt.Errorf("%w: %s", ErrGitUnknownType, typ)
		}
	}

	if slices.Contains(types, GitAnyType) {
		types = slices.Collect(maps.Keys(gitLookups))
	}

	slices.Sort(types)
	types = slices.Compact(types)

	if len(types) == 0 {
		return ErrGitEmptyTypes
	}

	v.types = types

	v.setSubvalidator("type", strings.Join(types, ","), func(input string) error {
		workers := sync.WaitGroup{}
		results := make(chan bool)

		for _, typ := range v.types {
			workers.Go(func() {
				results <- gitLookups[typ](input)
			})
		}

		go func() {
			workers.Wait()
			close(results)
		}()

		found := false
		for val := range results {
			found = found || val
		}

		if found {
			return nil
		}

		return ErrGitReferenceNotFound
	})

	return nil
}

func gitRefFilter(
	namespace string,
	filter func(string) ([]string, error),
) func(string) ([]string, error) {
	return func(input string) ([]string, error) {
		full := strings.HasPrefix(input, "refs/")
		if full {
			if !strings.HasPrefix(input, namespace) {
				return nil, nil
			}

			input = strings.TrimPrefix(input, namespace)
		}

		matches, err := filter(input)
		if err != nil || !full {
			return matches, err
		}

		results := make([]string, len(matches))
		for idx, match := range matches {
			results[idx] = namespace + match
		}

		return results, nil
	}
}

func (v *valueGit) genCompletions(input string) ([]string, cobra.ShellCompDirective) {
	if len(v.types) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	workers := &sync.WaitGroup{}
	results := make(chan string, 1)

	for _, typ := range v.types {
		workers.Go(func() {
			if values, err := gitCollectors[typ](input); err == nil {
				for _, val := range values {
					results <- val
				}
			}
		})
	}

	go func() {
		workers.Wait()
		close(results)
	}()

	suggestions := []string(nil)

	for val := range results {
		suggestions = append(suggestions, val)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
