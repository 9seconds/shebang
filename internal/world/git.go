package world

import (
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/9seconds/shebang/internal/log"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/hash"
)

// GitCollectMinLength is the minimum hexadecimal hash-prefix length in
// characters.
const GitCollectMinLength = 4

type gitCollectedReferences struct {
	branches       []string
	tags           []string
	annotatedTags  []string
	remoteBranches []string
}

// Git caches local repository queries and pools independently opened
// repositories. The working directory must remain unchanged during the
// instance's lifetime.
type Git struct {
	repoPool      sync.Pool
	references    funcOnce[gitCollectedReferences]
	hasHead       funcOnce[bool]
	commits       memoized[string, bool]
	hasCommit     memoized[string, bool]
	annotatedTags memoized[string, bool]
}

// HasHead reports whether HEAD resolves to an existing commit.
func (g *Git) HasHead() bool {
	return g.hasHead.Do(func() bool {
		repo, err := g.acquireRepo()
		if err != nil {
			log.Printf("Git: cannot acquire repo: %s", err)

			return false
		}

		defer g.releaseRepo(repo)

		ref, err := repo.Head()
		if err != nil {
			return false
		}

		return g.containsCommit(ref.Hash().String())
	})
}

// HasCommit checks full hashes or unambiguous object prefixes for a commit.
// Lookup errors yield false.
func (g *Git) HasCommit(input string) bool {
	if err := g.validHash(input); err != nil {
		return false
	}

	input = strings.ToLower(input)

	return g.hasCommit.Do(input, func() bool {
		if len(input) == hash.HexSize {
			return g.containsCommit(input)
		}

		hashes, err := g.collectMatchingHashes(input)
		if err == nil && len(hashes) == 1 {
			return g.containsCommit(hashes[0])
		}

		return false
	})
}

// HasLocalBranch looks up a short local branch name; errors yield false.
func (g *Git) HasLocalBranch(input string) bool {
	branches, err := g.GetLocalBranches()
	if err != nil {
		return false
	}

	_, found := slices.BinarySearch(branches, input)

	return found
}

// HasRemoteBranch looks up a remote/branch name; errors yield false.
func (g *Git) HasRemoteBranch(input string) bool {
	branches, err := g.GetRemoteBranches()
	if err != nil {
		return false
	}

	_, found := slices.BinarySearch(branches, input)

	return found
}

// HasTag looks up a short lightweight tag name; errors yield false.
func (g *Git) HasTag(input string) bool {
	tags, err := g.GetTags()
	if err != nil {
		return false
	}

	_, found := slices.BinarySearch(tags, input)

	return found
}

// HasAnnotatedTag looks up a short annotated tag name; errors yield false.
func (g *Git) HasAnnotatedTag(input string) bool {
	tags, err := g.GetAnnotatedTags()
	if err != nil {
		return false
	}

	_, found := slices.BinarySearch(tags, input)

	return found
}

// FilterCommits returns sorted distinct commit hashes matching a hexadecimal
// prefix.
func (g *Git) FilterCommits(prefix string) ([]string, error) {
	if err := g.validHash(prefix); err != nil {
		return nil, err
	}

	prefix = strings.ToLower(prefix)

	// fast path. if we are given a full commit, we can skip
	// an expensive scanning
	if len(prefix) == hash.HexSize {
		if g.containsCommit(prefix) {
			return []string{prefix}, nil
		}

		return nil, nil
	}

	hashes, err := g.collectMatchingHashes(prefix)
	if err != nil {
		return nil, err
	}

	workers := &sync.WaitGroup{}
	results := make(chan string)
	params := make(chan string)

	workers.Go(func() {
		for _, hsh := range hashes {
			params <- hsh
		}

		close(params)
	})

	for range min(len(hashes), runtime.GOMAXPROCS(0)) {
		workers.Go(func() {
			for param := range params {
				if g.containsCommit(param) {
					results <- param
				}
			}
		})
	}

	go func() {
		workers.Wait()
		close(results)
	}()

	collected := map[string]struct{}{}

	for val := range results {
		collected[val] = struct{}{}
	}

	return slices.Sorted(maps.Keys(collected)), nil
}

// FilterLocalBranches filters short local branch names by prefix.
func (g *Git) FilterLocalBranches(input string) ([]string, error) {
	items, err := g.GetLocalBranches()
	if err != nil {
		return nil, fmt.Errorf("cannot get local branches: %w", err)
	}

	filtered := make([]string, 0, len(items))

	for _, val := range items {
		if strings.HasPrefix(val, input) {
			filtered = append(filtered, val)
		}
	}

	return filtered, nil
}

// FilterRemoteBranches filters local remote/branch names by prefix.
func (g *Git) FilterRemoteBranches(input string) ([]string, error) {
	items, err := g.GetRemoteBranches()
	if err != nil {
		return nil, fmt.Errorf("cannot get remote branches: %w", err)
	}

	filtered := make([]string, 0, len(items))

	for _, val := range items {
		if strings.HasPrefix(val, input) {
			filtered = append(filtered, val)
		}
	}

	return filtered, nil
}

// FilterTags filters short lightweight tag names by prefix.
func (g *Git) FilterTags(input string) ([]string, error) {
	items, err := g.GetTags()
	if err != nil {
		return nil, fmt.Errorf("cannot get tags: %w", err)
	}

	filtered := make([]string, 0, len(items))

	for _, val := range items {
		if strings.HasPrefix(val, input) {
			filtered = append(filtered, val)
		}
	}

	return filtered, nil
}

// FilterAnnotatedTags filters short annotated tag names by prefix.
func (g *Git) FilterAnnotatedTags(input string) ([]string, error) {
	items, err := g.GetAnnotatedTags()
	if err != nil {
		return nil, fmt.Errorf("cannot get annotated tags: %w", err)
	}

	filtered := make([]string, 0, len(items))

	for _, val := range items {
		if strings.HasPrefix(val, input) {
			filtered = append(filtered, val)
		}
	}

	return filtered, nil
}

// GetLocalBranches returns sorted short branch names; the slice is read-only.
func (g *Git) GetLocalBranches() ([]string, error) {
	refs, err := g.collectReferences()

	return refs.branches, err
}

// GetTags returns sorted lightweight tag names; the slice is read-only.
func (g *Git) GetTags() ([]string, error) {
	refs, err := g.collectReferences()

	return refs.tags, err
}

// GetAnnotatedTags returns sorted annotated tag names; the slice is read-only.
func (g *Git) GetAnnotatedTags() ([]string, error) {
	refs, err := g.collectReferences()

	return refs.annotatedTags, err
}

// GetRemoteBranches returns sorted local remote-tracking names, including aliases.
// Names have the form remote/branch and the returned slice is read-only.
func (g *Git) GetRemoteBranches() ([]string, error) {
	refs, err := g.collectReferences()

	return refs.remoteBranches, err
}

func (g *Git) collectReferences() (gitCollectedReferences, error) {
	return g.references.DoE(func() (gitCollectedReferences, error) {
		log.Printf("Git: collecting local repository reference snapshot")

		collected := gitCollectedReferences{}

		repo, err := g.acquireRepo()
		if err != nil {
			return collected, fmt.Errorf("cannot open repo: %w", err)
		}

		defer g.releaseRepo(repo)

		refIter, err := repo.References()
		if err != nil {
			return collected, fmt.Errorf("cannot scan references: %w", err)
		}

		err = refIter.ForEach(func(ref *plumbing.Reference) error {
			name := ref.Name()

			switch {
			case name.IsBranch():
				collected.branches = append(collected.branches, name.Short())
			case name.IsTag():
				sn := name.Short()
				if g.containsAnnotatedTag(ref.Hash().String()) {
					collected.annotatedTags = append(collected.annotatedTags, sn)
				} else {
					collected.tags = append(collected.tags, sn)
				}
			case name.IsRemote():
				collected.remoteBranches = append(collected.remoteBranches, name.Short())
			}

			return nil
		})
		if err != nil {
			err = fmt.Errorf("cannot collect references: %w", err)
		}

		slices.Sort(collected.branches)
		slices.Sort(collected.remoteBranches)
		slices.Sort(collected.tags)
		slices.Sort(collected.annotatedTags)

		log.Printf(
			"Git: collected %d local branches, %d remote branches, %d tags, %d annotated tags",
			len(collected.branches),
			len(collected.remoteBranches),
			len(collected.tags),
			len(collected.annotatedTags),
		)

		return collected, err
	})
}

func (g *Git) containsCommit(strHash string) bool {
	return g.commits.Do(strHash, func() bool {
		if !plumbing.IsHash(strHash) {
			return false
		}

		repo, err := g.acquireRepo()
		if err != nil {
			log.Printf("Git: cannot acquire repo: %s", err)

			return false
		}

		defer g.releaseRepo(repo)

		_, err = repo.CommitObject(plumbing.NewHash(strHash))
		if err != nil {
			log.Printf("Git: hash %s is not a commit: %s", strHash, err)
		}

		return err == nil
	})
}

func (g *Git) containsAnnotatedTag(strHash string) bool {
	return g.annotatedTags.Do(strHash, func() bool {
		if !plumbing.IsHash(strHash) {
			return false
		}

		repo, err := g.acquireRepo()
		if err != nil {
			log.Printf("Git: cannot acquire repo: %s", err)

			return false
		}

		defer g.releaseRepo(repo)

		_, err = repo.TagObject(plumbing.NewHash(strHash))
		if err != nil {
			log.Printf("Git: hash %s is not an annotated tag: %s", strHash, err)
		}

		return err == nil
	})
}

func (g *Git) validHash(prefix string) error {
	if len(prefix) < GitCollectMinLength || len(prefix) > hash.HexSize {
		return fmt.Errorf(
			"%w: expected %d to %d hexadecimal characters",
			ErrGitHashLength,
			GitCollectMinLength,
			hash.HexSize,
		)
	}

	for _, char := range prefix {
		// for example, abcdez
		if !unicode.Is(unicode.ASCII_Hex_Digit, char) {
			return fmt.Errorf("%w: %q", ErrGitHashHex, prefix)
		}
	}

	return nil
}

func (g *Git) collectMatchingHashes(prefix string) ([]string, error) {
	if err := g.validHash(prefix); err != nil {
		return nil, err
	}

	prefix = strings.ToLower(prefix)

	// this looks a bit ridiculous but this is what happens:
	//  - git usually uses 7 characters for abbreviated commit hashes
	//    https://git-scm.com/book/en/v2/Git-Tools-Revision-Selection#Short-SHA-1
	//  - 7 % 2 == 1, so it means it already broken encoding
	//  - so, we do [:L/2*2]. 7/2*2 = (7/2) * 2 = 3 * 2 = 6
	//    this is just an alignment. But before, we want to check
	//  - filter out excess hashes
	pfx, err := hex.DecodeString(prefix[:len(prefix)/2*2])
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	repo, err := g.acquireRepo()
	if err != nil {
		return nil, fmt.Errorf("cannot open a repository: %w", err)
	}

	defer g.releaseRepo(repo)

	finder, ok := repo.Storer.(interface {
		HashesWithPrefix(prefix []byte) ([]plumbing.Hash, error)
	})
	if !ok {
		return nil, fmt.Errorf("%w: %T", ErrGitPrefixLookup, repo.Storer)
	}

	hashes, err := finder.HashesWithPrefix(pfx)
	if err != nil {
		return nil, fmt.Errorf("cannot scan hashes: %w", err)
	}

	uniques := map[string]struct{}{}

	for _, candidate := range hashes {
		strHash := candidate.String()
		if strings.HasPrefix(strHash, prefix) {
			uniques[strHash] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(uniques)), nil
}

var (
	// ErrGitHashLength indicates a hash prefix outside the supported length range.
	ErrGitHashLength = errors.New("invalid Git hash prefix length")
	// ErrGitHashHex indicates a non-ASCII-hexadecimal hash prefix.
	ErrGitHashHex = errors.New("invalid Git hexadecimal hash prefix")
	// ErrGitPrefixLookup indicates storage without hash-prefix lookup support.
	ErrGitPrefixLookup = errors.New("Git storage does not support hash-prefix lookup")
)

func (g *Git) acquireRepo() (*git.Repository, error) {
	if cached := g.repoPool.Get(); cached != nil {
		return cached.(*git.Repository), nil //nolint:forcetypeassert // Only releaseRepo fills this pool.
	}

	return g.openRepo()
}

func (g *Git) releaseRepo(repo *git.Repository) {
	g.repoPool.Put(repo)
}

func (g *Git) openRepo() (*git.Repository, error) {
	log.Printf("Git: discovering repository from current working directory")

	repo, err := git.PlainOpenWithOptions(
		".",
		&git.PlainOpenOptions{
			DetectDotGit:          true,
			EnableDotGitCommonDir: true,
		},
	)
	if err != nil {
		log.Printf("Git: repository discovery failed: %s", err)
	}

	return repo, err
}
