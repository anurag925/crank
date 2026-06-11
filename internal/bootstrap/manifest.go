package bootstrap

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func parseManifest(data []byte) (*manifest, error) {
	m := &manifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, err
	}
	return m, nil
}

func encodeManifest(m *manifest) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m); err != nil {
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}
