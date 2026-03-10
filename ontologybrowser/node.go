package ontologybrowser

// OntologyNode represents a single ontology term.
//
// Children use nil vs empty slices to distinguish "not loaded yet" from
// "loaded and confirmed leaf".
type OntologyNode struct {
	ID          string
	Name        string
	Description string
	Children    []OntologyNode
	Loaded      bool
	Depth       int
}

func cloneNodes(nodes []OntologyNode) []OntologyNode {
	if nodes == nil {
		return nil
	}

	cloned := make([]OntologyNode, len(nodes))
	for i := range nodes {
		cloned[i] = cloneNode(nodes[i])
	}
	return cloned
}

func cloneNode(node OntologyNode) OntologyNode {
	cloned := node
	if node.Children != nil {
		cloned.Children = cloneNodes(node.Children)
	}
	return cloned
}

func normalizeNodes(nodes []OntologyNode, depth int) []OntologyNode {
	if nodes == nil {
		return nil
	}

	normalized := make([]OntologyNode, len(nodes))
	for i := range nodes {
		normalized[i] = normalizeNode(nodes[i], depth)
	}
	return normalized
}

func normalizeNode(node OntologyNode, depth int) OntologyNode {
	normalized := OntologyNode{
		ID:          node.ID,
		Name:        node.Name,
		Description: node.Description,
		Loaded:      node.Loaded,
		Depth:       depth,
	}

	switch {
	case node.Children != nil:
		normalized.Children = normalizeNodes(node.Children, depth+1)
		normalized.Loaded = true
	case node.Loaded:
		normalized.Children = []OntologyNode{}
	default:
		normalized.Children = nil
	}

	return normalized
}

func findNodeByID(nodes []OntologyNode, id string) *OntologyNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
		if found := findNodeByID(nodes[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}

func parentOfNode(nodes []OntologyNode, id string) *OntologyNode {
	_, parent := findNodeAndParent(nodes, id, nil)
	return parent
}

func findNodeAndParent(nodes []OntologyNode, id string, parent *OntologyNode) (*OntologyNode, *OntologyNode) {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i], parent
		}
		if found, parent := findNodeAndParent(nodes[i].Children, id, &nodes[i]); found != nil {
			return found, parent
		}
	}
	return nil, nil
}

func flattenVisibleNodes(nodes []OntologyNode, expanded map[string]bool) []visibleNode {
	var flattened []visibleNode
	for i := range nodes {
		flattened = append(flattened, visibleNode{
			NodeID: nodes[i].ID,
			Depth:  nodes[i].Depth,
		})
		if expanded[nodes[i].ID] && len(nodes[i].Children) > 0 {
			flattened = append(flattened, flattenVisibleNodes(nodes[i].Children, expanded)...)
		}
	}
	return flattened
}

func flattenMatchingPaths(nodes []OntologyNode, include map[string]bool) []visibleNode {
	var flattened []visibleNode
	for i := range nodes {
		if !include[nodes[i].ID] {
			continue
		}
		flattened = append(flattened, visibleNode{
			NodeID: nodes[i].ID,
			Depth:  nodes[i].Depth,
		})
		if len(nodes[i].Children) > 0 {
			flattened = append(flattened, flattenMatchingPaths(nodes[i].Children, include)...)
		}
	}
	return flattened
}

func includedPathSet(nodes []OntologyNode, matches []searchResult) map[string]bool {
	include := make(map[string]bool, len(matches))
	for _, match := range matches {
		path := pathToNode(nodes, match.NodeID)
		for _, node := range path {
			include[node.ID] = true
		}
	}
	return include
}

func pathToNode(nodes []OntologyNode, id string) []OntologyNode {
	var path []OntologyNode
	if collectPath(nodes, id, &path) {
		return path
	}
	return nil
}

func collectPath(nodes []OntologyNode, id string, path *[]OntologyNode) bool {
	for i := range nodes {
		snapshot := nodes[i]
		snapshot.Children = nil

		*path = append(*path, snapshot)
		if nodes[i].ID == id {
			return true
		}
		if collectPath(nodes[i].Children, id, path) {
			return true
		}
		*path = (*path)[:len(*path)-1]
	}
	return false
}

func pathIDs(path []OntologyNode) []string {
	if len(path) == 0 {
		return nil
	}

	ids := make([]string, len(path))
	for i, node := range path {
		ids[i] = node.ID
	}
	return ids
}

func pathLabels(path []OntologyNode) []string {
	if len(path) == 0 {
		return nil
	}

	labels := make([]string, len(path))
	for i, node := range path {
		if node.Name != "" {
			labels[i] = node.Name
			continue
		}
		labels[i] = node.ID
	}
	return labels
}

func countNodes(nodes []OntologyNode) int {
	total := 0
	for i := range nodes {
		total++
		total += countNodes(nodes[i].Children)
	}
	return total
}
