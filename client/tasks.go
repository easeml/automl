package client

import (
	"github.com/ds3lab/easeml/database/model"
	"encoding/json"
	"path"

	"github.com/pkg/errors"
)

// GetTasks returns all tasks from the service.
func (context Context) GetTasks(job, user, status, stage, dataset, objective, modelName string) (result []model.Task, err error) {

	result = []model.Task{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if job != "" {
			query["job"] = job
		}
		if user != "" {
			query["user"] = user
		}
		if status != "" {
			query["status"] = status
		}
		if stage != "" {
			query["stage"] = stage
		}
		if dataset != "" {
			query["dataset"] = dataset
		}
		if objective != "" {
			query["objective"] = objective
		}
		if modelName != "" {
			query["model"] = modelName
		}
		resp, err := context.sendAPIGetRequest("tasks", query)
		if err != nil {
			return nil, err
		}

		type getResponse struct {
			Data     []model.Task             `json:"data"`
			Metadata model.CollectionMetadata `json:"metadata"`
			Links    map[string]string        `json:"links"`
		}
		respObject := getResponse{}
		err = json.NewDecoder(resp.Body).Decode(&respObject)
		if err != nil {
			return nil, errors.Wrap(err, "JSON decode error")
		}
		nextCursor = respObject.Metadata.NextPageCursor
		result = append(result, respObject.Data...)

		if nextCursor == "" || len(respObject.Data) == 0 {
			break
		}
	}

	return result, nil
}

// GetTaskByID returns a task given its ID.
func (context Context) GetTaskByID(id string) (result *model.Task, err error) {

	resp, err := context.sendAPIGetRequest(path.Join("tasks", id), nil)
	if err != nil {
		return nil, err
	}

	type getTaskByIDResponse struct {
		Data model.Task `json:"data"`
	}
	respObject := getTaskByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return nil, errors.Wrap(err, "JSON decode error")
	}

	return &respObject.Data, nil
}
