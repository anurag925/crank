package bootstrap

// FeaturesUsed reports the features that contributed to this generation, in the
// order they were applied.
func (r *Result) FeaturesUsed() []string {
	if r == nil {
		return nil
	}
	if len(r.Features) > 0 {
		return r.Features
	}
	// Fall back to scanning for the manifest on disk.
	m, err := readManifest(r.ProjectDir)
	if err != nil {
		return nil
	}
	return m.Features
}
