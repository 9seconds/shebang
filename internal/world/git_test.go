package world

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gitlib "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func newGitFixture(t *testing.T) (*gitlib.Repository, plumbing.Hash) {
	t.Helper()

	root := t.TempDir()
	repo, err := gitlib.PlainInit(root, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "file"), []byte("привет"), 0o600))

	_, err = worktree.Add("file")
	require.NoError(t, err)

	hash, err := worktree.Commit("fixture", &gitlib.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Unix(1700000000, 0),
		},
	})
	require.NoError(t, err)

	for _, name := range []string{
		"refs/heads/main",
		"refs/heads/feature/привет",
		"refs/remotes/origin/main",
		"refs/remotes/upstream/feature/topic",
		"refs/tags/lightweight",
	} {
		require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.ReferenceName(name), hash,
		)))
	}

	_, err = repo.CreateTag("embedded-name", hash, &gitlib.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Unix(1700000000, 0),
		},
		Message: "annotated tag",
	})
	require.NoError(t, err)

	ref, err := repo.Tag("embedded-name")
	require.NoError(t, err)

	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.NewTagReferenceName("renamed"), ref.Hash(),
	)))
	require.NoError(t, repo.Storer.RemoveReference(ref.Name()))

	t.Chdir(root)

	return repo, hash
}

type GitTestSuite struct {
	suite.Suite
}

func (s *GitTestSuite) TestReferences() {
	t := s.T()

	// Arrange
	_, _ = newGitFixture(t)

	var git Git

	// Act
	branches, branchesError := git.GetLocalBranches()
	remotes, remotesError := git.GetRemoteBranches()
	tags, tagsError := git.GetTags()
	annotated, annotatedError := git.GetAnnotatedTags()

	// Assert
	require.NoError(t, branchesError)
	require.Contains(t, branches, "main")
	require.Contains(t, branches, "feature/привет")

	require.NoError(t, remotesError)
	require.Equal(t, []string{
		"origin/main",
		"upstream/feature/topic",
	}, remotes)

	require.NoError(t, tagsError)
	require.Equal(t, []string{"lightweight"}, tags)

	require.NoError(t, annotatedError)
	require.Equal(t, []string{"renamed"}, annotated)

	require.True(t, git.HasLocalBranch("main"))
	require.False(t, git.HasLocalBranch("refs/heads/main"))
	require.True(t, git.HasRemoteBranch("origin/main"))
	require.False(t, git.HasRemoteBranch("missing/main"))
	require.True(t, git.HasTag("lightweight"))
	require.False(t, git.HasTag("renamed"))
	require.True(t, git.HasAnnotatedTag("renamed"))
	require.False(t, git.HasAnnotatedTag("lightweight"))
}

func (s *GitTestSuite) TestCommitCache() {
	t := s.T()

	// Arrange
	repo, hash := newGitFixture(t)

	var git Git

	commit, err := repo.CommitObject(hash)
	require.NoError(t, err)

	missing := "1111111111111111111111111111111111111111"

	// Act
	found := git.HasCommit(hash.String())
	notFound := git.HasCommit(missing)
	tree := git.HasCommit(commit.TreeHash.String())
	short := git.HasCommit(hash.String()[:8])
	invalid := git.HasCommit("привет")

	runtime.GC()

	cached := git.HasCommit(hash.String())

	// Assert
	require.True(t, found)
	require.False(t, notFound)
	require.False(t, tree)
	require.True(t, short)
	require.False(t, invalid)
	require.True(t, cached)
}

func (s *GitTestSuite) TestReferenceSnapshot() {
	t := s.T()

	// Arrange
	repo, hash := newGitFixture(t)

	var git Git

	_, err := git.GetLocalBranches()
	require.NoError(t, err)

	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(
		plumbing.NewBranchReferenceName("later"), hash,
	)))

	// Act
	cached := git.HasLocalBranch("later")

	var fresh Git

	updated := fresh.HasLocalBranch("later")

	// Assert
	require.False(t, cached)
	require.True(t, updated)
}

func (s *GitTestSuite) TestMissingRepository() {
	t := s.T()

	// Arrange
	t.Chdir(t.TempDir())

	var git Git

	// Act
	_, firstError := git.GetLocalBranches()
	_, cachedError := git.GetTags()
	commit := git.HasCommit("1111111111111111111111111111111111111111")
	branch := git.HasLocalBranch("main")
	remote := git.HasRemoteBranch("origin/main")
	tag := git.HasTag("tag")
	annotated := git.HasAnnotatedTag("tag")

	// Assert
	require.ErrorIs(t, firstError, gitlib.ErrRepositoryNotExists)
	require.Same(t, firstError, cachedError)
	require.False(t, commit)
	require.False(t, branch)
	require.False(t, remote)
	require.False(t, tag)
	require.False(t, annotated)
}

func (s *GitTestSuite) TestConcurrent() {
	t := s.T()

	// Arrange
	_, hash := newGitFixture(t)

	var (
		git     Git
		workers sync.WaitGroup
	)

	results := make([]bool, 32)
	errs := make([]error, len(results))

	// Act
	for idx := range results {
		workers.Go(func() {
			repo, err := git.acquireRepo()
			if err != nil {
				errs[idx] = err

				return
			}
			defer git.releaseRepo(repo)

			_, errs[idx] = repo.CommitObject(hash)
			results[idx] = git.HasCommit(hash.String()) && git.HasLocalBranch("main")
		})
	}

	workers.Wait()

	// Assert
	for idx, result := range results {
		require.NoError(t, errs[idx])
		require.True(t, result)
	}
}

func (s *GitTestSuite) TestPoolReuseAndOpenFailure() {
	t := s.T()

	// Arrange
	_, _ = newGitFixture(t)

	var git Git

	// Act
	first, firstError := git.acquireRepo()
	second, secondError := git.acquireRepo()

	// Assert
	require.NoError(t, firstError)
	require.NoError(t, secondError)
	require.NotSame(t, first, second)

	// Arrange
	git.releaseRepo(first)
	t.Chdir(t.TempDir())

	// Act
	pooled, pooledError := git.acquireRepo()
	missing, missingError := git.acquireRepo()

	// Assert
	// Pool entries may be discarded at any time, so reuse itself is optional.
	if pooledError == nil {
		require.Same(t, first, pooled)
	} else {
		require.ErrorIs(t, pooledError, gitlib.ErrRepositoryNotExists)
	}

	require.Nil(t, missing)
	require.ErrorIs(t, missingError, gitlib.ErrRepositoryNotExists)
}

func (s *GitTestSuite) TestFilters() {
	// Arrange
	_, _ = newGitFixture(s.T())

	var git Git

	for _, test := range []struct {
		name    string
		collect func(string) ([]string, error)
		prefix  string
		want    []string
	}{
		{
			name:    "local branches",
			collect: git.FilterLocalBranches,
			prefix:  "feature/",
			want:    []string{"feature/привет"},
		},
		{
			name:    "remote branches",
			collect: git.FilterRemoteBranches,
			prefix:  "origin/",
			want:    []string{"origin/main"},
		},
		{
			name:    "lightweight tags",
			collect: git.FilterTags,
			prefix:  "light",
			want:    []string{"lightweight"},
		},
		{
			name:    "annotated tags",
			collect: git.FilterAnnotatedTags,
			prefix:  "re",
			want:    []string{"renamed"},
		},
		{
			name:    "empty prefix",
			collect: git.FilterRemoteBranches,
			want: []string{
				"origin/main",
				"upstream/feature/topic",
			},
		},
		{
			name:    "no match",
			collect: git.FilterLocalBranches,
			prefix:  "missing",
		},
	} {
		s.Run(test.name, func() {
			// Arrange
			prefix := test.prefix

			// Act
			got, err := test.collect(prefix)

			// Assert
			s.Require().NoError(err)
			s.ElementsMatch(test.want, got)
		})
	}
}

func (s *GitTestSuite) TestCollectCommits() {
	// Arrange
	repo, hash := newGitFixture(s.T())
	commit, err := repo.CommitObject(hash)
	s.Require().NoError(err)

	minimum := GitCollectMinLength

	for _, test := range []struct {
		name    string
		prefix  string
		want    []string
		invalid bool
	}{
		{
			name:   "minimum prefix",
			prefix: hash.String()[:minimum],
			want:   []string{hash.String()},
		},
		{
			name:   "full hash",
			prefix: hash.String(),
			want:   []string{hash.String()},
		},
		{
			name:   "uppercase prefix",
			prefix: strings.ToUpper(hash.String()[:minimum]),
			want:   []string{hash.String()},
		},
		{
			name:    "below minimum",
			prefix:  hash.String()[:minimum-2],
			invalid: true,
		},
		{
			name:    "empty prefix",
			invalid: true,
		},
		{
			name:   "odd number of hex digits",
			prefix: hash.String()[:minimum+1],
			want:   []string{hash.String()},
		},
		{
			name:    "invalid hexadecimal",
			prefix:  "привет",
			invalid: true,
		},
		{
			name:   "no matching object",
			prefix: strings.Repeat("0", 40),
		},
		{
			name:   "tree is not a commit",
			prefix: commit.TreeHash.String(),
		},
		{
			name:    "prefix longer than hash",
			prefix:  hash.String() + "00",
			invalid: true,
		},
	} {
		s.Run(test.name, func() {
			// Arrange
			var git Git

			// Act
			got, err := git.FilterCommits(test.prefix)

			// Assert
			if test.invalid {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			s.ElementsMatch(test.want, got)
		})
	}
}

func (s *GitTestSuite) TestCollectorsConcurrent() {
	// Arrange
	_, hash := newGitFixture(s.T())

	var (
		git     Git
		workers sync.WaitGroup
	)

	results := make(chan string)
	collectors := []func(string) ([]string, error){
		git.FilterLocalBranches,
		git.FilterRemoteBranches,
		git.FilterTags,
		git.FilterAnnotatedTags,
	}

	var got []string

	// Act
	for _, collect := range collectors {
		workers.Go(func() {
			items, _ := collect("")
			for _, item := range items {
				results <- item
			}
		})
	}

	workers.Go(func() {
		items, _ := git.FilterCommits(hash.String())
		for _, item := range items {
			results <- item
		}
	})
	go func() {
		workers.Wait()
		close(results)
	}()

	for result := range results {
		got = append(got, result)
	}

	// Assert
	s.Contains(got, "main")
	s.Contains(got, "feature/привет")
	s.Contains(got, "origin/main")
	s.Contains(got, "upstream/feature/topic")
	s.Contains(got, "lightweight")
	s.Contains(got, "renamed")
	s.Contains(got, hash.String())
}

func (s *GitTestSuite) TestCollectorsOutsideRepository() {
	// Arrange
	s.T().Chdir(s.T().TempDir())

	var git Git

	for _, collect := range []func(string) ([]string, error){
		git.FilterLocalBranches,
		git.FilterRemoteBranches,
		git.FilterTags,
		git.FilterAnnotatedTags,
	} {
		// Arrange
		prefix := ""

		// Act
		results, err := collect(prefix)

		// Assert
		s.Empty(results)
		s.Require().Error(err)
	}
}

func (s *GitTestSuite) TestHeadStates() {
	// Arrange
	repo, hash := newGitFixture(s.T())
	commit, err := repo.CommitObject(hash)
	s.Require().NoError(err)

	for _, test := range []struct {
		name string
		ref  *plumbing.Reference
		want bool
	}{
		{
			name: "attached",
			ref:  plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
			want: true,
		},
		{
			name: "detached",
			ref:  plumbing.NewHashReference(plumbing.HEAD, hash),
			want: true,
		},
		{
			name: "unborn",
			ref:  plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("absent")),
		},
		{
			name: "dangling",
			ref:  plumbing.NewHashReference(plumbing.HEAD, plumbing.NewHash(strings.Repeat("1", 40))),
		},
		{
			name: "tree",
			ref:  plumbing.NewHashReference(plumbing.HEAD, commit.TreeHash),
		},
	} {
		s.Run(test.name, func() {
			// Arrange
			s.Require().NoError(repo.Storer.SetReference(test.ref))

			var git Git

			// Act
			found := git.HasHead()

			// Assert
			s.Equal(test.want, found)
		})
	}
}

func (s *GitTestSuite) TestAmbiguousObjectPrefix() {
	// Arrange
	repo, hash := newGitFixture(s.T())
	prefix := hash.String()[:4]

	var blobHash plumbing.Hash

	// Find a real blob sharing the commit's first four hexadecimal digits.
	for index := range 2000000 {
		data := []byte(strconv.Itoa(index))

		candidate := plumbing.ComputeHash(plumbing.BlobObject, data)
		if !strings.HasPrefix(candidate.String(), prefix) {
			continue
		}

		encoded := repo.Storer.NewEncodedObject()
		encoded.SetType(plumbing.BlobObject)
		encoded.SetSize(int64(len(data)))
		writer, err := encoded.Writer()
		s.Require().NoError(err)

		_, err = writer.Write(data)
		s.Require().NoError(err)

		s.Require().NoError(writer.Close())

		blobHash, err = repo.Storer.SetEncodedObject(encoded)
		s.Require().NoError(err)

		break
	}

	s.Require().NotEqual(plumbing.ZeroHash, blobHash)

	var git Git

	// Act
	ambiguous := git.HasCommit(prefix)
	full := git.HasCommit(hash.String())
	blob := git.HasCommit(blobHash.String())
	commits, err := git.FilterCommits(prefix)

	// Assert
	s.False(ambiguous)

	s.True(full)

	s.False(blob)

	s.Require().NoError(err)
	s.Equal([]string{hash.String()}, commits)
}

//nolint:paralleltest // Repository discovery uses the process working directory.
func TestGit(t *testing.T) {
	suite.Run(t, &GitTestSuite{})
}
