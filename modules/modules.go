package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ds3lab/easeml/schema/dataset"
	"github.com/ds3lab/easeml/schema/schema"
	"github.com/ds3lab/easeml/storage"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// InferModuleProperties takes a module available on the local docker instance and tries to infer
// its basic properties such as id, name and description.
func InferModuleProperties(sourcePath string) (id, name, description, schemaIn, schemaOut, configSpace string, err error) {

	// Extract id from source path.
	id = strings.Split(sourcePath, "@")[0] // Get rid of the digest.
	splits := strings.Split(id, "/")
	id = splits[len(splits)-1]             // If there are more slash separated elements, take the last one.
	id = strings.Replace(id, ".", "-", -1) // Replace dots with dashes.
	id = strings.Replace(id, ":", "-", -1) // Replace colons with dashes.

	// Extract all filenames from the image working dir.
	var outReader io.ReadCloser
	outReader, err = RunContainerAndCollectOutput(sourcePath, []string{"ls"}, []string{"."})
	if err != nil {
		err = errors.Wrap(err, "docker container start error")
		return
	}
	defer outReader.Close()
	var containerOutput []byte
	containerOutput, err = ioutil.ReadAll(outReader)
	if err != nil {
		err = errors.Wrap(err, "docker container output read error")
		return
	}
	workDirFiles := strings.Fields(string(containerOutput))

	// Look for files we need to read from the working directory.
	var readmeFileName, schemaInFileName, schemaOutFileName, configSpaceFilename string
	for i := range workDirFiles {

		if readmeFileName == "" {
			match, err := filepath.Match("README*", workDirFiles[i])
			if err != nil {
				panic(err) // This can only happen if the pattern is bad.
			}
			if match {
				readmeFileName = workDirFiles[i]
			}
		}
		if schemaInFileName == "" {
			match, err := filepath.Match("schema-in*", workDirFiles[i])
			if err != nil {
				panic(err) // This can only happen if the pattern is bad.
			}
			if match {
				schemaInFileName = workDirFiles[i]
			}
		}
		if schemaOutFileName == "" {
			match, err := filepath.Match("schema-out*", workDirFiles[i])
			if err != nil {
				panic(err) // This can only happen if the pattern is bad.
			}
			if match {
				schemaOutFileName = workDirFiles[i]
			}
		}
		if configSpaceFilename == "" {
			match, err := filepath.Match("config-space*", workDirFiles[i])
			if err != nil {
				panic(err) // This can only happen if the pattern is bad.
			}
			if match {
				configSpaceFilename = workDirFiles[i]
			}
		}
	}

	// If a README file was found, read it.
	if readmeFileName != "" {
		outReader, err = RunContainerAndCollectOutput(sourcePath, []string{"cat"}, []string{readmeFileName})
		if err != nil {
			err = errors.Wrap(err, "docker container start error")
			return
		}
		defer outReader.Close()
		name, description = storage.ScanReadme(outReader)
	}

	// Look for the schema file.
	if schemaInFileName != "" {
		outReader, err = RunContainerAndCollectOutput(sourcePath, []string{"cat"}, []string{schemaInFileName})
		if err != nil {
			err = errors.Wrap(err, "docker container start error")
			return
		}
		defer outReader.Close()
		schemaIn, err = getJSONFromReader(schemaInFileName, outReader)
		if err != nil {
			return
		}
	}
	if schemaOutFileName != "" {
		outReader, err = RunContainerAndCollectOutput(sourcePath, []string{"cat"}, []string{schemaOutFileName})
		if err != nil {
			err = errors.Wrap(err, "docker container start error")
			return
		}
		defer outReader.Close()
		schemaOut, err = getJSONFromReader(schemaOutFileName, outReader)
		if err != nil {
			return
		}
	}

	// Maybe this is not the best place to do this check. Maybe the caller should handle an empty schema.
	// A model must have a schema file, if none was found, return an error.
	/* if schema == "" {
		err = errors.New("each model must have a schema.json or schema.yaml file in its working directory")
		return
	} */

	// Look for the config space file.
	if configSpaceFilename != "" {
		outReader, err = RunContainerAndCollectOutput(sourcePath, []string{"cat"}, []string{configSpaceFilename})
		if err != nil {
			err = errors.Wrap(err, "docker container start error")
			return
		}
		defer outReader.Close()
		configSpace, err = getJSONFromReader(configSpaceFilename, outReader)
		if err != nil {
			return
		}
	}

	// A model must have a config space file, if none was found, return an error.
	// NOTE: Actually, a model without a config space can be ok. It simply means there are no hyperparameters.
	// if configSpace == "" {
	// 	err = errors.New("each model must have a config-space.json or config-space.yaml file in its working directory")
	// 	return
	// }

	return
}

func getJSONFromReader(fileName string, reader io.Reader) (string, error) {

	containerOutput, err := ioutil.ReadAll(reader)
	if err != nil {
		err = errors.Wrap(err, "docker container output read error")
		return "", err
	}
	if strings.HasSuffix(fileName, ".json") {

		var jsonSchemaBuf bytes.Buffer
		err = json.Compact(&jsonSchemaBuf, containerOutput)
		if err != nil {
			err = errors.Wrap(err, "json schema parse error")
			return "", err
		}
		return string(jsonSchemaBuf.Bytes()), nil

	} else if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
		var jsonSchema []byte
		jsonSchema, err = yaml.YAMLToJSON(containerOutput)
		if err != nil {
			err = errors.Wrap(err, "yaml schema parse error")
			return "", err
		}
		return string(jsonSchema), nil
	}

	return "", nil
}

// GetDockerClient returns an instance of the Docker client.
func GetDockerClient() *client.Client {
	// TODO: Get API version automatically.
	// See: https://stackoverflow.com/a/48638182
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		panic(err)
	}
	return cli
}

// MntPrefix must be placed before all command line arguments if they represent a local directory
// or file which we want to mount to the image.
const MntPrefix = "^^^"

// RunContainerAndCollectOutput runs a given image name and returns the standard output reader.
func RunContainerAndCollectOutput(imageName string, entrypoint, command []string) (io.ReadCloser, error) {

	// Go through all commands and see if any of them correspond to a file. If yes, mount it to
	// the container and remap the command argument.
	remappedCommand := make([]string, len(command))
	bindsMap := map[string]interface{}{}
	targetDirsMap := map[string]interface{}{}
	binds := []string{}
	for i := range command {

		if strings.HasPrefix(command[i], MntPrefix) {

			command[i] = strings.TrimPrefix(command[i], MntPrefix)

			if stats, err := os.Stat(command[i]); err == nil {

				// Get absolute path.
				absPath, err := filepath.Abs(command[i])
				if err != nil {
					err = errors.Wrap(err, "absolute path inference failed")
					return nil, err
				}

				// If it is a directory, we can immediately mount it. Otherwise we mount the parent.
				var mapping string
				if stats.IsDir() {
					dirName := filepath.Base(absPath)

					// If this dirname was already mounted, we rename it by appending underscores.
					for {
						if _, ok := targetDirsMap[dirName]; ok {
							dirName = dirName + "_"
						} else {
							break
						}
					}
					targetDirsMap[dirName] = nil

					mountedPath := filepath.Join("/mnt", dirName)

					remappedCommand[i] = mountedPath
					mapping = fmt.Sprintf("%s:%s", absPath, mountedPath)

				} else {
					fileName := filepath.Base(absPath)
					parentPath := filepath.Dir(absPath)
					dirName := filepath.Base(parentPath)

					// If this dirname was already mounted, we rename it by appending underscores.
					for {
						if _, ok := targetDirsMap[dirName]; ok {
							dirName = dirName + "_"
						} else {
							break
						}
					}
					targetDirsMap[dirName] = nil

					mountedPath := filepath.Join("/mnt", dirName)

					remappedCommand[i] = filepath.Join(mountedPath, fileName)
					mapping = fmt.Sprintf("%s:%s", parentPath, mountedPath)

				}

				// Add to binds if it doesn't exist yet.
				if _, ok := bindsMap[mapping]; ok == false {
					bindsMap[mapping] = nil
					binds = append(binds, mapping)
				}

				continue
			}
		}
		// We consider it not to be a file path.
		remappedCommand[i] = command[i]

	}

	ctx := context.Background()
	cli := GetDockerClient()
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:      imageName,
		Entrypoint: entrypoint,
		Cmd:        remappedCommand,
		Tty:        true,
	}, &container.HostConfig{
		Binds: binds,
	}, nil, "")
	if err != nil {
		panic(err)
	}

	defer func() {
		cli.ContainerRemove(
			context.Background(),
			resp.ID,
			types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	}()

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}
	return out, nil
}

// LoadImage loads a Docker image from a tar file.
func LoadImage(imageFilePath string) (string, error) {

	cli := GetDockerClient()
	imageFile, err := os.Open(imageFilePath)
	if err != nil {
		return "", err
	}
	resp, err := cli.ImageLoad(context.Background(), imageFile, false)
	if resp.JSON != true {
		panic("expected JSON response")
	}
	defer resp.Body.Close()

	type responseBodyType struct {
		Stream string `json:"stream"`
	}
	var result responseBodyType
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	imageName := strings.TrimSpace(strings.TrimPrefix(result.Stream, "Loaded image:"))

	return imageName, nil
}

// ValidateModel takes a model image and the input and output schema, generates a random input data set,
// runs train and predict on the model and verifies that the output data matches the output schema.
func ValidateModel(modelImageName string, schemaStringIn, schemaStringOut, configSpace string, cleanup bool) (err error) {

	// Parse schema JSON strings.
	var schemaStructIn, schemaStructOut interface{}
	err = json.Unmarshal([]byte(schemaStringIn), &schemaStructIn)
	if err != nil {
		err = errors.Wrap(err, "failed to decode input schema JSON")
		return
	}
	err = json.Unmarshal([]byte(schemaStringOut), &schemaStructOut)
	if err != nil {
		err = errors.Wrap(err, "failed to decode output schema JSON")
		return
	}
	schemaIn, err := schema.Load(schemaStructIn)
	if err != nil {
		err = errors.Wrap(err, "failed to load input schema")
		return
	}
	schemaOut, err := schema.Load(schemaStructOut)
	if err != nil {
		err = errors.Wrap(err, "failed to load output schema")
		return
	}

	// Generate temp directories.
	tempDirName, err := ioutil.TempDir("", "easeml_model_val")
	if err != nil {
		err = errors.Wrap(err, "failed to generate temp directory")
		return
	}
	err = os.MkdirAll(filepath.Join(tempDirName, "data", "input"), storage.DefaultFilePerm)
	if err != nil {
		err = errors.Wrap(err, "failed to generate temp data input directory")
		return
	}
	err = os.MkdirAll(filepath.Join(tempDirName, "data", "output"), storage.DefaultFilePerm)
	if err != nil {
		err = errors.Wrap(err, "failed to generate temp data input directory")
		return
	}
	err = os.MkdirAll(filepath.Join(tempDirName, "data", "predictions"), storage.DefaultFilePerm)
	if err != nil {
		err = errors.Wrap(err, "failed to generate temp data predictions directory")
		return
	}
	err = os.MkdirAll(filepath.Join(tempDirName, "memory"), storage.DefaultFilePerm)
	if err != nil {
		err = errors.Wrap(err, "failed to generate temp memory directory")
		return
	}
	if cleanup {
		defer os.RemoveAll(tempDirName)
	}

	// Generate sample names.
	sampleNames := make([]string, 10)
	for i := range sampleNames {
		sampleNames[i] = dataset.RandomString(10, "")
	}

	// Generate random input and output data.
	datasetIn, err := dataset.GenerateFromSchema("", schemaIn, sampleNames, 10)
	if err != nil {
		err = errors.Wrap(err, "failed to generate input dataset")
		return
	}
	err = datasetIn.Dump(filepath.Join(tempDirName, "data", "input"), dataset.DefaultOpener{})
	if err != nil {
		err = errors.Wrap(err, "failed to save input dataset")
		return
	}
	datasetOut, err := dataset.GenerateFromSchema("", schemaOut, sampleNames, 10)
	if err != nil {
		err = errors.Wrap(err, "failed to generate input dataset")
		return
	}
	err = datasetOut.Dump(filepath.Join(tempDirName, "data", "output"), dataset.DefaultOpener{})
	if err != nil {
		err = errors.Wrap(err, "failed to save output dataset")
		return
	}

	// Generate temporary config.
	var configStruct interface{}
	err = json.Unmarshal([]byte(configSpace), &configStruct)
	if err != nil {
		err = errors.Wrap(err, "failed to load config space")
		return
	}
	config, err := LoadConfig(configStruct)
	if err != nil {
		err = errors.Wrap(err, "failed to load config space")
		return
	}
	configSample := config.Sample()
	configJSON, err := json.Marshal(configSample.Dump())
	if err != nil {
		err = errors.Wrap(err, "failed to serialize config space")
		return
	}
	configFilePath := filepath.Join(tempDirName, "config.json")
	err = ioutil.WriteFile(configFilePath, []byte(configJSON), storage.DefaultFilePerm)
	if err != nil {
		err = errors.Wrap(err, "failed to write config file")
		return
	}

	// Call train.
	// Run the training.
	command := []string{
		"train",
		"--data", MntPrefix + filepath.Join(tempDirName, "data"),
		"--conf", MntPrefix + filepath.Join(tempDirName, "config.json"),
		"--output", MntPrefix + filepath.Join(tempDirName, "memory"),
	}
	outReader, err := RunContainerAndCollectOutput(modelImageName, nil, command)
	defer outReader.Close()
	if err != nil {
		err = errors.Wrap(err, "failed to run train")
		return
	}

	// Call predict.
	command = []string{
		"predict",
		"--data", MntPrefix + filepath.Join(tempDirName, "data"),
		"--memory", MntPrefix + filepath.Join(tempDirName, "memory"),
		"--output", MntPrefix + filepath.Join(tempDirName, "data", "predictions"),
	}
	outReader, err = RunContainerAndCollectOutput(modelImageName, nil, command)
	defer outReader.Close()
	if err != nil {
		err = errors.Wrap(err, "failed to run train")
		return
	}

	// Infer schema of output data.
	predDataset, err := dataset.Load(filepath.Join(tempDirName, "data", "predictions", "output"), true, dataset.DefaultOpener{})
	if err != nil {
		err = errors.Wrap(err, "failed to load output dataset")
		return
	}
	predSchema, err := predDataset.InferSchema()
	if err != nil {
		err = errors.Wrap(err, "failed to infer schema from output dataset")
		return
	}

	// Check if schemas match.
	match, _ := schemaOut.Match(predSchema, false)

	if match == false {
		err = errors.New("output dataset schema doesn't match the schema definition")
		return
	}

	return nil
}
