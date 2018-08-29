package model

import (
	"github.com/globalsign/mgo/bson"
)

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

var pipelineQuery = []bson.M{
	bson.M{"$group": bson.M{
		"_id":         bson.M{"model": "$model", "config": "$config", "dataset": "$dataset", "objective": "$objective"},
		"avg-quality": bson.M{"$avg": "$quality"},
		"cum-quality": bson.M{"$sum": "$quality"},
		"count":       bson.M{"$sum": 1}},
	},
	bson.M{
		"$group": bson.M{
			"_id":         bson.M{"model": "$_id.model", "config": "$_id.config", "dataset": "$_id.dataset"},
			"cum-quality": bson.M{"$sum": "$cum-quality"},
			"count":       bson.M{"$sum": "$count"},
			"quality":     bson.M{"$push": bson.M{"k": "$_id.objective", "v": "$avg-quality"}},
		},
	},
	bson.M{
		"$group": bson.M{
			"_id":         bson.M{"model": "$_id.model", "config": "$_id.config"},
			"cum-quality": bson.M{"$sum": "$cum-quality"},
			"count":       bson.M{"$sum": "$count"},
			"quality":     bson.M{"$push": bson.M{"k": "$_id.dataset", "v": bson.M{"$arrayToObject": "$quality"}}},
		},
	},
	bson.M{
		"$project": bson.M{
			"_id":         0,
			"model":       "$_id.model",
			"config":      "$_id.config",
			"avg-quality": bson.M{"$divide": []string{"$cum-quality", "$count"}},
			"count":       1,
			"quality":     bson.M{"$arrayToObject": "$quality"},
		},
	}}

// GetConfigCaches returns a list of config for a given set of models. If models is nil,
// then all models are considered.
func (context Context) GetConfigCaches(models []string) (result []ConfigCache, err error) {

	pipeline := pipelineQuery

	// If the filter was specified, prepend it to the pipeline.
	if len(models) > 0 {
		filter := bson.M{
			"$match": bson.M{
				"model": bson.M{"$in": models},
			},
		}
		pipeline = append([]bson.M{filter}, pipeline...)
	}

	c := context.Session.DB(context.DBName).C("tasks")
	err = c.Pipe(pipeline).All(&result)
	return
}
