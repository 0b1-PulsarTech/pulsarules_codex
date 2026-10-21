package registry

func Rules() map[string]string {
	rules := make(map[string]string)
	for _, spec := range specs() {
		if spec.Rule != "" {
			rules[spec.ID] = spec.Rule
		}
	}
	return rules
}
