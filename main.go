package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"sort"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	identChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type identifier struct {
	chars     string
	color     *color.Color
	identStr  string
	signature object.Signature
}

func (i identifier) String() string {
	return i.color.Sprint(i.chars)
}

// hash :: identifier - this makes sure we grab the same identifier for the same hash every time
var (
	seenHashes = map[string]identifier{} //nolint:gochecknoglobals
	// identMap    = map[string]identifier{} //nolint:gochecknoglobals
	identifiers = []identifier{}
	identColors = []*color.Color{ //nolint:gochecknoglobals
		color.New(color.FgHiBlue),
		color.New(color.FgHiCyan),
		color.New(color.FgHiGreen),
		color.New(color.FgMagenta),
		color.New(color.FgRed),
		color.New(color.FgHiYellow),
		color.New(color.FgWhite, color.BgBlue),
		color.New(color.FgWhite, color.BgRed),
		color.New(color.FgBlack, color.BgHiGreen),
		color.New(color.FgWhite, color.BgGreen),
		color.New(color.FgBlack, color.BgHiYellow),
	}
	graphCommitter bool
)

func nextIdent(person object.Signature) identifier {
	char1 := string(identChars[len(identifiers)/len(identChars)])
	char2 := string(identChars[len(identifiers)%len(identChars)])
	identColor := identColors[len(identifiers)%len(identColors)]
	identStr := char1 + char2

	ident := identifier{
		chars:     char1 + char2,
		color:     identColor,
		identStr:  identStr,
		signature: person,
	}

	identifiers = append(identifiers, ident)

	return ident
}

func getCommitterIdent(person object.Signature) (identifier, error) {
	sha := sha256.New()
	if _, err := sha.Write([]byte(person.Name + person.Email)); err != nil {
		return identifier{}, fmt.Errorf("failed to get author name+email hash (name=%q, email=%q): %w", person.Name, person.Email, err)
	}

	hash := sha.Sum(nil)
	hashStr := hex.EncodeToString(hash)

	if ident, ok := seenHashes[hashStr]; ok {
		return ident, nil
	}

	ident := nextIdent(person)
	ident.signature = person

	seenHashes[hashStr] = ident

	return ident, nil
}

func getCommitterMap(commits []*object.Commit) error {
	for _, commit := range commits {
		person := commit.Author
		if graphCommitter {
			person = commit.Committer
		}

		ident, err := getCommitterIdent(person)
		if err != nil {
			return err
		}

		fmt.Printf("%s ", ident)
	}

	return nil
}

func walkRepo(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	log, err := repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return fmt.Errorf("failed to get git log: %w", err)
	}

	defer log.Close()

	commits := make([]*object.Commit, 0, 1000)

	for i := 0; ; i++ {
		commit, err := log.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			fmt.Printf("error with commit %d: %v\n", i, err)

			continue
		}

		commits = append(commits, commit)
	}

	// Get commits from first to last
	slices.Reverse(commits)

	if err := getCommitterMap(commits); err != nil {
		return fmt.Errorf("failed to map unique committer :: identifier: %w", err)
	}

	if graphCommitter {
		fmt.Println("\n\nGit committers:")
	} else {
		fmt.Println("\n\nGit authors:")
	}

	sort.Slice(identifiers, func(i, j int) bool {
		return identifiers[i].identStr < identifiers[j].identStr
	})

	for _, ident := range identifiers {
		fmt.Printf("%s: %s (%s)\n", ident.String(), ident.signature.Name, ident.signature.Email)
	}

	return nil
}

func setupFlags() {
	flag.BoolVar(&graphCommitter, "committer", false, "graph committer instead of author")

	flag.Parse()
}

func main() {
	setupFlags()

	if len(flag.Args()) < 1 {
		panic(fmt.Errorf("must supply a path to a repo"))
	}

	path := flag.Arg(0)

	if err := walkRepo(path); err != nil {
		panic(fmt.Errorf("failed to walk %q: %w", path, err))
	}
}
