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
	root := s.T().TempDir()
	s.Require().NoError(os.Mkdir(filepath.Join(root, "nested"), 0o700))

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
			for _, prefix := range []string{
				"",
				"nested/",
				"missing",
			} {
				s.Run(prefix, func() {
					// Arrange
					value, err := values.NewFilepath(nil, map[string][]any{
						"glob": {pattern},
						"type": {"regular"},
					})
					s.Require().NoError(err)

					// Act
					paths, directive := value.Complete(prefix)

					// Assert
					s.Equal(cobra.ShellCompDirectiveNoFileComp, directive)

					switch prefix {
					case "":
						s.Equal([]string{
							"a.txt",
							"nested/привет.txt",
						}, paths)
					case "nested/":
						s.Equal([]string{"nested/привет.txt"}, paths)
					default:
						s.Empty(paths)
					}

					for _, path := range paths {
						s.False(filepath.IsAbs(path))
						s.Require().NoError(value.Validate(path))
					}
				})
			}
		})
	}
}

//nolint:paralleltest // Relative completion tests change the process working directory.
func TestFilepathGlob(t *testing.T) {
	suite.Run(t, &FilepathGlobTestSuite{})
}
