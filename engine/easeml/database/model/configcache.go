package model

import (
	"github.com/ds3lab/easeml/engine/easeml/database/model/types"
	"github.com/globalsign/mgo/bson"
)

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
func (context Context) GetConfigCaches(models []string) (result []types.ConfigCache, err error) {

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
