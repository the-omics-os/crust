package ontologybrowser

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

func searchKnownNodes(query string, roots []OntologyNode) []searchResult {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	var results []searchResult
	collectSearchResults(roots, trimmed, &results)

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		left := findNodeByID(roots, results[i].NodeID)
		right := findNodeByID(roots, results[j].NodeID)
		switch {
		case left == nil:
			return false
		case right == nil:
			return true
		case left.Depth != right.Depth:
			return left.Depth < right.Depth
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})

	return results
}

func collectSearchResults(nodes []OntologyNode, query string, results *[]searchResult) {
	for i := range nodes {
		if score, ok := scoreSearch(nodes[i], query); ok {
			*results = append(*results, searchResult{
				NodeID: nodes[i].ID,
				Score:  score,
			})
		}
		collectSearchResults(nodes[i].Children, query, results)
	}
}

func scoreSearch(node OntologyNode, query string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return 0, false
	}

	name := strings.ToLower(node.Name)
	id := strings.ToLower(node.ID)
	desc := strings.ToLower(node.Description)
	score := 0

	if name == q {
		score = 1000
	}
	if idx := strings.Index(name, q); idx >= 0 {
		score = maxInt(score, 900-idx*2)
	}
	if idx := strings.Index(id, q); idx >= 0 {
		score = maxInt(score, 840-idx)
	}
	if idx := strings.Index(desc, q); idx >= 0 {
		score = maxInt(score, 680-idx/4)
	}

	tokens := strings.Fields(q)
	if tokensMatch([]string{name, id, desc}, tokens) {
		score = maxInt(score, 560)
	}
	if score == 0 && fuzzySubsequence(q, name) {
		score = 360
	}
	if score == 0 && fuzzySubsequence(q, id) {
		score = 320
	}

	return score, score > 0
}

func tokensMatch(texts []string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}

	for _, token := range tokens {
		found := false
		for _, text := range texts {
			if strings.Contains(text, token) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func fuzzySubsequence(query, text string) bool {
	qrunes := []rune(strings.ToLower(query))
	if len(qrunes) == 0 {
		return false
	}

	idx := 0
	for _, r := range strings.ToLower(text) {
		if unicode.IsSpace(r) {
			continue
		}
		if idx < len(qrunes) && r == qrunes[idx] {
			idx++
			if idx == len(qrunes) {
				return true
			}
		}
	}
	return false
}

func searchHighlightToken(query string) string {
	tokens := strings.Fields(strings.TrimSpace(query))
	if len(tokens) == 0 {
		return ""
	}

	longest := tokens[0]
	for _, token := range tokens[1:] {
		if utf8.RuneCountInString(token) > utf8.RuneCountInString(longest) {
			longest = token
		}
	}
	return longest
}

func highlightMatch(text, token string, base, highlight lipgloss.Style) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return base.Render(text)
	}

	lowerText := strings.ToLower(text)
	lowerToken := strings.ToLower(token)
	idx := strings.Index(lowerText, lowerToken)
	if idx < 0 {
		return base.Render(text)
	}

	end := idx + len(lowerToken)
	return base.Render(text[:idx]) + highlight.Render(text[idx:end]) + base.Render(text[end:])
}
