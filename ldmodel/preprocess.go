package ldmodel

type targetPreprocessedData struct {
	valuesMap map[string]bool
}

// PreprocessFlag precomputes internal data structures based on the flag configuration, to speed up
// evaluations.
//
// This is called once after a flag is deserialized from JSON, or is created with ldbuilders. If you
// construct a flag by some other means, you should call PreprocessFlag exactly once before making it
// available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessFlag(f *FeatureFlag) {
	for i, t := range f.Targets {
		f.Targets[i].preprocessed = preprocessTarget(t)
	}
}

func preprocessTarget(t Target) targetPreprocessedData {
	ret := targetPreprocessedData{}
	if len(t.Values) > 0 {
		m := make(map[string]bool, len(t.Values))
		for _, v := range t.Values {
			m[v] = true
		}
		ret.valuesMap = m
	}
	return ret
}
