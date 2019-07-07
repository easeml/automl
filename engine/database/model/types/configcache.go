package types

// ConfigCache is a structure that holds all measurements obtained for a certain model and its
// configuration.
type ConfigCache struct {
	Model      string                        `bson:"model" json:"model"`
	Config     string                        `bson:"config" json:"config"`
	AvgQuality float64                       `bson:"avg-quality" json:"avg-quality"`
	Quality    map[string]map[string]float64 `bson:"quality" json:"quality"`
}

// Distance measures the distance between two config points by comparing their training history.
func (c1 ConfigCache) Distance(c2 ConfigCache) float64 {
	var sum float64
	for dataset, v1 := range c1.Quality {
		if v2, ok := c2.Quality[dataset]; ok {
			for objective, quality1 := range v1 {
				if quality2, ok := v2[objective]; ok {
					sum += quality1 * quality2
				}
			}
		}
	}
	return sum
}
