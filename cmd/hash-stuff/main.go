package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	hash_stuff "github.com/davidwallacejackson/hash-stuff"
	"github.com/jessevdk/go-flags"
)

var dir = ""
var include = []string{}
var exclude = []string{}

type hashStuffOpts struct {
	IncludePatterns []string `short:"i" long:"include" description:"glob to include (can be passed multiple times)"`
	ExcludePatterns []string `short:"e" long:"exclude" description:"globs to exclude (can be passed multiple times)"`
	ShowSummary     bool     `long:"show-summary" description:"print a list of files with their individual hashes to stderr"`
	WriteSummaryTo  string   `long:"write-summary-to" description:"write a list of files with their individual hashes to the given path"`
	Positional      struct {
		RootPaths []string `positional-arg-name:"path" required:"1"`
	} `positional-args:"yes"`
}

func (opts hashStuffOpts) Usage() string {
	return "foo bar"
}

func main() {
	opts := hashStuffOpts{}

	parser := flags.NewParser(&opts, flags.Default)
	parser.Name = "hash-stuff"
	parser.LongDescription = `Pass one or more file or directory paths as arguments to hash them. The digest will be written to stdout.

The summary presented by the --show-summary and --write-summary-to options is what gets hashed to form the digest.`

	_, err := parser.Parse()
	if err != nil {
		// the parser will output info about invalid args, so we don't have
		// to do it here
		os.Exit(1)
	}

	if len(opts.IncludePatterns) == 0 {
		log.Println("No --include passed, assuming \"**\"")
		opts.IncludePatterns = []string{"**"}
	}

	digest, summary, err := hash_stuff.GetDigest(
		opts.Positional.RootPaths,
		opts.IncludePatterns,
		opts.ExcludePatterns,
	)
	if err != nil {
		panic(err)
	}

	if opts.ShowSummary {
		fmt.Fprintln(os.Stderr, "Summary:")
		fmt.Fprintln(os.Stderr, summary)
	}

	if opts.WriteSummaryTo != "" {
		if err := ioutil.WriteFile(opts.WriteSummaryTo, []byte(summary), 0777); err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "Wrote summary to %s\n", opts.WriteSummaryTo)
	}

	fmt.Printf("%x", digest)
}
