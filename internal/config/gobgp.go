package config

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// normalizeNeighborEndpointPorts maps local peer-port aliases before strict
// decoding so the runtime can keep using GoBGP's upstream config structs.
func normalizeNeighborEndpointPorts(data []byte) ([]byte, error) {
	var node yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&node); err != nil {
		return nil, err
	}
	if node.Kind == 0 {
		return data, nil
	}

	if err := applyNeighborEndpointPortAliases(&node); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func applyNeighborEndpointPortAliases(document *yaml.Node) error {
	root := document
	if document.Kind == yaml.DocumentNode {
		if len(document.Content) == 0 {
			return nil
		}
		root = document.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}

	gobgp := mappingValue(root, "gobgp")
	if gobgp == nil || gobgp.Kind != yaml.MappingNode {
		return nil
	}

	neighbors := mappingValue(gobgp, "neighbors")
	if neighbors == nil || neighbors.Kind != yaml.SequenceNode {
		return nil
	}

	for i, neighbor := range neighbors.Content {
		if neighbor.Kind != yaml.MappingNode {
			continue
		}
		if err := applyNeighborEndpointPortAlias(neighbor, i); err != nil {
			return err
		}
	}

	return nil
}

func applyNeighborEndpointPortAlias(neighbor *yaml.Node, index int) error {
	if err := normalizeRemotePortKey(neighbor, index); err != nil {
		return err
	}

	configNode := mappingValue(neighbor, "config")
	if configNode == nil || configNode.Kind != yaml.MappingNode {
		return nil
	}

	portNode := mappingValue(configNode, "port")
	if portNode == nil {
		return nil
	}

	label := neighborLabel(neighbor, index)
	endpointPort, err := parseNeighborPort(portNode, false)
	if err != nil {
		return fmt.Errorf("%w for neighbor %s: %q", ErrInvalidNeighborPort, label, portNode.Value)
	}

	transportNode := ensureMappingValue(neighbor, "transport")
	if transportNode == nil {
		return fmt.Errorf("%w for neighbor %s: transport must be a mapping", ErrInvalidNeighborPort, label)
	}
	transportConfigNode := ensureMappingValue(transportNode, "config")
	if transportConfigNode == nil {
		return fmt.Errorf("%w for neighbor %s: transport.config must be a mapping", ErrInvalidNeighborPort, label)
	}
	remotePortNode := mappingValue(transportConfigNode, "remoteport")
	if remotePortNode != nil {
		remotePort, err := parseNeighborPort(remotePortNode, true)
		if err != nil {
			return fmt.Errorf("%w for neighbor %s: %q", ErrInvalidNeighborPort, label, remotePortNode.Value)
		}
		if remotePort != endpointPort {
			return fmt.Errorf("%w for neighbor %s: config.port=%d transport.config.remoteport=%d", ErrConflictingNeighborPort, label, endpointPort, remotePort)
		}
	} else {
		setMappingScalar(transportConfigNode, "remoteport", strconv.FormatUint(uint64(endpointPort), 10))
	}

	removeMappingKey(configNode, "port")
	return nil
}

func normalizeRemotePortKey(neighbor *yaml.Node, index int) error {
	transportNode := mappingValue(neighbor, "transport")
	if transportNode == nil || transportNode.Kind != yaml.MappingNode {
		return nil
	}
	transportConfigNode := mappingValue(transportNode, "config")
	if transportConfigNode == nil || transportConfigNode.Kind != yaml.MappingNode {
		return nil
	}

	remotePortNode := mappingValue(transportConfigNode, "remoteport")
	hyphenRemotePortNode := mappingValue(transportConfigNode, "remote-port")
	if hyphenRemotePortNode == nil {
		return nil
	}

	label := neighborLabel(neighbor, index)
	hyphenRemotePort, err := parseNeighborPort(hyphenRemotePortNode, true)
	if err != nil {
		return fmt.Errorf("%w for neighbor %s: %q", ErrInvalidNeighborPort, label, hyphenRemotePortNode.Value)
	}

	if remotePortNode != nil {
		remotePort, err := parseNeighborPort(remotePortNode, true)
		if err != nil {
			return fmt.Errorf("%w for neighbor %s: %q", ErrInvalidNeighborPort, label, remotePortNode.Value)
		}
		if remotePort != hyphenRemotePort {
			return fmt.Errorf("%w for neighbor %s: transport.config.remoteport=%d transport.config.remote-port=%d", ErrConflictingNeighborPort, label, remotePort, hyphenRemotePort)
		}
		removeMappingKey(transportConfigNode, "remote-port")
		return nil
	}

	renameMappingKey(transportConfigNode, "remote-port", "remoteport")
	return nil
}

func parseNeighborPort(portNode *yaml.Node, allowZero bool) (uint16, error) {
	if portNode.Kind != yaml.ScalarNode {
		return 0, ErrInvalidNeighborPort
	}

	port, err := strconv.ParseUint(strings.TrimSpace(portNode.Value), 10, 16)
	if err != nil {
		return 0, ErrInvalidNeighborPort
	}
	if port == 0 && !allowZero {
		return 0, ErrInvalidNeighborPort
	}

	return uint16(port), nil
}

func neighborLabel(neighbor *yaml.Node, index int) string {
	configNode := mappingValue(neighbor, "config")
	if configNode != nil && configNode.Kind == yaml.MappingNode {
		if addressNode := mappingValue(configNode, "neighboraddress"); addressNode != nil && addressNode.Value != "" {
			return strconv.Quote(addressNode.Value)
		}
		if addressNode := mappingValue(configNode, "neighbor-address"); addressNode != nil && addressNode.Value != "" {
			return strconv.Quote(addressNode.Value)
		}
	}

	return fmt.Sprintf("#%d", index+1)
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func ensureMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	if value := mappingValue(mapping, key); value != nil {
		if value.Kind != yaml.MappingNode {
			return nil
		}
		return value
	}

	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	valueNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
	return valueNode
}

func setMappingScalar(mapping *yaml.Node, key, value string) {
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	valueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: value,
	}
	mapping.Content = append(mapping.Content, keyNode, valueNode)
}

func removeMappingKey(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

func renameMappingKey(mapping *yaml.Node, oldKey, newKey string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value == oldKey {
			mapping.Content[i].Value = newKey
			return
		}
	}
}
