package values_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/shebang/internal/utils"
	"github.com/9seconds/shebang/internal/values"
	"github.com/9seconds/shebang/internal/world"
	gitlib "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type GitTestSuite struct {
	suite.Suite

	hash string
}

func (s *GitTestSuite) SetupTest() {
	root := s.T().TempDir()
	repo, err := gitlib.PlainInit(root, false)
	s.Require().NoError(err)

	worktree, err := repo.Worktree()
	s.Require().NoError(err)

	s.Require().NoError(os.WriteFile(filepath.Join(root, "file"), []byte("привет"), 0o600))

	_, err = worktree.Add("file")
	s.Require().NoError(err)

	hash, err := worktree.Commit("fixture", &gitlib.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Unix(1700000000, 0),
		},
	})
	s.Require().NoError(err)

	s.hash = hash.String()
	for _, name := range []string{
		"refs/heads/main",
		"refs/heads/feature/привет",
		"refs/remotes/origin/main",
		"refs/tags/lightweight",
	} {
		s.Require().NoError(repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.ReferenceName(name), hash,
		)))
	}

	_, err = repo.CreateTag("annotated", hash, &gitlib.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Unix(1700000000, 0),
		},
		Message: "release",
	})
	s.Require().NoError(err)

	s.T().Chdir(root)

	world.World.Git = world.Git{}

	s.T().Cleanup(func() { world.World.Git = world.Git{} })
}

func (s *GitTestSuite) TestTypes() {
	for _, test := range []struct {
		kind string
		good []string
		bad  []string
	}{
		{
			kind: "head",
			good: []string{"HEAD"},
			bad:  []string{"main"},
		},
		{
			kind: "commit",
			good: []string{
				s.hash,
				s.hash[:7],
				strings.ToUpper(s.hash[:8]),
			},
			bad: []string{
				"HEAD",
				"main",
				s.hash[:3],
			},
		},
		{
			kind: "local-branch",
			good: []string{
				"main",
				"refs/heads/main",
				"feature/привет",
			},
			bad: []string{
				"refs/tags/main",
				"missing",
				"HEAD",
			},
		},
		{
			kind: "remote-branch",
			good: []string{
				"origin/main",
				"refs/remotes/origin/main",
			},
			bad: []string{"main"},
		},
		{
			kind: "tag",
			good: []string{
				"lightweight",
				"refs/tags/lightweight",
			},
			bad: []string{
				"annotated",
				"refs/heads/lightweight",
			},
		},
		{
			kind: "annotated-tag",
			good: []string{
				"annotated",
				"refs/tags/annotated",
			},
			bad: []string{"lightweight"},
		},
	} {
		s.Run(test.kind, func() {
			for _, valid := range []bool{true, false} {
				inputs := test.bad
				if valid {
					inputs = test.good
				}

				for _, input := range inputs {
					s.Run(input, func() {
						// Arrange
						value, err := values.New("git", nil, map[string][]any{
							"type": {test.kind},
						})
						s.Require().NoError(err)

						// Act
						err = value.Validate(input)

						// Assert
						if valid {
							s.Require().NoError(err)
						} else {
							s.Require().ErrorIs(err, values.ErrGitReferenceNotFound)
						}
					})
				}
			}
		})
	}
}

func (s *GitTestSuite) TestSyntaxOnly() {
	for _, input := range []string{
		"HEAD",
		"nonexistent",
		"v1.2",
		"origin/main",
		"abcd123",
		"refs/heads/main",
		"",
		"bad name",
		"main~1",
		"a..b",
	} {
		s.Run(input, func() {
			// Arrange
			value, err := values.NewGit(nil, nil)
			s.Require().NoError(err)

			// Act
			err = value.Validate(input)

			// Assert
			if input == "" || strings.ContainsAny(input, " ~") || input == "a..b" {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *GitTestSuite) TestCompletion() {
	for _, test := range []struct {
		kind  string
		input string
		want  []cobra.Completion
	}{
		{
			kind:  "head",
			input: "HE",
			want:  []cobra.Completion{"HEAD"},
		},
		{
			kind:  "local-branch",
			input: "refs/heads/mai",
			want:  []cobra.Completion{"refs/heads/main"},
		},
		{
			kind:  "remote-branch",
			input: "refs/remotes/origin/ma",
			want:  []cobra.Completion{"refs/remotes/origin/main"},
		},
		{
			kind:  "tag",
			input: "refs/tags/li",
			want:  []cobra.Completion{"refs/tags/lightweight"},
		},
		{
			kind:  "annotated-tag",
			input: "ann",
			want:  []cobra.Completion{"annotated"},
		},
		{
			kind:  "any",
			input: "refs/tags/li",
			want:  []cobra.Completion{"refs/tags/lightweight"},
		},
		{
			kind:  "any",
			input: "HEAD",
			want:  []cobra.Completion{"HEAD"},
		},
		{
			kind:  "local-branch",
			input: "refs/tags/ma",
		},
		{
			kind:  "commit",
			input: strings.ToUpper(s.hash[:7]),
			want:  []cobra.Completion{strings.ToUpper(s.hash[:7])},
		},
	} {
		s.Run(test.kind+"/"+test.input, func() {
			// Arrange
			value, err := values.NewGit(nil, map[string][]any{
				"type": {test.kind},
			})
			s.Require().NoError(err)

			// Act
			candidates, directive := value.Complete(test.input)

			// Assert
			s.ElementsMatch(test.want, candidates)
			s.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func (s *GitTestSuite) TestConfiguration() {
	for _, test := range []struct {
		args []any
		want error
	}{
		{
			want: values.ErrGitEmptyTypes,
		},
		{
			args: []any{"typo"},
			want: values.ErrGitUnknownType,
		},
		{
			args: []any{
				"any",
				"typo",
			},
			want: values.ErrGitUnknownType,
		},
		{
			args: []any{true},
		},
	} {
		s.Run("invalid configuration", func() {
			// Arrange
			subvalidators := map[string][]any{
				"type": test.args,
			}

			// Act
			value, err := values.NewGit(nil, subvalidators)

			// Assert
			s.Nil(value)

			if test.want != nil {
				s.Require().ErrorIs(err, test.want)
			} else {
				var typeError *utils.ArgumentTypeError

				s.Require().ErrorAs(err, &typeError)
			}
		})
	}

	// Arrange
	subvalidators := map[string][]any{
		"type": {
			"tag",
			"local-branch",
			"tag",
		},
	}

	// Act
	value, err := values.NewGit(nil, subvalidators)

	// Assert
	s.Require().NoError(err)
	s.Equal("git(type:local-branch,tag)", value.String())
}

func (s *GitTestSuite) TestOutsideRepository() {
	// Arrange
	s.T().Chdir(s.T().TempDir())

	world.World.Git = world.Git{}
	value, err := values.NewGit(nil, map[string][]any{
		"type": {"any"},
	})
	s.Require().NoError(err)

	// Act
	err = value.Validate("HEAD")
	candidates, directive := value.Complete("HE")

	// Assert
	s.Require().ErrorIs(err, values.ErrGitReferenceNotFound)

	s.Empty(candidates)
	s.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
}

func (s *GitTestSuite) TestUnknownSubvalidator() {
	// Arrange
	subvalidators := map[string][]any{
		"unknown": {},
	}

	// Act
	value, err := values.NewGit(nil, subvalidators)

	// Assert
	s.Nil(value)
	s.Require().ErrorIs(err, values.ErrUnknownSubvalidator)
}

//nolint:paralleltest // Git fixtures change the process working directory and global cache.
func TestGit(t *testing.T) {
	suite.Run(t, &GitTestSuite{})
}
