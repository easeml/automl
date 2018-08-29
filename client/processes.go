package client

import (
	"github.com/ds3lab/easeml/database/model"
	"encoding/json"

	"github.com/pkg/errors"
)

// GetProcesses returns all processes from the service.
func (context Context) GetProcesses(status string) (result []model.Process, err error) {

	result = []model.Process{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if status != "" {
			query["status"] = status
		}
		resp, err := context.sendAPIGetRequest("processes", query)
		if err != nil {
			return nil, err
		}

		type getResponse struct {
			Data     []model.Process          `json:"data"`
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
