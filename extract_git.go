package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ParseGitDiff parses a unified diff and returns a per-file summary with stats.
func ParseGitDiff(text string) (string, error) {
	lines := strings.Split(text, "\n")
	type fileChange struct {
		path string
		adds int
		dels int
	}
	var files []fileChange
	var current *fileChange

	diffFileRe := regexp.MustCompile(`^diff --git a/(.+) b/(.+)`)
	addRe := regexp.MustCompile(`^\+[^+]`)
	delRe := regexp.MustCompile(`^-[^-]`)

	for _, line := range lines {
		if m := diffFileRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				files = append(files, *current)
			}
			current = &fileChange{path: m[2]}
			continue
		}
		if current == nil {
			continue
		}
		if addRe.MatchString(line) {
			current.adds++
		} else if delRe.MatchString(line) {
			current.dels++
		}
	}
	if current != nil {
		files = append(files, *current)
	}

	if len(files) == 0 {
		return text, nil
	}

	totalAdds, totalDels := 0, 0
	for _, f := range files {
		totalAdds += f.adds
		totalDels += f.dels
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[+%d -%d in %d files]\n\n", totalAdds, totalDels, len(files)))
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("%s: +%d -%d\n", f.path, f.adds, f.dels))
	}
	return sb.String(), nil
}

// ParseGitLog parses git log output and clusters by conventional commit prefix.
func ParseGitLog(text string) (string, error) {
	commitRe := regexp.MustCompile(`^commit [0-9a-f]+`)
	prefixRe := regexp.MustCompile(`^(feat|fix|chore|refactor|docs|test|perf|ci|build|style)`)

	groups := map[string][]string{}
	var other []string
	var currentSubject string

	for _, line := range strings.Split(text, "\n") {
		if commitRe.MatchString(line) || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") {
			continue
		}
		subject := strings.TrimSpace(line)
		if subject == "" || currentSubject == subject {
			continue
		}
		currentSubject = subject
		if m := prefixRe.FindString(subject); m != "" {
			groups[m] = append(groups[m], subject)
		} else {
			other = append(other, subject)
		}
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		subjects := groups[k]
		sb.WriteString(fmt.Sprintf("%s (%d):\n", k, len(subjects)))
		for _, s := range subjects {
			sb.WriteString("  - " + s + "\n")
		}
		sb.WriteRune('\n')
	}
	if len(other) > 0 {
		sb.WriteString(fmt.Sprintf("other (%d):\n", len(other)))
		for _, s := range other {
			sb.WriteString("  - " + s + "\n")
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// ParseGitBlame parses git blame output.
func ParseGitBlame(text string) (string, error) {
	authors := map[string]int{}
	lineRe := regexp.MustCompile(`^\^?[0-9a-f]+ \(([^)]+)\s+\d{4}-\d{2}-\d{2}`)
	for _, line := range strings.Split(text, "\n") {
		if m := lineRe.FindStringSubmatch(line); m != nil {
			author := strings.TrimSpace(m[1])
			nameParts := strings.Fields(author)
			if len(nameParts) > 0 {
				authors[nameParts[0]]++
			}
		}
	}
	if len(authors) == 0 {
		return text, nil
	}
	type pair struct {
		name  string
		count int
	}
	var pairs []pair
	for n, c := range authors {
		pairs = append(pairs, pair{n, c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })

	var sb strings.Builder
	sb.WriteString("Blame by author:\n")
	for _, p := range pairs {
		sb.WriteString(fmt.Sprintf("  %s: %d lines\n", p.name, p.count))
	}
	return sb.String(), nil
}

// FetchGitHubPR fetches a GitHub PR URL and returns its diff+description.
func FetchGitHubPR(ctx context.Context, prURL string) (string, error) {
	u, err := url.Parse(prURL)
	if err != nil {
		return "", fmt.Errorf("parse PR URL: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return "", fmt.Errorf("not a GitHub PR URL: %s", prURL)
	}
	return extractGitHubPR(ctx, parts[0], parts[1], parts[3])
}

// resolveGitInput validates args and returns the raw git text.
func resolveGitInput(ctx context.Context, args CondenseGitArgs) (string, error) {
	count := 0
	if args.Text != "" {
		count++
	}
	if args.Path != "" {
		count++
	}
	if args.PRUrl != "" {
		count++
	}
	if count != 1 {
		return "", fmt.Errorf("condense_git requires exactly one of: text, path, pr_url (got %d)", count)
	}
	if args.Text != "" {
		return args.Text, nil
	}
	if args.Path != "" {
		b, err := os.ReadFile(args.Path)
		if err != nil {
			return "", fmt.Errorf("read git file: %w", err)
		}
		return string(b), nil
	}
	return FetchGitHubPR(ctx, args.PRUrl)
}

// detectAndParse auto-detects git content type and parses accordingly.
func detectAndParse(text string) (string, error) {
	if strings.Contains(text, "diff --git") {
		return ParseGitDiff(text)
	}
	if regexp.MustCompile(`^commit [0-9a-f]{7,}`).MatchString(text) {
		return ParseGitLog(text)
	}
	if regexp.MustCompile(`^\^?[0-9a-f]+ \(`).MatchString(text) {
		return ParseGitBlame(text)
	}
	return text, nil
}
