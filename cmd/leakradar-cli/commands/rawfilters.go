package commands

import (
	"fmt"

	"github.com/spf13/pflag"

	"leakradar-cli/internal/api"
)

// rawOpts holds the raw dataset's filter surface — unlike the other
// datasets, there's only one search shape here (no email/domain/advanced
// split), so `raw search`, `raw count`, `raw unlock` and `raw export` all
// share this one flag set via registerRawFilterFlags.
var rawOpts struct {
	q           string
	containerID int

	exts, extsNot             []string
	categories, categoriesNot []string

	fileName, fileNameNot               []string
	fileNameMatch, fileNameNotMatch     string
	folderName, folderNameNot           []string
	folderNameMatch, folderNameNotMatch string

	forceAnd bool
	qExact   bool

	includeTerms, excludeTerms []string
	ingestedFrom, ingestedTo   string
}

var validRawMatchTypes = map[string]bool{"contains": true, "starts_with": true, "ends_with": true, "exact": true}

func rawMatchTypePtr(name, val string) (*api.RawMatchType, error) {
	if val == "" {
		return nil, nil
	}
	if !validRawMatchTypes[val] {
		return nil, fmt.Errorf("invalid --%s %q (must be contains, starts_with, ends_with, or exact)", name, val)
	}
	t := api.RawMatchType(val)
	return &t, nil
}

// registerRawFilterFlags attaches the full raw filter flag set to cmd.
func registerRawFilterFlags(f *pflag.FlagSet) {
	f.StringVar(&rawOpts.q, "q", "", "full-text query to match within raw block content (usually what you want; see 'leakradar-cli raw --help' for when it's optional)")
	f.IntVar(&rawOpts.containerID, "container-id", 0, "restrict the search to a specific container ID")

	f.StringArrayVar(&rawOpts.exts, "ext", nil, "include file extension, no leading dot (repeatable)")
	f.StringArrayVar(&rawOpts.extsNot, "ext-not", nil, "exclude file extension (repeatable)")
	f.StringArrayVar(&rawOpts.categories, "category", nil, "include category: stealer-logs|database|combolist (repeatable)")
	f.StringArrayVar(&rawOpts.categoriesNot, "category-not", nil, "exclude category (repeatable)")

	f.StringArrayVar(&rawOpts.fileName, "file-name", nil, "match entry file name, wildcard (repeatable)")
	f.StringArrayVar(&rawOpts.fileNameNot, "file-name-not", nil, "exclude entry file name (repeatable)")
	f.StringVar(&rawOpts.fileNameMatch, "file-name-match", "", "contains|starts_with|ends_with|exact (default contains)")
	f.StringVar(&rawOpts.fileNameNotMatch, "file-name-not-match", "", "contains|starts_with|ends_with|exact (default contains)")

	f.StringArrayVar(&rawOpts.folderName, "folder-name", nil, "match the folder containing the block (repeatable)")
	f.StringArrayVar(&rawOpts.folderNameNot, "folder-name-not", nil, "exclude a containing folder (repeatable)")
	f.StringVar(&rawOpts.folderNameMatch, "folder-name-match", "", "contains|starts_with|ends_with|exact (default contains)")
	f.StringVar(&rawOpts.folderNameNotMatch, "folder-name-not-match", "", "contains|starts_with|ends_with|exact (default contains)")

	f.BoolVar(&rawOpts.forceAnd, "force-and", false, "require all file-name/folder-name values to match (AND, default OR)")
	f.BoolVar(&rawOpts.qExact, "q-exact", false, "require --q to match on word boundaries (e.g. won't match '4.4.4.4' inside '4.4.4.400')")

	f.StringArrayVar(&rawOpts.includeTerms, "include-term", nil, "extra content term that must also be present, min 4 chars (repeatable)")
	f.StringArrayVar(&rawOpts.excludeTerms, "exclude-term", nil, "content term to exclude, min 4 chars (repeatable)")
	f.StringVar(&rawOpts.ingestedFrom, "ingested-from", "", "only blocks ingested on/after this RFC3339 datetime")
	f.StringVar(&rawOpts.ingestedTo, "ingested-to", "", "only blocks ingested on/before this RFC3339 datetime")
}

// buildRawSearchRequest builds the RawSearchRequest body from rawOpts.
// Field-presence validation (the API requires --q or a positive filter) is
// left to the server, whose error message is authoritative here — the
// exact rule has more exceptions (e.g. --include-term alone is valid)
// than are worth re-deriving client-side.
func buildRawSearchRequest() (api.RawSearchRequest, error) {
	var req api.RawSearchRequest

	req.Q = rawOpts.q
	if rawOpts.containerID != 0 {
		id := rawOpts.containerID
		req.ContainerID = &id
	}
	req.Exts, req.ExtsNot = rawOpts.exts, rawOpts.extsNot
	req.Categories, req.CategoriesNot = rawOpts.categories, rawOpts.categoriesNot
	req.FileName, req.FileNameNot = rawOpts.fileName, rawOpts.fileNameNot
	req.FolderName, req.FolderNameNot = rawOpts.folderName, rawOpts.folderNameNot
	req.IncludeTerms, req.ExcludeTerms = rawOpts.includeTerms, rawOpts.excludeTerms
	req.IngestedAtMin, req.IngestedAtMax = rawOpts.ingestedFrom, rawOpts.ingestedTo

	if rawOpts.forceAnd {
		v := true
		req.ForceAnd = &v
	}
	if rawOpts.qExact {
		v := true
		req.QExact = &v
	}

	var err error
	if req.FileNameMatchType, err = rawMatchTypePtr("file-name-match", rawOpts.fileNameMatch); err != nil {
		return req, err
	}
	if req.FileNameNotMatchType, err = rawMatchTypePtr("file-name-not-match", rawOpts.fileNameNotMatch); err != nil {
		return req, err
	}
	if req.FolderNameMatchType, err = rawMatchTypePtr("folder-name-match", rawOpts.folderNameMatch); err != nil {
		return req, err
	}
	if req.FolderNameNotMatchType, err = rawMatchTypePtr("folder-name-not-match", rawOpts.folderNameNotMatch); err != nil {
		return req, err
	}

	return req, nil
}
