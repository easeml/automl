# Ease.ml Design Document

This document contains all major design aspects of the ease.ml system. Its main purpose is to serve as a first point of contact for anyone who is starting to work with the code base. It does not aim to be complete, but rather a solid overview (the definitive source of documentation is the code itself).

## User Interface and User Experience

Main guidelines of UX design:

* [Convention over configuration](https://en.wikipedia.org/wiki/Convention_over_configuration) - prefer good default values and avoid expecting the user to make decisions as much as possible
* Frequent tasks need to be performed in the least number of steps possible (preferably 1)

### Command Line interface for running the services

Starting an ease.ml service is performed in the shell by typing:

```bash
$ easeml start <name_of_service> <arguments>
```

Name of service can be one of the following:

* `controller` - provides an external API to the user for controlling and monitoring the system
* `scheduler` - takes complex jobs and schedules particular tasks based on some optimization scheme
* `worker` - executes the model training and testing tasks, when started takes the first free GPU

If no name of service is provided, all three types of service are started on localhost (one worker service is started per GPU).

#### Database

All ease.ml services are stateless and don't communicate among themselves. Therefore they need a central database to provide state and coordination. As we use the ***MongoDB*** database system, having it installed is a prerequisite for running ease.ml. When started, each service looks for a running MongoDB instance and an appropriate database inside. In case the database is not there, an initialization script is automatically executed to create a blank database and populate it with necessary data structures.

Configuration arguments:

```bash
--database-instance <address_of_mongodb>
--database-name <name_of_target_database>
```

The default value for `database-instance` is `.` which means that the `easeml start` will start a new instance (unless it is already running) and store the database state in the working directory under `/shared/db`.

The default value for `database-name`  is `easeml`.

#### Working directory

All models, datasets and other types of files are kept in a central directory. It is possible to use any mounted Unix-like file system. The working directory path is configurable. The default is to use one local directory. Choices are `/var/lib/easeml` (which requires root access to set up during installation), or `~/.local/share/easeml` or `~/.easeml`. **[TO-DO: Figure this out]**

Configuration arguments:

```bash
--working-dir <path_to_working_directory>
```

### Notes

* When starting a new job, it would be useful to filter models by e.g. their inference latency (time needed to do inference given a schema)

### Developing and Deploying Models

Ease.ml runs external modules that can be models, objectives and optimizers. All of them are deployed as Docker images to encapsulate their dependencies and permit plug-and-play execution. They are accessed through a command line interface. (Potentially also REST API for serving predictions but also maybe training. **[TO-DO: Figure this out]**)

All files and executables need to be stored in the working directory of the Docker image.

#### Model

##### Schema

The model schema for inputs and outputs is stored in the `schema.in.json` and `schema.out.json` files respectively.

##### Getting a feasible set

The feasible set is stored in the `config-space.json` file (or alternatively `config-space.yml`).

##### Train the model

```bash
./train --data <training_data.hdf5> --config <model_config.json> --output <model_memory.hdf5> [--dbg <path_to_dbg_dir>]
```

##### Make predictions

```bash
./predict --data <test_data.hdf5> --memory <model_memory.hdf5> --output <predictions.hdf5> [--dbg <path_to_dbg_dir>]
```

#### Objective

##### Schema

The objective schema for inputs is stored in the `schema.in.json` file.

##### Compute value of objective

```bash
./eval --actual <actual_data.hdf5> --predicted <predicted_data.hdf5>
```



#### Optimizer

##### Generate a given number of tasks

```bash
./suggest --space <feasible_space.json> --history <history_data.json> --output <config.json> --num-tasks <int_number_of_tasks_to_generate>
```





### REST API

In this section we will cover the outline of the REST API. The detailed description based on the OpenAPI specification will be placed in a separate file. A REST API is generally centered around resources and actions that we can perform on them given as HTTP verbs (e.g. GET, PUT, POST, etc.) Here we will list those resources along with verbs that will be made available.

The API contains several types of resources that are more or less directly mapped to collections in MongoDB. Here we explain the main access pattern to all those resources. They are: `users`, `processes`, `datasets`, `modules`, `jobs` and `tasks`. Calling `GET` on any of these resources gives access the whole set (with cursor-based pagination supported). The collection can also be filtered based on some parameters. It is possible to specify a `sort-by` parameter to specify the field which we want to use for sorting. (**TO-DO:** It will also be possible to specify a `project` parameter which will cause the response to contain only specified fields in order to minimize data transfer.)

Calling `POST` on the collection is used to create new items when possible where the request body contains the properties of the item. Items of type `dataset` and `module` are specific because they can involve a file upload, in which case the `POST` response contains an upload link to which the API user can upload the content (**TO-DO:** Rewrite this as it is not accurate).

Each item in the collection has a unique string identifier which can be appended to the resource name after a forward slash to access that exact item (e.g. `users/alex` or `datasets/master/cifar10`). Notice that the dataset's identifier is formatted as `owner`/`id`. Jobs and tasks have GUID-like identifiers where the task has a `job-id/task-id` format. Calling `PATCH` on an item is used to change any of its properties.

Now we list all the resources:

* `users`
  * `GET` filter parameters: `status`
  * Other verbs: `POST`, `PATCH`
* `processes`
  - `GET` filter parameters: `status`, `type` (either `controller`, `scheduler` or `worker`)
  - Other verbs: n/a
* `datasets`
  - `GET` filter parameters: `user`, `status`, `source` (either `local`, `remote` or `upload`) or `schema` in the request body
    - `sort-by`: `creation-time`
  - Other verbs: `POST`, `PATCH`
  - Upload and download links are `datasets/{owner}/{id}/upload` and `datasets/{owner}/{id}/download`
* `modules`
  - `GET` filter parameters: `user`, `status`, `type` (either `model`, `objective` or `optimizer`) `source` (either `hub`, `local`, `remote` or `upload`) or `schema` in the request body
    - `sort-by`: `creation-time`
  - Other verbs: `POST`, `PATCH`
  - Upload and download links are `modules/{owner}/{id}/upload` and `modules/{owner}/{id}/download`
* `jobs`
  * `GET` filter parameters: `user`, `dataset`, `process`, `model`, `objective`,  `status`,  or `schema` in the request body
    * `sort-by`: `creation-time`, `running-time`
  * Other verbs: `POST`, `PATCH`
* `tasks`
  * `GET` filter parameters: `job`, `user`, `dataset`, `process`, `model`, `objective`,  `status`,  or `schema` in the request body
    * `sort-by`: `quality`, `creation-time`, `running-time`
  * Other verbs: n/a
  * Download link for the trained model as a Docker image is `tasks/{job-id}/{task-id}/download`
  * Replacing `{task-id}` with `best` returns the task with the highest score.

**TO-DO**: Describe how training logs and debug data is accessed through the REST API.

### Web UI



## Data Design

This section describes all data structures used in the system, regardless if they are stored as files or as records in a database.

* Container Registry
* MongoDB vs Cassandra



### Schema

A schema is a formal definition of the structure of data. It is used to specify the *type* of data regardless of the actual content. Every valid data set conforms to a schema. Every model (as well as other data processing modules) defines an input and output schema which indicates what kinds of data sets it is able to read and produce. We use input/output schema pairs to determine if a model is applicable to a data set.

**Schema design goals.** A schema design must fulfill the following conditions:

1. Only one way to represent one type of data structure (isomorphic data structures have the same schema)
2. Uniquely define layout of data in the data set
3. Balance between being expressive for more complex data types and simple for simple ones

The schema contains the following elements:

* **base types** - Represent simple data types. Currently we support:

  * `Float[dim1, dim2,...]` - N-dimensional floating-point tensors. A dimension can be either a number (denoting that a dimension has a fixed size) or an alphanumeric string denoting a variable size. Repeating the same variable dimension name enforces those dimensions to be of the same size. It is also permitted to specify `*` as a dimension, this matches it to any size, and doesn't enforce any equality constraint between dimensions. Dimensions can have quantifiers to denote a certain dimension count. For example `Float[A[2]]` is equivalent to writing `Float[A, A]` (note that here we enforce square matrices, to permit any matrix size we would write `Float[*[2]]`).
  * `Categorical[dim]` - Categorical one-hot encoded vector. Under the hood it is represented as a floating point vector. The elements of this vector can contain any real value (e.g. parameters of a multinomial distribution). The main difference between a `Float[dim]` vector is in the semantics of its content &mdash; a float is by default not interpretable as categorical.
  * `Word` - Single word (variable-length) character string.

* **sets** - Represent unordered collections of elements, where an element can be represented by one or more base types. Each element also has the ability to contain pointers to elements in the same set or in other sets. A set is defined as:

  `setlabel : { basetype1, basetype2, ... }{ setlabel1, setlabel2, ... }`

  A reference to elements in the same set is represented by a `this` label. Labels can have quantifiers to denote a certain label count. For example `this[2]` is the same as writing `this, this`. We also support variable label counts. For example `this[:2]` denotes 0, 1 or 2 references, and `this[:]` denotes any number of references. To avoid ambiguity, a set of references can have at most one variable-count reference and references must appear in the same order as the sets.

* **structures** - A structure contains one or more sets with certain constraints on their references. This allows us to enforce a specific structure of set elements (e.g. a tree, a graph etc). A set cannot exist outside of a structure. The possible constraints are:

  * **Cyclic/Acyclic** - Determines whether we permit references to form cycles.
  * **Directed/Undirected** - An *undirected* structure enforces the following constraint on all pointers: if A points to B, then B must point to A.
  * **Single incoming pointer / Multiple incoming pointers** (candidate single-word names: Incident/Fusing) - Determines if we permit each element to have only one incoming pointer or do we permit multiple. This can make a difference between a tree and a directed acyclic graph.

  A structure is written as:

  `structure { set1, set2, ... } as constraint1 constraint2 ...`

* **type** - Root of the type hierarchy. Contains zero or more structures followed by zero or more base types. Written as `{ structure1, structure2, ..., basetype1, basetype2, ... }`.

A simplified version of the schema syntax is given as follows:

```
type := "{" list of (struct|basetype) "}"
struct := "structure {" list of set "}" ["as" (list of modifier)]
modifier := "CYCLIC" | "FUSING" | "UNDIRECTED"
set := label ":" "{" list of basetype "}{" list of reference "}"
reference := label | label "[" [INT] ":" [INT] "]"
basetype := Float ["[" list of dim "]"] | Categorical ["[" dim "]"] | Word
dim := WORD | INT | (WORD|"*") "[" [INT] ":" [INT] "]"
```

We can notice that the schema (as described so far) permits multiple ways to represent the same data. For example if we want our model to take a square image and a feature vector we could represent it in two ways: `{Float[A[2]], Float[B]}` or `{, Float[B], Float[A[2]]}`. The same applies for ordering of all other type elements. To resolve this, we enforce a strict ordering scheme:

* In a type, structures must come before base data types
* A structure with more sets must come before a structure with less sets
* A set with more fields comes before a set with less fields
* A base type with more dimensions must come before a base type with less dimensions
* `Float` comes before `Category`, which comes before `Word`

In each of the aforementioned points, if two elements are of the same "priority", we resolve their priority by looking at points below. For example a set with `Float[A, B], Float[C, D]` comes before a set with `Float[A, B], Float[C]`.

Here is a (non-exhaustive) list of example data types and their corresponding schema representations:

* Feature vector:
```
{
   Float[A]
}
```
* Square image:
```
{
   Float[A[2]]
}
```
* 3D Tensor:
```
{
   Float[A[3]]
}
```
* Variable-length sequence of feature vectors:
```
{
   structure { elem : { Float[A] }{ this } }
}
```
* Variable-length sequence of images:
```
{
   structure { elem : { Float[A[2]] }{ this } }
}
```
* Variable-length sequence of words:
```
{
   structure { elem : { Word }{ this } }
}
```
* Binary tree of words:
```
{
   structure { elem : { Word }{ this[:2] } }
}
```
* Directed acyclic graph of words:
```
{
   structure { elem : { Word }{ this[:] } } as FUSING
}
```
* Unrooted tree of feature vectors:
```
{
   structure { elem : { Float[A] }{ this[:] } } as UNDIRECTED FUSING
}
```
* Undirected graph where nodes are associated with feature vectors:
```
{
   structure { node : { Float[A] }{ this[:] } } as CYCLIC UNDIRECTED FUSING
}
```
* Undirected graph where *edges* are associated with feature vectors:
```
{
   structure {
      node : { }{ edge[:] }
      edge : { Float[A] }{ node[2] }
   } as CYCLIC UNDIRECTED FUSING
}
```

The schema syntax described above is used by ease.ml to enable users to write and see the schema as compactly as possible. However, it is not reasonable to expected for someone who implements a model module or an objective function module to also implement a schema parser. That is why, the schema is given in a JSON/YAML format to these models. The rough *schema* of a JSON schema descriptor is given as follows:

```json
type := [ ( structure | basetype ), ... ]
structure := { 'constraints' : [constraint, ...], 'sets' : [set, ...] }
constraint := 'cyclic' | 'undirected' | 'fusing'
set := { 'label' : STR, 'fields' : [basetype, ...], 'refs' : [ref, ...] }
ref/dim := { 'label' : STR, 'count' : ( INT | { 'from' : INT, 'to' : INT } ) }
basetype := { 'type' : STR, 'dims' : [dim, ...] }
```







```json
type := { 'struct-#': structure, 'type-#': basetype }
structure := { 'constraints' : [constraint, ...], 'sets' : [set, ...] }
constraint := 'cyclic' | 'undirected' | 'fusing'
set := { 'label' : STR, 'fields' : [basetype, ...], 'refs' : [ref, ...] }
ref/dim := { 'label' : STR, 'count' : ( INT | { 'from' : INT, 'to' : INT } ) }
basetype := { 'type' : STR, 'dims' : [dim, ...] }
```



```json
{
    'set-0' : {
        'tensor-0' : [{ 'A' : 1 }, { 'B' : [0, 'inf'] }],
        'category-0' : 'category-class-1',
        'refs' : [{'set-0' : 1}, {'set-0' : [0, 'inf']}]
    },
    'tensor-0' : [{ 'A' : 1 }, { 'B' : [0, 'inf'] }],
    'category-0' : 'category-class-1',
    'ref-constraints' : {
        'cyclic' : false,
        'undirected' : false,
		'fan-in' : false,
    }
}
```



```json
{
    'fields' : [
        {
            'type' : 'set',
            'label' : 'graph',
            'fields' : [
                {
                    'type' : 'tensor',
                    'label': 't1',
                    'dim' : [
                        { 'var': 'A', 'count' : 1 },
                        { 'var': 'B', 'count' : [0, 'inf'] }
                    ],
                },
                {
                    'type' : 'category',
                    'label': 'c1',
                    'dim' : 128,
                }
            ],
            'refs' : [{'graph' : 1}, {'graph' : [0, 'inf']}]
        },
        {
            'type' : 'tensor',
            'label': 't1',
            'dim' : [{ 'A' : 1 }, { 'B' : [0, 'inf'] }],
        },
        {
            'type' : 'category',
            'label': 'c1',
            'dim' : 128,
        },
    ],
    'ref-constraints' : {
        'cyclic' : false,
        'undirected' : false,
        'fan-in' : false,
    }
}
```





```yaml
fields:
  - type: 'set'
    name: 'graph'
    fields:
      - type: 'tensor'
        name: 't1'
        dim: ['A+', 'B']
      - type: 'category'
        name: 'c1'
        class: 'words'
    links:
      graph: [1, 'inf']
  - type: 'tensor'
    name: 't1'
    dim: ['A?', 'B']
  - type: 'category'
    name: 'c1'
    class: 'words'
ref-constraints:
  cyclic: false
  undirected: false
  fan-in: false
classes:
  words:
  	dim: 128
```



```yaml
fields:
  - type: 'set'
    label: 'graph'
    fields:
      - type: 'tensor'
        label: 't1'
        dim: ['A', 'A']
    links:
      graph: [0, 'inf']
ref-constraints:
  cyclic: true
  undirected: true
  fan-in: true
```







```
/input
	/words.class.txt
	/[id]
		/graph
			/[id]
				/t1.ten.hdf5
				/c1.cat.txt
		/links.json	
		/t1.ten.hdf5
		/c1.cat.txt

```



```
/input
	/words.txt
	/[id]
		/graph
			/t1.hdf5
			/c1.txt
		/links.csv	
		/t1.hdf5
		/c1.txt
```





```json
{
    'tensor-0' : [{'A' : 1}, { 'B' : [0, 'inf'] }]
}
```





It should be noted that even though this JSON structure seems to be complex, it remains simple for simpler models, so a simpler model can have an easier time reading the JSON schema specification. For example, a model that accepts images should be able to interpret this:

```json
[{
    'type' : 'float',
    'dims' : [
        { 'label' : 256, 'count' : 2 },
        { 'label' : 3, 'count' : 1 }
    ]
}]
```



### Dataset Structure

**TO-DO:** Two possibilities: dataset stored in an HDF5 file or dataset stored in a regular directory structure where leaves are HDF5 files. There has to be a single way to read/write files across modules (model, objective, preprocessing unit etc). Whichever choice is made, the data structure will be hierarchical.

The hierarchy of datasets depends on the schema. There needs to be a 1-1 mapping from file structure and file formats to the schema representation. We take a minimal approach where the file structure is kept as simple as possible.

The hierarchy levels are modeled after the hierarchical structure of the schema. Some levels have a fixed number of nodes (e.g. there is a fixed number of sets, a set element has a fixed number of fields, etc), while others have a variable number of nodes (e.g. a data set has a variable number of samples, a set has a variable number of elements). We achieve minimalism by collapsing any fixed node level if it has only one child node (e.g. if there is only one base type per set element instead of multiple). Here are two significant hierarchy paths:

```
                    dataset
                       |
                 {sample, ...}
                     /   \
      [structure, ...]   [base type, ...]
             |
        [set, ...]
             |
   {set element, ...}                      Legend:
             |                            [] - list (index -> element)
     [base type, ...]                     {} - set (key -> element)
```

Curly braces `{}` represent variable number nodes that are stored as mappings from some unique `id` to the actual file structure. Square braces `[]` represent fixed number nodes that are stored as mapping from their zero-based integer ordinal to the actual file structure. The collapsing is performed by taking a fixed number node that has no siblings (e.g. if a structure has only one set) and passing all pointers to its children to its parent (e.g. the structure will point to all elements of the set). Every structure has a pointer file (stored as `refs.csv` as the child of the `structure` node) which has two string columns (from and to) and each row is a pointer.

As a schema is uniquely mappable to a file structure it is possible to do automated schema inference given a correct file structure. This is achieved recursively by going deeper into the hierarchy and returning the schema hypothesis and then testing/correcting it as we traverse the hierarchy.

It is often the case that users' data sets don't conform to the given file structure. A user may have a list of directories named by output class and each directory contains images that belong to that class. Those images might be stored as JPEG files. We need to facilitate these kinds of scenarios. **TO-DO**



### Model configuration

Models are functions that map their inputs to outputs given a set of *differentiable* parameters (which we call *memory*) and *non-differentiable* hyperparameters (which we call *configuration*). The differentiable parameters are obtained through training by learning algorithms (such as gradient descent). Non-differentiable hyperparameters are specified before training and optimized over a multitude of training runs through a process called *hyperparameter optimization*. This involves using a black-box function optimization method (such as Bayesian optimization) to search for optimal hyperparameter values over a predefined *feasible set*.

The feasible set is defined by a JSON/YAML dictionary where keys are parameters and values are domains represented as nested dictionaries. Properties of a domain are prefixed with a period `.` sign. Here is a list of available domains:

* **Integer** - Defined when the domain dictionary has a `.int` key and a value that is a 2-element list representing the lower and upper bounds (inclusive) of the integer range. It is possible to add a `.scale` key to the dictionary with a value from `linear` (default) and `exp`.
* **Real** - Defined when the domain dictionary has a `.float` key and a value that is a 2-element list representing the lower and upper bounds of the real range. It is possible to add a `.scale` key to the dictionary with a value from `linear` (default), `log` and `exp`.
* **Choice** - Defined when the domain dictionary has a `.choice` key and a value that is a list with at least two elements. The optimizer will choose one of the values from the list and assign them to the parent key of the domain dictionary.
* **Constant** - This is a special domain because it is ignored by the optimizer.

Here is an example of a feasible set definition written in YAML:

```yaml
number_of_layers:
  .int: [1,5]
learning_rate:
  .float: [-4, -1]
  .scale: exp
units_per_layer:
  .choice: [128, 256, 512]
```

At runtime, the optimizer traverses this feasible set definition and whenever it encounters a dictionary with `.int`, `float` or `.choice` keys it collapses it based on the domain type. Here is an example of a configuration passed to a model written in JSON:

```json
{
    "number_of_layers" : 2,
    "learning_rate" : 0.0025,
    "units_per_layer" : 512
}
```

It should be noted that the `.choice` domain is more powerful because it not only permits a choice of a numeric and string value but also a dictionary. This dictionary can have sub-parameters with their own domain descriptors, thus forming a hierarchical choice-based feasible set. Here is a simplified example:

```yaml
optimizer:
  .choice:
    - type: sgd
      learning_rate:
        .float: [-4,-1]
        .scale: exp
      momentum:
        .float: [0, 0.5]
    - type: rmsprop
      learning_rate:
        .float: [-5, -2]
        .scale: exp
      rho:
        .float: [0, 0.3]
```

Notice that the choice of optimizer entails a choice of a different set of parameters. During optimization, after the optimizer choice has been made, the domain dictionary is simple replaced with the nested dictionary representing the choice. Then all nested parameters are replaced with their values. Notice that the `type` sub-parameter is an example of a constant domain. Here is an example of one resulting configuration:

```json
{
    "optimizer" : {
        "type" : "sgd",
        "learning_rate" : 0.001,
        "momentum" : 0.32
    }
}
```

We can see that the choice between models is easily represented with this framework as the choice of model is simply a `.choice` domain type where values are sub-dictionaries with individual model configurations.



### Data Directory Structure

Each ease.ml process has access to a working directory. In case we want to run in a distributed setup, the working directory should be mounted to a shared file system.

The storage is divided into:

* **Shared storage** - contains all docker images and data sets
* **Individual storage** - each process storing logs, debug files, etc

**TO-DO**: Decide if both storages should be on the same file system or we should split them by placing individual storage on a low latency local file system. ***Solution:*** Prefer same system for now, maybe change mind later.

The directory structure is the following:

```
/shared
    /images
        /model
            /[user-id]
                /[module-id]
        /objective
            /[user-id]
                /[module-id]
        /scheduler
            /[user-id]
                /[module-id]
    /data
        /stable
            /[user-id]
                /[dataset-id]
        /temporary
            /[job-id]
                /[task-id]
    /jobs
        /[job-id]
            /[task-id]
            	/parameters
            	/logs
            		train.log
            		predict.log
            		eval.log
            	/debug
            		/train
            		/predict
            	/predictions
            	/evaluation
            		/[module-id].csv
    
    /scheduling
    	/input
    		/config
    			/[job-id].json
    		/history
    			/[job-id]
    				/[task-id].json

/individual
    /[host-id]-[process-id]
        /[start-time].log
        /[job-id]
            /[task-id]
            
```

The `shared` directory holds all files that are common to the whole system. These include Docker images which we divide into `model`, `objective` and `scheduler`. Data is kept in this directory as well. The `stable` subdirectory contains primarily user-uploaded data which is meant to never get deleted. The `parameters` subdirectory contains model parameters obtained after training. The `temporary` subdirectory contains prediction outputs of trained models. Files in this subdirectory can have a maximum lifetime after which they get deleted. Each file stored in these directories has a unique ID filename. The meta-data for all files is stored in the database (**TO-DO:** Rewrite this as it is not accurate).

The `individual` directory is intended for direct storage for modules (models, objectives etc).

### Database

The database is the central repository which acts as the second pillar of the complete state of the system. We use the MongoDB database because of its ease of use and scalability capabilities. In case performance becomes an issue in the future, we may chose to explore other options such as Redis or Cassandra, or it might even not be suitable to use a No-SQL option.

We describe the data model in terms of all MongoDB collections and the schema of the documents they store:

* `users` - All users of the system. Users can have private models and data sets that are not visible to everyone. There is one predefined user: `master`.
  * `name` - User's name (must be unique).
  * `password` - SHA1 hash of the user's password.
  * `status` - User's status.
    * Possible values: `active`, `archived`
  * `access-token` - When a user logs in, they are assigned an access token which they can use to interact with the REST API. The `master` user doesn't have a password but has an access token which is written in the console each time a `control` service is started.
* `processes` - All processes that were ever running in the system.
  * `id` - Identifier.
  * `process-id`
  * `host-id`
  * `host-address`
  * `start-time`
  * `type` - Type of the process.
    * Possible values: `controller`, `worker`, `scheduler`
  * `resource` - One of: `cpu` or `gpu`.
  * `status` - Status of the process
    * Possible values: `idle`, `working`, `terminated`
* `datasets` - All (stable?) datasets.
  * `id` - Identifier. Should be a human readable string.
  * `user` - Owner of the data set and the only one who can see it. If `null` then everyone can see it.
  * `path` - Path to dataset in the directory structure.
  * `name` - Name of the dataset.
  * `description` - Description of the dataset written in Markdown.
  * `schema` - Dictionary containing `input` and `output` schema descriptors.
  * `source` - If `null` then the source is a HTTP file upload. Otherwise it must be a dictionary with one of the following formats:
    * `{"local" : "<path to file on a mounted file system>"}`
    * `{"remote" : "<remote file URL prefixed with http or ftp"}`
  * `creation-time` - Time when the dataset was uploaded.
  * `status` - Status of the data set.
    * Possible values:
      * `created` - Dataset registered with schema description.
      * `transferred` - Dataset transfer completed. Schema may not match the dataset.
      * `validated` - After we check that the schema matches the content.
      * `archived` - We cannot use it in future jobs.
* `modules` - All modules that are kept as docker instances (including models and objectives?)
  * `id` - Identifier. Should be a human readable string.
  * `name` - Human readable name of the model.
  * `description` - Description of the module written in Markdown.
  * `type` - Type of module.
    - Possible values:
      - `model` - Machine learning model.
      - `objective` - Objective function that measures the quality of predictions.
      - `optimizer` - Black box optimizer that is plugged into the scheduler.
  * `status` - Status of the module.
    - Possible values:
      - `created` - Module registered but not transferred.
      - `active` - Module image transfer completed. Ready for usage.
      - `archived` - We cannot use it in future jobs.
  * `schema` - Schema definition (in the JSON representation). Can be `null` for `optimizer` modules.
    * Keys: `input` and `output`
  * `image` - Identifier of the Docker image which contains the module.
  * `source` - If `null` then the source is a HTTP file upload. Otherwise it must be a dictionary with one of the following formats:
    * `{"hub" : "<image identifier on a docker registry>"}`
    * `{"local" : "<path to file on a mounted file system>"}`
    * `{"remote" : "<remote file URL prefixed with http or ftp"}`
  * `user` - Owner of the module and the only one who can see it. If `null` then everyone can see it.
* `jobs` - Collection of jobs that can be picked up by schedulers.
  * `id` - Identifier. Unique random character string.
  * `dataset` - Id of the dataset used for training/evaluation.
  * `process` - Identifier of the scheduler process that is handling the job.
  * `models` - List of id's of all applicable models.
  * `accept-new-models` - Boolean. If set to `true` (default) then whenever a new models is added, if it is applicable to the dataset it will be automatically added to the `models` list.
  * `objective` - Objective to use to use when evaluating models.
  * `alt-objectives` - Additional objectives to run on the models' predictions. These don't impact the optimization. **TO-DO**: Consider enabling adding new alt objectives for tasks that have been completed.
  * `start-time` - Start time of the task. Used to estimate running time.
  * `status` - Status of the job.
    * Possible values: `scheduled`, `running`, `paused`, `completed`, `terminating`, `terminated`
  * `user` - User that submitted the job.
* `tasks` - Collection of tasks that can be picked up by workers.
  * `id` - Identifier.
  * `job` - Identifier of the job that this task was spawned from.
  * `process` - Identifier of the process that is handling the task.
  * `model` - Identifier of the target model.
  * `objective` - Identifier of the objective to apply (here for easier searching).
  * `dataset` - Identifier of the dataset (here for easier searching).
  * `user` - Identifier of the user who started this task's job (here for easier searching).
  * `config` - Configuration that is used to initialize the model.
  * `memory-file` - Path to the file that contains the memory of the task. Available after the `training` stage is finished.
  * `prediction-dataset` - Path to the file that contains the predictions that this model made. Available after the `predicting` stage is finished.
  * `quality` - Value of the quality metric of the trained model obtained through the objective function. Available after the `evaluating` stage is finished.
  * `quality-alt` - Quality metric values of additional objectives (if defined in the `job`). These don't impact the optimization.
  * `status` - Status of the task.
    * Possible values: `scheduled`, `running`, `paused`, `completed`, `terminated`
  * `stage` - Current execution stage. Invalid if the status is not `running` or `paused`.
    * Possible values: `training`, `predicting`, `evaluating`
  * `start-time` - Start time of the task. Used to estimate running time.
* `config` - Contains a single configuration JSON document (**TO-DO:** Maybe this will be removed. We maybe won't store configurations in the database but rather in a file.)



### Configuring ease.ml

We define a set of properties that are used to configure the functionality of ease.ml services. These properties can be stored in configuration files, in environment variables or specified directly (or even maybe stored in the centralized database). Configuration files must be named either `config.yml` or `config.json`. There is a strict list of precedence of sources, going from lowest to highest priority:

1. Global configuration stored in the `~/.easeml` directory
2. Configuration file stored in the same directory as the `easeml` binary
3. Configuration file stored in the working directory
4. Environment variables
5. Configuration file that is specified by a command line argument `--config`
6. Explicitly specified command line flags





## System Operation

### Manipulating users

Users can be added, or archived. All actions are performed by the `controller` process through the REST API. Consider restricting the right to perform these actions to the `master` user.

### Manipulating processes

Processes can be started or terminated with the `easeml start` and `easeml stop` commands. Processes then update the `processes` collection of the database. Their status can be accessed through the `controller` process by using the REST API (currently read-only, a "stop process" signal might be useful, but that is trivial to add later).

### Manipulating datasets

There are two steps to submitting a dataset to ease.ml: (1) schema inference; and (2) data & schema upload. If the schema is known, then step (1) can be skipped.

***Schema inference*** is done on the client side by scanning the HDF5 file. This can be done in the command line or in the browser, which means the functionality needs to be implemented both in JavaScript and Go. If the schema inference is successful, the schema files (`input.schema` and `output.schema`) are generated.

***Data and schema upload*** is performed through the REST API of the `controller`. The client (browser and command line) simple send the HDF5 file and the schema file(s) to the controller. It is performed in two steps: (1) first a dataset object is created and the schema is passed in the request body, the response contains the upload link; (2) the upload link is used to upload the dataset to the server. The controller does the final schema verification. If it passes, then the file gets stored in the stable data directory and a document is added to the `datasets` collection. **TO-DO:** The schema, dataset description etc should be included in the HDF5 file?

**TO-DO:** There is a possible step before the schema inference. Some users have datasets in different formats (collection of images, CSV files, text files, audio files, etc). We should facilitate the process of packing and unpacking these datasets to/from an HDF5 file.

A dataset can have its fields edited (e.g. name, description) and the dataset can be archived which means it can't be used in future experiments.

### Manipulating Models and Objectives

All modules (models, objectives, optimizers, and possibly some additional future types) are packed as Docker images with the interface described in [Developing and Deploying Models](#developing-and-deploying-models).

Before a new image can be added to the ease.ml system, it has to be pushed to a registry (e.g. Docker Hub) which is accessible for all ease.ml processes. When we create a module, we specify the Docker image identifier which is appended to the `docker pull` command. The identifier is of the format `[REGISTY/]NAME[:TAG|@DIGEST]`. After a module is created, an image is immediately pulled and saved in the scratch directory so that it can be used by other processes without downloading again. Layers of the image may remain in the cache of the controller which should speed up future downloads.

Finally, in case we are dealing with a model, the controller goes through all datasets to which the model can be applied. It then finds all the active jobs that have `accept-new-models` set to `true` and are running on those datasets in order to add the model to those jobs.

An image can be edited (e.g. changing the name) and archived which means it cannot be used for future jobs.

**TO-DO:** It may be useful to add image versions (enables updating the image while still logically keeping the same image identity) and benchmarking or some other performance metrics (enables time/cost estimates).

### Running jobs

This is the central task of the system. It is performed by multiple entities.

Firstly, a job is created through the REST API of the `controller` by specifying the dataset, objective and (optionally) a list of models. If a list of models is omitted, the controller will do pattern matching and find all applicable models. Otherwise, it will validate that the job submission is correct and add it to the `jobs` collection.

The new job is picked up by the `scheduler` which is periodically listening for new jobs that are associated with less than `N` tasks (parameter specified in the configuration). It unpacks it, performs the optimization procedure and produces tasks which are added to the `tasks` collection. The field `config` contains the unique hyperparameter configuration. The fields `process`, `memory-file`, `prediction-dataset` and `quality` are left empty as they will be updated by the `worker` process.

Workers that are not occupied listen for new tasks. When they find one, they set their process id to the task and start handling it. They run the train-predict-evaluate cycle and update the `stage`, `memory-file`, `prediction-dataset` and `quality` fields accordingly. After the task is handled, the `status` is set to `completed` and the next task is handled if available. The record of the task remains in the `tasks` collection and can be analyzed through the controller.