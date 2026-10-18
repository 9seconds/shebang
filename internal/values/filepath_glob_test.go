package values_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/9seconds/shebang/internal/values"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type FilepathGlobTestSuite struct {
	suite.Suite
}

func (s *FilepathGlobTestSuite) TestRelativeCompletions() {
	// Arrange
	parent := s.T().TempDir()
	root := filepath.Join(parent, "work")
	outside := filepath.Join(parent, "outside.txt")

	s.Require().NoError(os.Mkdir(root, 0o700))
	s.Require().NoError(os.Mkdir(filepath.Join(root, "nested"), 0o700))
	s.Require().NoError(os.WriteFile(outside, nil, 0o600))

	for _, name := range []string{
		"a.txt",
		"b.json",
		"nested/привет.txt",
	} {
		s.Require().NoError(os.WriteFile(filepath.Join(root, name), nil, 0o600))
	}

	s.T().Chdir(root)

	for _, pattern := range []string{
		"**/*.txt",
		filepath.Join(root, "**", "*.txt"),
	} {
		s.Run(pattern, func() {
			for _, test := range []struct {
				prefix string
				want   []string
			}{
				{
					prefix: "",
					want: []string{
						"a.txt",
						"nested/привет.txt",
					},
				},
				{
					prefix: "nested/",
					want:   []string{"nested/привет.txt"},
				},
				{
					prefix: "./",
					want: []string{
						"./a.txt",
						"./nested/привет.txt",
					},
				},
				{
					prefix: "./nested/",
					want:   []string{"./nested/привет.txt"},
				},
				{
					prefix: "missing",
					want:   []string{},
				},
			} {
				s.Run(test.prefix, func() {
					// Arrange
					value, err := values.NewFilepath(nil, map[string][]any{
						"glob": {pattern},
						"type": {"regular"},
					})
					s.Require().NoError(err)

					// Act
					paths, directive := value.Complete(test.prefix)

					// Assert
					s.Equal(cobra.ShellCompDirectiveNoFileComp, directive)
					s.Equal(test.want, paths)

					for _, path := range paths {
						s.False(filepath.IsAbs(path))
						s.Require().NoError(value.Validate(path))
					}
				})
			}
		})
	}
}

func (s *FilepathGlobTestSuite) TestGlobEdgeCasesAndDotSlash() {
	// Arrange
	parent := s.T().TempDir()
	root := filepath.Join(parent, "work")
	outside := filepath.Join(parent, "outside.txt")

	s.Require().NoError(os.Mkdir(root, 0o700))
	s.Require().NoError(os.Mkdir(filepath.Join(root, "nested"), 0o700))
	s.Require().NoError(os.WriteFile(outside, nil, 0o600))
	s.Require().NoError(os.WriteFile(filepath.Join(root, "a.txt"), nil, 0o600))

	s.T().Chdir(root)

	for _, test := range []struct {
		name      string
		glob      string
		input     string
		want      []string
		directive cobra.ShellCompDirective
	}{
		{
			name:      "broad glob skips current directory dot",
			glob:      "**",
			input:     "./",
			want:      []string{"./a.txt"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:      "glob outside working directory skipped",
			glob:      filepath.Join(parent, "*.txt"),
			input:     "",
			want:      []string{},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:  "cwd completion preserves dot-slash for directories and files",
			input: "./",
			want: []string{
				"./a.txt",
				"./nested/",
			},
			directive: cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:      "cwd completion preserves dot-slash file prefix",
			input:     "./a",
			want:      []string{"./a.txt"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	} {
		s.Run(test.name, func() {
			// Arrange
			subvalidators := map[string][]any{
				"type": {"regular"},
			}
			if test.glob != "" {
				subvalidators["glob"] = []any{test.glob}
			}

			value, err := values.NewFilepath(nil, subvalidators)
			s.Require().NoError(err)

			// Act
			paths, directive := value.Complete(test.input)

			// Assert
			s.Equal(test.want, paths)
			s.Equal(test.directive, directive)
		})
	}
}

//nolint:paralleltest // Relative completion tests change the process working directory.
func TestFilepathGlob(t *testing.T) {
	suite.Run(t, &FilepathGlobTestSuite{})
}
