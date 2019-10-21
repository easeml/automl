---
title: "Client"
---

# client
--
    import "."


## Usage

```go
var ValidDatasetSources = []string{
	types.DatasetUpload,
	types.DatasetDownload,
	types.DatasetLocal,
}
```
ValidDatasetSources is a list of possible dataset sources.

```go
var ValidModuleSources = []string{
	types.ModuleUpload,
	types.ModuleDownload,
	types.ModuleLocal,
	types.ModuleRegistry,
}
```
ValidModuleSources is a list of possible module sources.

```go
var ValidModuleTypes = []string{
	types.ModuleModel,
	types.ModuleObjective,
	types.ModuleOptimizer,
}
```
ValidModuleTypes is a list of possible module types.

#### func  DatasetSourceAddressRequired

```go
func DatasetSourceAddressRequired(source string) bool
```
DatasetSourceAddressRequired returns true if the source address property is
required for a given source.

#### func  DatasetSourceValid

```go
func DatasetSourceValid(source string) bool
```
DatasetSourceValid checks if the provided dataset source is valid.

#### func  ModuleSourceAddressRequired

```go
func ModuleSourceAddressRequired(source string) bool
```
ModuleSourceAddressRequired returns true if the source address property is
required for a given source.

#### func  ModuleSourceValid

```go
func ModuleSourceValid(source string) bool
```
ModuleSourceValid checks if the provided module source is valid.

#### func  ModuleTypeValid

```go
func ModuleTypeValid(moduleType string) bool
```
ModuleTypeValid checks if the provided module type is valid.

#### type APIErrorResponse

```go
type APIErrorResponse struct {
	Code      int    `json:"code"`
	Error     string `json:"error"`
	RequestID string `json:"request-id"`
}
```

APIErrorResponse is a JSON object that is returned by the ease.ml API when an
error occurs.

#### func (*APIErrorResponse) String

```go
func (resp *APIErrorResponse) String() string
```

#### type APIKeyCredentials

```go
type APIKeyCredentials struct {
	APIKey string
}
```

APIKeyCredentials contains an API key which can be applied to a request.

#### func (APIKeyCredentials) Apply

```go
func (cred APIKeyCredentials) Apply(header http.Header)
```
Apply applies the given credentials to an HTTP request.

#### type BasicCredentials

```go
type BasicCredentials struct {
	Username string
	Password string
}
```

BasicCredentials represents a username and password pair which can be applied to
a request.

#### func (BasicCredentials) Apply

```go
func (cred BasicCredentials) Apply(header http.Header)
```
Apply applies the given credentials to an HTTP request.

#### type Context

```go
type Context struct {
	ServerAddress   string
	UserCredentials Credentials
}
```

Context contains all information needed to use the api functionality.

#### func (Context) CreateDataset

```go
func (context Context) CreateDataset(id, name, description, source, sourceAddress string) (string, error)
```
CreateDataset creates a new dataset given the provided parameters.

#### func (Context) CreateJob

```go
func (context Context) CreateJob(dataset, objective string, models []string, altObjectives []string, acceptNewModels bool, maxTasks uint64) (string, error)
```
CreateJob creates a new job given the provided parameters.

#### func (Context) CreateModule

```go
func (context Context) CreateModule(id, moduleType, label, name, description, source, sourceAddress string) (string, error)
```
CreateModule creates a new module given the provided parameters.

#### func (Context) CreateUser

```go
func (context Context) CreateUser(id, password, name string) (string, error)
```
CreateUser creates a new user given the provided parameters.

#### func (Context) GetDatasetByID

```go
func (context Context) GetDatasetByID(id string) (result *types.Dataset, err error)
```
GetDatasetByID returns a dataset given its ID.

#### func (Context) GetDatasets

```go
func (context Context) GetDatasets(status, source, schemaIn, schemaOut string) (result []types.Dataset, err error)
```
GetDatasets returns all datasets from the service.

#### func (Context) GetJobByID

```go
func (context Context) GetJobByID(id string) (result *types.Job, err error)
```
GetJobByID returns a job given its ID.

#### func (Context) GetJobs

```go
func (context Context) GetJobs(user, status, job, objective, modelName string) (result []types.Job, err error)
```
GetJobs returns all jobs from the service.

#### func (Context) GetModuleByID

```go
func (context Context) GetModuleByID(id string) (result *types.Module, err error)
```
GetModuleByID returns a module given its ID.

#### func (Context) GetModules

```go
func (context Context) GetModules(moduleType, user, status, source, schemaIn, schemaOut string) (result []types.Module, err error)
```
GetModules returns all modules from the service.

#### func (Context) GetMyID

```go
func (context Context) GetMyID() (result string, err error)
```
GetMyID returns the ID of the current user.

#### func (Context) GetProcesses

```go
func (context Context) GetProcesses(status string) (result []types.Process, err error)
```
GetProcesses returns all processes from the service.

#### func (Context) GetTaskByID

```go
func (context Context) GetTaskByID(id string) (result *types.Task, err error)
```
GetTaskByID returns a task given its ID.

#### func (Context) GetTasks

```go
func (context Context) GetTasks(job, user, status, stage, dataset, objective, modelName string) (result []types.Task, err error)
```
GetTasks returns all tasks from the service.

#### func (Context) GetUserByID

```go
func (context Context) GetUserByID(id string) (result *types.User, err error)
```
GetUserByID returns a user given its ID.

#### func (Context) GetUsers

```go
func (context Context) GetUsers(status string) (result []types.User, err error)
```
GetUsers returns all users from the service.

#### func (Context) Login

```go
func (context Context) Login(username, password string) (result string, err error)
```
Login takes a username and password and attempts to log the user in. If the
login was successful, the API key is returned which can be used to authenticate
the user.

#### func (Context) Logout

```go
func (context Context) Logout() error
```
Logout takes the provided user credentials and tries to log the user out.

#### func (Context) UpdateDataset

```go
func (context Context) UpdateDataset(id string, updates map[string]interface{}) (err error)
```
UpdateDataset applies the given updates to the dataset fields.

#### func (Context) UpdateModule

```go
func (context Context) UpdateModule(id string, updates map[string]interface{}) (err error)
```
UpdateModule applies the given updates to the module fields.

#### func (Context) UploadDataset

```go
func (context Context) UploadDataset(id, sourcePath string) error
```
UploadDataset uploads the dataset to the server.

#### func (Context) UploadModule

```go
func (context Context) UploadModule(id, sourcePath string) error
```
UploadModule uploads the module to the server.

#### type Credentials

```go
type Credentials interface {
	Apply(header http.Header)
}
```

Credentials represents a structure that is able to authenticate a user.
