package client

import (
	"bytes"
	"github.com/ds3lab/easeml/database/model"
	"encoding/json"
	"path"

	"github.com/pkg/errors"
)

// GetJobs returns all jobs from the service.
func (context Context) GetJobs(user, status, job, objective, modelName string) (result []model.Job, err error) {

	result = []model.Job{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if user != "" {
			query["user"] = user
		}
		if status != "" {
			query["status"] = status
		}
		if job != "" {
			query["job"] = job
		}
		if objective != "" {
			query["objective"] = objective
		}
		if modelName != "" {
			query["model"] = modelName
		}
		resp, err := context.sendAPIGetRequest("jobs", query)
		if err != nil {
			return nil, err
		}

		type getResponse struct {
			Data     []model.Job              `json:"data"`
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

// GetJobByID returns a job given its ID.
func (context Context) GetJobByID(id string) (result *model.Job, err error) {

	resp, err := context.sendAPIGetRequest(path.Join("jobs", id), nil)
	if err != nil {
		return nil, err
	}

	type getJobByIDResponse struct {
		Data model.Job `json:"data"`
	}
	respObject := getJobByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return nil, errors.Wrap(err, "JSON decode error")
	}

	return &respObject.Data, nil
}

// CreateJob creates a new job given the provided parameters.
func (context Context) CreateJob(dataset, objective string, models []string, altObjectives []string, acceptNewModels bool, maxTasks uint64) (string, error) {

	// TODO: Perform checks.

	job := model.Job{
		Dataset:         dataset,
		Objective:       objective,
		Models:          models,
		AltObjectives:   altObjectives,
		AcceptNewModels: acceptNewModels,
		MaxTasks:        maxTasks,
	}

	jobBytes, err := json.Marshal(&job)
	if err != nil {
		return "", err
	}
	resp, err := context.sendAPIPostRequest("jobs", bytes.NewReader(jobBytes), "application/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract job ID if possible.
	id := ""
	location := resp.Header.Get("Location")
	if location != "" {
		id = path.Base(location)
	}

	return id, nil
}
