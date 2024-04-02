package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	identChars      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	identColorChars = "bcgmry"
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
	seenHashes  = map[string]identifier{} //nolint:gochecknoglobals
	identMap    = map[string]identifier{} //nolint:gochecknoglobals
	identColors = map[byte]*color.Color{  //nolint:gochecknoglobals
		'b': color.New(color.FgHiBlue),
		'c': color.New(color.FgHiCyan),
		'g': color.New(color.FgHiGreen),
		'm': color.New(color.FgMagenta),
		'r': color.New(color.FgRed),
		'y': color.New(color.FgHiYellow),
	}
	graphCommitter bool
)

func nextIdent(person object.Signature) identifier {
	char1 := string(identChars[len(identMap)/len(identChars)])
	char2 := string(identChars[len(identMap)%len(identChars)])
	colorChar := identColorChars[len(identMap)%len(identColorChars)]
	identStr := char1 + char2 + string(colorChar)

	id := identifier{
		chars:     char1 + char2,
		color:     identColors[colorChar],
		identStr:  identStr,
		signature: person,
	}

	identMap[identStr] = id

	return id
}

func getCommitterIdent(person object.Signature) (identifier, error) {
	// can re-use characters between name and email

	sha := sha256.New()
	_, err := sha.Write([]byte(person.Name + person.Email))
	if err != nil {
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

	for _, ident := range identMap {
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
