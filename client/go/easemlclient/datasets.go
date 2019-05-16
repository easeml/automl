package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/ds3lab/easeml/client/go/easemlclient/types"

	tus "github.com/eventials/go-tus"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

// GetDatasets returns all datasets from the service.
func (context Context) GetDatasets(status, source, schemaIn, schemaOut string) (result []types.Dataset, err error) {

	result = []types.Dataset{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if status != "" {
			query["status"] = status
		}
		if source != "" {
			query["source"] = source
		}
		if schemaIn != "" {
			query["schema-in"] = schemaIn
		}
		if schemaOut != "" {
			query["schema-out"] = schemaOut
		}
		resp, err := context.sendAPIGetRequest("datasets", query)
		if err != nil {
			return nil, err
		}

		type getResponse struct {
			Data     []types.Dataset          `json:"data"`
			Metadata types.CollectionMetadata `json:"metadata"`
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

// GetDatasetByID returns a dataset given its ID.
func (context Context) GetDatasetByID(id string) (result *types.Dataset, err error) {

	resp, err := context.sendAPIGetRequest(path.Join("datasets", id), nil)
	if err != nil {
		return nil, err
	}

	type getDatasetByIDResponse struct {
		Data types.Dataset `json:"data"`
	}
	respObject := getDatasetByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return nil, errors.Wrap(err, "JSON decode error")
	}

	return &respObject.Data, nil
}

// CreateDataset creates a new dataset given the provided parameters.
func (context Context) CreateDataset(id, name, description, source, sourceAddress string) (string, error) {

	if id == "" {
		panic("id argument cannot be empty")
	}
	if DatasetSourceValid(source) == false {
		panic("invalid data source: " + source)
	}
	if DatasetSourceAddressRequired(source) {
		if sourceAddress == "" {
			panic("data source address expected")
		}
	} else {
		sourceAddress = ""
	}
	if nameRegex.MatchString(name) == false {
		return "", errors.New("invalid dataset name")
	}

	dataset := types.Dataset{
		ID:            id,
		Name:          name,
		Description:   description,
		Source:        source,
		SourceAddress: sourceAddress,
	}

	datasetBytes, err := json.Marshal(&dataset)
	if err != nil {
		return "", err
	}
	resp, err := context.sendAPIPostRequest("datasets", bytes.NewReader(datasetBytes), "application/json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract ID from location header.
	location := resp.Header.Get("Location")
	if location != "" {
		id = fmt.Sprintf("%s/%s", path.Base(path.Dir(location)), path.Base(location))
	}

	return id, nil
}

// UpdateDataset applies the given updates to the dataset fields.
func (context Context) UpdateDataset(id string, updates map[string]interface{}) (err error) {
	if id == "" {
		panic("id argument cannot be empty")
	}
	datasetBytes, err := json.Marshal(&updates)
	if err != nil {
		return err
	}
	url := path.Join("datasets", id)
	resp, err := context.sendAPIPatchRequest(url, bytes.NewReader(datasetBytes), "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ValidDatasetSources is a list of possible dataset sources.
var ValidDatasetSources = []string{
	types.DatasetUpload,
	types.DatasetDownload,
	types.DatasetLocal,
}

// DatasetSourceValid checks if the provided dataset source is valid.
func DatasetSourceValid(source string) bool {
	for i := range ValidDatasetSources {
		if source == ValidDatasetSources[i] {
			return true
		}
	}
	return false
}

// DatasetSourceAddressRequired returns true if the source address property is required for a given source.
func DatasetSourceAddressRequired(source string) bool {
	switch source {
	case types.DatasetUpload:
		return false
	case types.DatasetLocal:
		return true
	case types.DatasetDownload:
		return true
	default:
		panic("unknown source")
	}
}

// UploadDataset uploads the dataset to the server.
func (context Context) UploadDataset(id, sourcePath string) error {

	// Assemble the upload URL.
	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, "datasets", id, "upload"),
	}

	client, err := tus.NewClient(reqURL.String(), nil)
	if err != nil {
		return err
	}
	context.UserCredentials.Apply(client.Header)
	var upload *tus.Upload

	// First check if the dataset exists.
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(sourcePath)
	if err != nil {
		err = errors.Wrap(err, "dataset access error")
		return err
	}

	// If our source path is a file, we simply uload it.
	// TODO: Check if the file is a valid archive.
	if fileInfo.IsDir() == false {
		file, err := os.Open(sourcePath)
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return err
		}
		defer file.Close()

		upload, err = tus.NewUploadFromFile(file)
		if err != nil {
			err = errors.Wrap(err, "upload init error")
			return err
		}
	} else {
		// TODO: Maybe this should be configurable.
		arch := archiver.Zip

		// Enumerate all the sources.
		files, err := ioutil.ReadDir(sourcePath)
		if err != nil {
			err = errors.Wrap(err, "dataset access error")
			return err
		}
		sources := make([]string, len(files))
		for i := range files {
			sources[i] = filepath.Join(sourcePath, files[i].Name())
		}

		// Archive all the enumerated sources.
		var b bytes.Buffer
		arch.Write(&b, sources)
		data := b.Bytes()
		size := int64(len(data))
		upload = tus.NewUpload(bytes.NewReader(data), size, tus.Metadata{"filename": "data.zip"}, sourcePath)
		//upload = tus.NewUploadFromBytes(b.Bytes())
	}

	// Execute the upload.
	uploader, err := client.CreateUpload(upload)
	if err != nil {
		err = errors.Wrap(err, "dataset upload creation error")
		return err
	}
	err = uploader.Upload()
	if err != nil {
		err = errors.Wrap(err, "dataset upload error")
	}

	// Set status to be transferred.
	err = context.UpdateDataset(id, map[string]interface{}{"status": types.DatasetTransferred})
	if err != nil {
		err = errors.Wrap(err, "dataset update error")
	}

	return err
}
