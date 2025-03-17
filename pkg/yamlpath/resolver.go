package yamlpath

import "gopkg.in/yaml.v3"

type MergeState struct {
	InMergeBlock bool
	Aliases      []*yaml.Node
}

// MergeNodes will merge src -> dst while following proper YAML merge behavior
//
// This function will modify dst. Use a duplicate, dummy, dst node if needed.
func MergeNodes(dst, src *yaml.Node) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return // Only merge mapping nodes
	}

	// Create a map of keys in the destination node
	keyMap := make(map[string]int)
	for i := 0; i < len(dst.Content); i += 2 {
		key := dst.Content[i].Value
		keyMap[key] = i + 1 // Store the index of the value
	}

	// Merge keys from the source node into the destination node
	for i := 0; i < len(src.Content); i += 2 {
		key := src.Content[i].Value
		if key == "<<" {
			continue // Skip merge keys as we handled them
		}

		// If the key already exists in the destination, skip it
		// Keys in mapping nodes earlier in the sequence override keys specified in later mapping nodes
		// Additionally, explicit keys take precedence over merge keys
		if _, exists := keyMap[key]; exists {
			continue
		}

		dst.Content = append(dst.Content, src.Content[i], src.Content[i+1])
	}
}

// ProcessMappingNode will extract the mergeState if found within the given map
//
// Condition: node.Kind == yaml.MappingNode
func ProcessMappingNode(node *yaml.Node, mergeState *MergeState) {
	// Resolve aliases in the node itself
	//if node.Kind == yaml.AliasNode {
	//	node = node.Alias
	//}

	// Process the node's content to handle merge keys
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Value == "<<" {
			mergeValue := node.Content[i+1]
			if mergeValue.Kind == yaml.AliasNode {
				// <<: *alias
				mergeState.InMergeBlock = true
				mergeState.Aliases = append(mergeState.Aliases, mergeValue.Alias)
			} else if mergeValue.Kind == yaml.SequenceNode {
				// <<: [*alias_one, *alias_two, ...]
				for _, aliasNode := range mergeValue.Content {
					if aliasNode.Kind == yaml.AliasNode {
						mergeState.InMergeBlock = true
						mergeState.Aliases = append(mergeState.Aliases, aliasNode.Alias)
					}
				}
			}

			// Remove the merge key and its value (don't modify the original node)
			//node.Content = append(node.Content[:i], node.Content[i+2:]...)
			//i -= 2 // Adjust the index after removal
			i += 1 // skip alias that we just dealt with
		}
	}
}

// ProcessNode will handle the following cases without modifying the original node:
//
// - yaml.AliasNode: Replacing an alias with its underlying node
// - yaml.MappingNode: Embed any merge keys inside its node.Content (returns dummy rendered node)
func ProcessNode(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}

	// Embed merge keys if the node is a mapping node
	if node.Kind == yaml.MappingNode {
		var mergeState MergeState
		ProcessMappingNode(node, &mergeState)

		if mergeState.InMergeBlock {
			mergedNode := &yaml.Node{
				Kind:    node.Kind,
				Content: append([]*yaml.Node{}, node.Content...), // Copy original content
			}
			for _, alias := range mergeState.Aliases {
				// Ensure this node doesn't have any merges...
				alias = ProcessNode(alias)
				MergeNodes(mergedNode, alias)
			}
			node = mergedNode
		}
	}

	return node
}
