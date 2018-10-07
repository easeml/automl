package client

import (
	"bytes"
	ctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/ds3lab/easeml/database/model"

	"github.com/docker/docker/client"
	tus "github.com/eventials/go-tus"
	"github.com/pkg/errors"
)

// GetModules returns all modules from the service.
func (context Context) GetModules(moduleType, user, status, source, schemaIn, schemaOut string) (result []model.Module, err error) {

	result = []model.Module{}
	nextCursor := ""

	for {

		query := map[string]string{}
		if nextCursor != "" {
			query["cursor"] = nextCursor
		}
		if moduleType != "" {
			query["type"] = moduleType
		}
		if user != "" {
			query["user"] = user
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
		resp, err := context.sendAPIGetRequest("modules", query)
		if err != nil {
			return nil, err
		}

		type getResponse struct {
			Data     []model.Module           `json:"data"`
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

// GetModuleByID returns a module given its ID.
func (context Context) GetModuleByID(id string) (result *model.Module, err error) {

	resp, err := context.sendAPIGetRequest(path.Join("modules", id), nil)
	if err != nil {
		return nil, err
	}

	type getModuleByIDResponse struct {
		Data model.Module `json:"data"`
	}
	respObject := getModuleByIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respObject)
	if err != nil {
		return nil, errors.Wrap(err, "JSON decode error")
	}

	return &respObject.Data, nil
}

// CreateModule creates a new module given the provided parameters.
func (context Context) CreateModule(id, moduleType, label, name, description, source, sourceAddress string) (string, error) {

	if id == "" {
		panic("id argument cannot be empty")
	}
	if ModuleTypeValid(moduleType) == false {
		panic("invalid module type: " + moduleType)
	}
	if ModuleSourceValid(source) == false {
		panic("invalid data source: " + source)
	}
	if ModuleSourceAddressRequired(source) && sourceAddress == "" {
		panic("data source address expected")
	}
	if nameRegex.MatchString(name) == false {
		return "", errors.New("invalid module name")
	}

	module := model.Module{
		ID:            id,
		Type:          moduleType,
		Label:         label,
		Name:          name,
		Description:   description,
		Source:        source,
		SourceAddress: sourceAddress,
	}

	moduleBytes, err := json.Marshal(&module)
	if err != nil {
		return "", err
	}
	resp, err := context.sendAPIPostRequest("modules", bytes.NewReader(moduleBytes), "application/json")
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

// UpdateModule applies the given updates to the module fields.
func (context Context) UpdateModule(id string, updates map[string]interface{}) (err error) {
	if id == "" {
		panic("id argument cannot be empty")
	}
	moduleBytes, err := json.Marshal(&updates)
	if err != nil {
		return err
	}
	url := path.Join("modules", id)
	resp, err := context.sendAPIPatchRequest(url, bytes.NewReader(moduleBytes), "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ValidModuleTypes is a list of possible module types.
var ValidModuleTypes = []string{
	model.ModuleModel,
	model.ModuleObjective,
	model.ModuleOptimizer,
}

// ModuleTypeValid checks if the provided module type is valid.
func ModuleTypeValid(moduleType string) bool {
	for i := range ValidModuleTypes {
		if moduleType == ValidModuleTypes[i] {
			return true
		}
	}
	return false
}

// ValidModuleSources is a list of possible module sources.
var ValidModuleSources = []string{
	model.ModuleUpload,
	model.ModuleDownload,
	model.ModuleLocal,
	model.ModuleRegistry,
}

// ModuleSourceValid checks if the provided module source is valid.
func ModuleSourceValid(source string) bool {
	for i := range ValidModuleSources {
		if source == ValidModuleSources[i] {
			return true
		}
	}
	return false
}

// ModuleSourceAddressRequired returns true if the source address property is required for a given source.
func ModuleSourceAddressRequired(source string) bool {
	switch source {
	case model.ModuleUpload:
		return false
	case model.ModuleLocal:
		return true
	case model.ModuleDownload:
		return true
	default:
		panic("unknown source")
	}
}

// UploadModule uploads the module to the server.
func (context Context) UploadModule(id, sourcePath string) error {

	// Assemble the upload URL.
	reqURL := url.URL{
		Scheme: "http",
		Host:   context.ServerAddress,
		Path:   path.Join(apiPrefix, "modules", id, "upload"),
	}

	tusClient, err := tus.NewClient(reqURL.String(), nil)
	if err != nil {
		return err
	}
	context.UserCredentials.Apply(tusClient.Header)

	// TODO: Get API version automatically.
	// See: https://stackoverflow.com/a/48638182
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		panic(err)
	}

	// Save the image as a TAR.
	reader, err := cli.ImageSave(ctx.Background(), []string{sourcePath})
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		err = errors.Wrap(err, "reader access error")
		return err
	}
	size := int64(len(data))

	// TODO: Is this resumeable?
	upload := tus.NewUpload(bytes.NewReader(data), size, tus.Metadata{"filename": "module.tar"}, sourcePath)
	uploader, err := tusClient.CreateUpload(upload)
	if err != nil {
		err = errors.Wrap(err, "module upload creation error")
		return err
	}

	err = uploader.Upload()
	if err != nil {
		err = errors.Wrap(err, "module upload error")
	}

	// Set status to be transferred.
	err = context.UpdateModule(id, map[string]interface{}{"status": model.ModuleTransferred})
	if err != nil {
		err = errors.Wrap(err, "module update error")
	}

	return err
}
