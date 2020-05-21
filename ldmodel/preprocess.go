package ldmodel

type targetPreprocessedData struct {
	valuesMap map[string]bool
}

type segmentPreprocessedData struct {
	includeMap map[string]bool
	excludeMap map[string]bool
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

// PreprocessSegment precomputes internal data structures based on the segment configuration, to speed up
// evaluations.
//
// This is called once after a segment is deserialized from JSON, or is created with ldbuilders. If you
// construct a segment by some other means, you should call PreprocessSegment exactly once before making
// it available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessSegment(s *Segment) {
	p := segmentPreprocessedData{}
	if len(s.Included) > 0 {
		p.includeMap = make(map[string]bool, len(s.Included))
		for _, key := range s.Included {
			p.includeMap[key] = true
		}
	}
	if len(s.Excluded) > 0 {
		p.excludeMap = make(map[string]bool, len(s.Excluded))
		for _, key := range s.Excluded {
			p.excludeMap[key] = true
		}
	}
	s.preprocessed = p
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
