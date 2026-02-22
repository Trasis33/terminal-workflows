package picker

import (
	"strings"

	"github.com/fredriklanga/wf/internal/store"
	"github.com/sahilm/fuzzy"
)

// ParseQuery splits a raw query string into a tag filter and fuzzy query.
// If the query starts with "@", the first word (without @) is the tag filter,
// and the rest is the fuzzy query. Otherwise the entire string is the fuzzy query.
func ParseQuery(raw string) (tagFilter, fuzzyQuery string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	if strings.HasPrefix(raw, "@") {
		rest := raw[1:]
		parts := strings.SplitN(rest, " ", 2)
		tagFilter = parts[0]
		if len(parts) > 1 {
			fuzzyQuery = strings.TrimSpace(parts[1])
		}
		return tagFilter, fuzzyQuery
	}

	return "", raw
}

// WorkflowSource adapts a slice of store.Workflow to the sahilm/fuzzy Source
// interface. String(i) concatenates all searchable fields: name, description,
// tags, and command content (SRCH-01).
type WorkflowSource []store.Workflow

// String returns the searchable text for workflow at index i.
func (ws WorkflowSource) String(i int) string {
	w := ws[i]
	return w.Name + " " + w.Description + " " + strings.Join(w.Tags, " ") + " " + w.Command
}

// Len returns the number of workflows in the source.
func (ws WorkflowSource) Len() int {
	return len(ws)
}

// Search performs fuzzy search over workflows with optional tag pre-filtering.
// If tagFilter is non-empty, workflows are narrowed to those with a matching tag
// (case-insensitive) before fuzzy matching. If query is empty, all (filtered)
// workflows are returned as synthetic matches preserving original order.
func Search(query, tagFilter string, workflows []store.Workflow) []fuzzy.Match {
	filtered := workflows
	if tagFilter != "" {
		filtered = filterByTag(workflows, tagFilter)
	}

	if len(filtered) == 0 {
		return nil
	}

	// Empty query returns all filtered workflows as synthetic matches.
	// sahilm/fuzzy returns 0 results on empty query, so we bypass it.
	if query == "" {
		matches := make([]fuzzy.Match, len(filtered))
		for i := range filtered {
			matches[i] = fuzzy.Match{
				Str:   WorkflowSource(filtered).String(i),
				Index: indexOf(workflows, &filtered[i]),
			}
		}
		return matches
	}

	// Build a source from filtered workflows and run fuzzy matching.
	source := WorkflowSource(filtered)
	results := fuzzy.FindFrom(query, source)

	// Remap indices back to the original workflows slice.
	for i := range results {
		results[i].Index = indexOf(workflows, &filtered[results[i].Index])
	}

	return results
}

// filterByTag returns workflows that have a tag matching the given filter
// (case-insensitive comparison).
func filterByTag(workflows []store.Workflow, tag string) []store.Workflow {
	tag = strings.ToLower(tag)
	var result []store.Workflow
	for i := range workflows {
		for _, t := range workflows[i].Tags {
			if strings.ToLower(t) == tag {
				result = append(result, workflows[i])
				break
			}
		}
	}
	return result
}

// indexOf finds the index of workflow wf in the original slice by pointer comparison.
// Falls back to name comparison if pointer match fails (e.g. after copy).
func indexOf(workflows []store.Workflow, wf *store.Workflow) int {
	for i := range workflows {
		if &workflows[i] == wf {
			return i
		}
	}
	// Fallback: match by name
	for i := range workflows {
		if workflows[i].Name == wf.Name {
			return i
		}
	}
	return -1
}
