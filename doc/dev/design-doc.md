# Ease.ml Design Document

This document contains all major design aspects of the ease.ml system. The main purpose of is to serve as a first point of contact for anyone who is starting to work with the code base. We assume here that the reader is familiar with the overall concept behind ease.ml and aims to explain how components of the system work and how they interact with each other. This document does not aim to be complete with all implementation details, but rather a solid (more functional) overview. The definitive source of documentation is the code itself.

## Interfaces

This section describes the different interfaces of ease.ml - most notably the command line interface, the REST API and the interface to Docker images.

Main guidelines of interface design from a UX perspective:

* [Convention over configuration](https://en.wikipedia.org/wiki/Convention_over_configuration) - prefer good default values and avoid expecting the user to make decisions as much as possible
* Frequent tasks need to be performed in the least number of steps possible (preferably 1)

### Command Line interface for running the services

Starting an ease.ml service is performed in the shell by typing:

```bash
$ easeml start <process_types> <arguments>
```

Process type can be one of the following:

* `controller` - provides an external API to the user for controlling and monitoring the system; currently we support only one instance running
* `scheduler` - takes jobs and schedules tasks for workers based on some optimization scheme; currently we support only one instance running
* `worker` - executes the model training and testing tasks; one instance should correspond to one computational resource such as CPU/GPU

If no process type is provided, one of each type of process gets started on localhost.

NOTE: Even though we call them processes here, we mean it on a logical level. In reality each "process type" actually gets started as a thread. Running them as separate processes requires typing `easeml start <process_type>` multiple times, each time with a different process type.

### Data Storage

All ease.ml services are stateless and don't communicate among themselves. Therefore they rely on a central storage system. We use two complementary systems: (1) a database system for smaller scale meta-data; (2) a file system for larger scale storage. The overall guideline is that all data that is transient and rapidly changing should be stored in the database, while data that is more permanent should be stored in the file system.

#### Database

We currently use the ***MongoDB*** database system (this will most probably change in the future as we plan to switch to a relational database system). Having it installed is a prerequisite for running ease.ml. When started, each service looks for a running MongoDB instance and an appropriate database inside. In case the database is not there, an initialization script is automatically executed to create a blank database and populate it with necessary data structures.

Configuration arguments:

```bash
--database-instance <address_of_mongodb>
--database-name <name_of_target_database>
```

The default value for `database-instance` is `localhost` which assumes that a database instance is running on the local host.

The default value for `database-name`  is `easeml`. It will be created if it doesn't exist.

#### Working directory

All models, datasets and other types of files are kept in a central directory. It is possible to use any mounted Unix-like file system. The working directory path is configurable. The default is to use `~/.easeml` as it can be easily created when ease.ml is first started.

Configuration arguments:

```bash
--working-dir <path_to_working_directory>
```

### Notes

* When starting a new job, it would be useful to filter models by e.g. their inference latency (time needed to do inference given a schema)

### Developing and Deploying Models

Ease.ml runs independently developed modules that can be models, objectives and optimizers. All of them are deployed as Docker images to encapsulate their dependencies and permit plug-and-play execution. Ease.ml interacts with them solely through a command line interface (we might switch to a socket-based interface in the future).

To run modules, ease.ml calls specifically defined commands in the command line and passes the relevant file/directory paths. These correspond to nodes in the working directory tree that get mounted to the file system of the Docker container.

In the following sections we describe each module type and the interface it must provide to the system.

#### Model

Models correspond to machine learning models that can be trained with training data and after that be used to make predictions given test data. The input and output data types are defined by a schema. Models can also define a set of hyperparameters which are used to configure them.

##### Schema

The model schema for inputs and outputs is stored in the `schema.in.json` and `schema.out.json` files respectively.

##### Hyperparameters

The feasible set of hyperparameters is stored in the `config-space.json` file (or alternatively `config-space.yml`). During training, ease.ml provides a set of hyperparameters to the model that are instantiated based on the feasible set that the model defined.

##### Train the model

```bash
./train --data <training_data_dir> --config <model_config.json> --output <model_memory_dir> [--dbg <path_to_dbg_dir>]
```

##### Make predictions

```bash
./predict --data <test_data.hdf5> --memory <model_memory.hdf5> --output <predictions.hdf5> [--dbg <path_to_dbg_dir>]
```

#### Objective

Objectives are used to evaluate the quality of model predictions. They take actual output from the dataset and model prediction output and print the evaluation results in the standard output.

##### Schema

The objective schema for inputs is stored in the `schema.in.json` file.

##### Compute value of objective

```bash
./eval --actual <actual_data.hdf5> --predicted <predicted_data.hdf5>
```

The last line of the objective standard output must be a single floating point number between 0 and 1.

If per-sample score is available, it can be printed in previous lines (one line per data sample) with the following format: `<sample_id>|<score>`.

#### Optimizer

##### Generate a given number of tasks

```bash
./suggest --space <feasible_space.json> --history <history_data.json> --output <config.json> --num-tasks <int_number_of_tasks_to_generate>
```



### REST API

In this section we will cover the outline of the REST API. The detailed formal description based on the OpenAPI specification will be placed in a separate file (`rest-api.yml`) and can be viewed [here](http://petstore.swagger.io/?url=https://raw.githubusercontent.com/DS3Lab/easeml/master/doc/dev/rest-api.yml).

A REST API is generally centered around resources and actions that we can perform actions on them given as HTTP verbs (e.g. `GET`, `PUT`, `POST`, etc.) Here we simply list general groups of resources and general usage patterns. For a detailed description, consult the OpenAPI specification.

#### Resources

The API contains several types of resources that are more or less directly mapped to collections in MongoDB. Here we explain the main access pattern to all those resources. They are: `users`, `processes`, `datasets`, `modules`, `jobs` and `tasks`. Calling `GET` on any of these resources gives access the whole set (with cursor-based pagination supported). The collection can also be filtered based on some parameters. It is possible to specify a `sort-by` parameter to specify the field which we want to use for sorting and an `order` parameter to specify either ascending (`asc`) or descending (`desc`) order.

Calling `POST` on the collection is used to create new items when possible where the request body contains the properties of the item. Items of type `dataset` and `module` are specific because they can involve a file upload, in which case the `POST` response contains an upload link to which the API user can upload the content (**TO-DO:** Rewrite this as it is not accurate).

Each item in the collection has a unique string identifier which can be appended to the resource name after a forward slash to access that exact item (e.g. `users/alex` or `datasets/root/cifar10`). Notice that the dataset's identifier is formatted as `owner`/`id`. Jobs have GUID-like identifiers and the tasks have a `job-id/task-id` format where `task-id` is an digit-only string with leading zeros. Calling `PATCH` on an item is used to change some subset of its properties (not all properties are changeable).

#### Authentication

Preferred way of user authentication is through API keys. The intention is to implement the "bearer token" authentication mechanism defined under the OAuth 2.0 standard ([RFC](https://tools.ietf.org/html/rfc6750)), where the API key is essentially equal to the bearer token. An API key must be specified in almost all API requests (details in OpenAPI specification). There are three ways to specify the API key:

1. Query parameter: in the URL after the `?` sign add a parameter `api-key=<api_key_value>`
2. Header: in the request headers, specify the `X-API-KEY` header and assign the API key to it
3. Bearer token: in the request headers, specify the `Authorization` header with a string value formatted as `Bearer <api_key_value>`.

Only one of the three methods should be used. To obtain an API key the regular user must perform a login. This is done by invoking a GET request on `/users/login` with Client Side Basic HTTP authentication ([Wikipedia](https://en.wikipedia.org/wiki/Basic_access_authentication#Client_side)). This is done by adding an `Authorization` header, and the value is generated by taking a string formatted as `username:password` and encoding it with Base64 encoding and perpending it with `Basic` (followed by a single space).

Basic authentication can be used in place of API key authentication. However storing the API key for reuse is more convenient and safe than storing a password in plain text so using the API key is strongly advised.

**TO-DO**: Describe how training logs and debug data is accessed through the REST API.

**TO-DO:** Replacing `{task-id}` with `best` returns the task with the highest score.

**TO-DO:** Replacing user with `this` returns the current user.

### Web UI

TO-DO

## Data Structures

This section describes all data structures used in the system, regardless if they are stored as files or as records in a database.

* Container Registry

### Dataset

Ease.ml is a system that runs a configurable machine learning pipeline. Each element of that pipeline (i.e. module) operates on datasets. Our aim here is to define a unified dataset structure that is able to support most machine learning workloads. The main requirement of this structure is that datasets that correspond to the same machine learning problem (e.g. image classification, text translation etc.) should have the same structure.

Here we present an approach to the dataset structure design which is featured in the current ease.ml implementation. The current design is hierarchical in nature. Potential improvements are likely to happen in the future, as well as potentially shifting to a relational (i.e. table based) design. The reason to choose a hierarchical design is because it seemed to most naturally represent common machine learning datasets. 

Here we list a few examples of common machine learning problems and their corresponding datasets in an attempt to better understand them and to be able to design a general approach:

* **Real-valued vector classification**: This is a very common machine learning problem where inputs are *real-valued feature vectors* and outputs are *single-value categorical values*. Most classical machine learning models (e.g. logistic regression, SVM, etc) are built for this type of problem. A similar problem is real-valued vector regression that instead of categorical values outputs a single real value.
* **Image classification**: This machine learning problem is the biggest use case for Convolutional Neural Networks. It features an input *real-valued matrix (2D tensor)* representing the image and an output *single-value categorical value*.
* **Text translation**: This is a type of a sequence to sequence machine learning problem. If we consider the text to be represented as a sequence of words, and we represent those words as categorical values, then this problem has the same type of input and output data - a *variable-length sequence of categorical values*.

Before describing a general dataset structure, we can make some observations about machine learning datasets:

* Each dataset is made up of two sets of ***samples***, one for input and one for output. Each of the two sets should have elements of the same (logical) type.
* Some dimensions of a sample's type are fixed for all sample (e.g. dimensions of an image), while some dimensions are variable between individual samples (e.g. length of a text sequence).
* We want to support having tuples of features (e.g. pairs of images, an image and a sequence, sequences of images and feature vectors etc).
* The basic constituent type of data is either an N-dimensional floating point array (i.e. tensor) or a categorical value (i.e. category). Each categorical value must belong to a defined set of allowed values (i.e. class).

The current dataset structure is given below:

```
dataset/
  train/
  val/
    in/
    out/
      <sample-id>/
        <node-name-1>.ten.npy
        <node-name-2>.cat.txt
        <node-name-3>/
          <field-name-1>.ten.npy
          <field-name-2>.cat.txt
        [links.csv]
      <category-class-name-1>.class.txt
```

There are a few things to note about this dataset structure:

* We divide datasets into train and validation datasets, and each one is then divided into input and output sets. The structure of all of these subsets is the same.
* A sample ID in the input set must correspond to a sample ID in the output set in order to constitute a valid sample. (**TO-DO**: Change this, sample should ID should be above in and out)
* Tensors are all files with extension either `.ten.npy` (stored as numpy n-D arrays) or `.ten.csv` stored as CSV files. Tensors must have the same dimension across different samples.
* Categories are all files with extension `.cat.txt`. The categorical value (which is a single line in the text file) must be one of the lines in one of the category class files, thus signifying that the categorical value belongs to that class. Category fields must belong to the same class across different samples.
* A field file of type tensor and category has a similar meaning as a node file, except that it supports variable dimensions. We say that a node can have a variable number of instances. Specifically, the first dimension of a tensor is variable and represents the length of a sequence. The category file can have multiple lines. If a node has multiple fields, their dimensionality must be the same.
* The links file is optional and permits arbitrary linking of node instances. This file has two space separated values, each formatted as `<node-name>/<instance-idx>` where the instance ordinal is the zero-based index of a node instance.

Each dataset is constructed as a UNIX file structure. It can reside in a directory or inside a TAR or ZIP archive. There is a 1-1 mapping from file structure and file formats to the schema representation.

### Schema

A schema is a formal definition of the structure of data. In ease.ml it serves two purposes: (1) uniquely describe the structure of a dataset in an abstract way; (2) describe what type of dataset a pipeline module can accept as input and output. Note that to achieve the second purpose, we have to permit a schema to have dimensions that would be variable for a module, but constant for a particular dataset. The intuition is to permit image classification models to be applicable to datasets with input matrices and output categories, regardless of image dimensions in a particular dataset.

A schema is described in a JSON or YAML format that closely corresponds to the dataset structure:

```yaml
nodes:
  node-name-1:
    singleton: true
    type: tensor
    dim: [16, 16]
  node-name-2:
    singleton: true
    type: category
    class: category-class-name-1
  node-name-3:
    singleton: false
    fields:
      field-name-1:
        type: tensor
        dim: [16, 16]
      field-name-2:
        type: category
        class: category-class-1
    links:
      node-name-3: [1, 1]
classes:
  category-class-1:
    dim: 16
ref-constraints:
  cyclic: false
  undirected: false
  fan-in: false
```

There are a few things to note about this schema structure:

* Our schema describes a graph with multiple types of nodes.

* Tensor and category dimensions are given here as constant values. This is how they would appear for a dataset. However, for a module, they can be replaced with string identifiers to make them variable. Dimension matching rules are explained below.
* A node is a singleton if it doesn't permit multiple instances. This type of node doesn't have fields and it's file is stored in the base directory of a sample. A non-singleton node is the opposite of a singleton and its fields are stored in a subdirectory of a sample directory, in order to group them together.
* Nodes can be linked together. The link cardinality has an upper and lower bound (here both are 1). This lets us impose a certain structure on the graph. For example, to chain nodes in a list we have to specify the cardinality `(1,1)`. We would build a binary tree with a cardinality `(0,2)` and a general graph with cardinality `(0, null)`. Here `null` represents infinity.
* Referential constraints help us additionally impose restrictions on the structure of our graph.
  * `cyclic` specifies if we allow the links to form cycles.
  * `undirected` specifies if our graph is a directed or undirected. Links in undirected graphs must always be two-way (meaning node A points to node B and vice versa).
  * `fan-in` specifies if we permit a single node to have more than one incoming link. This is a way to differentiate trees from other types of graphs.

#### Schema Matching

This section is related to the second major purpose of having a schema: describing module input and output data type. The main difference between dataset schema and module schema is that module schema doesn't have to have constant dimensions, but instead we allow it to be described in a variable way. We are then able to perform a simple matching algorithm in order to decide if a dataset can be matched with a module.

Currently the only variability we permit in module schema is setting variable dimensions of tensors and category classes. This is simply achieved by specifying a string identifier in the place of a numeric constant. For example this schema represents any rectangular tensor:

```yaml
nodes:
  node-name-1:
    singleton: true
    type: tensor
    dim: [dim_a, dim_b]
```

The same goes for category class schema:

```yaml
nodes:
  node-name-1:
    singleton: true
    type: category
    class: category-class-name-1
classes:
  category-class-1:
    dim: num_categories
```

There are a few things to note about variable schemata:

* In module schema, dimensions don't have to be variable -- they can be numeric constants as well.
* Dimensions that are assigned the same name must correspond to the same numeric constant during matching. Therefore `dim: [dim_a, dim_a]` corresponds to a square.
* It is possible to use wildcard characters to specify a variable number of dimensions. These wildcard characters are appended to the variable name and signify: (1) one or more instances - `+`; or (2) zero or more instances - `*`. For example: `dim: [dim_a+, dim_b]` corresponds to any tensor with two or more dimensions where all but the last dimension must be equal.

### Model configuration

Models can be seen as functions that map their inputs to outputs given a set of *trainable* parameters (which we call *memory*) and *non-trainable* hyperparameters (which we call *configuration*). The trainable parameters are obtained through training by learning algorithms (such as gradient descent). Non-trainable hyperparameters are specified before training and optimized over a multitude of training runs through a process called *hyperparameter optimization*. This involves using a black-box function optimization method (such as Bayesian optimization) to search for optimal hyperparameter values over a predefined *feasible set*.

The feasible set is specified by a JSON or YAML file where a certain type of nested object gets treated as a special type of object called a *domain object*. Domain objects have at least one property prefixed with a period `.` sign. During optimization, each domain object gets collapsed into one of the elements of that domain. Every other element in the feasible set is treated as constant. Each domain type can have some specific properties that define the set of its elements. Here is a list of available domain types:

* **Integer** - Defined when the domain object has a `.int` property and a value that is a 2-element list representing the lower and upper bounds (inclusive) of the integer range. It is possible to specify a `.scale` property with a value set to either `linear` (default) or `exp`.
* **Real** - Defined when the domain object has a `.float` property and a value that is a 2-element list representing the lower and upper bounds of the real range. It is possible to add a `.scale` property to the object with a value set to either `linear` (default), `log` or `exp`.
* **Choice** - Defined when the domain dictionary has a `.choice` property and a value that is a list with multiple elements (at least one). The optimizer will choose one of the values from the list and treat them as categories of equal importance.

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

**TO-DO:** Explain how equality between configurations is implemented, as well as distance between them. Both features are important for Bayesian optimization.

### Data Directory Structure

Each ease.ml process has access to a working directory, the location of which is configurable. In case we want to run in a distributed setup, the working directory should be mounted to a shared/distributed file system (responsibility of an administrator). First we will explain the current basic design and then we will discuss briefly potential improvements if performance becomes an issue.

Currently all file-system data resides under the `shared` directory that is placed as a child of the working directory. We then further branch into sub-directories and sort all of the data based on the type of content.

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
    /jobs
        /[job-id]
            /[task-id]
                /config
                    /config.json
              /parameters
              /logs
                /train.log
                /predict.log
              /debug
                /train
                /predict
              /predictions
              /evaluation
                /evals.csv
    
    /scheduling
      /input
        /config
          /[job-id].json
        /history
          /[job-id]
            /[task-id].json
    /processes
        /[process-id]
            /[process-type].log            
```

#### Discussion and Future Improvements

The `images` directory holds all Docker images of modules as TAR files. We assume that processes that will be running containers from these images could reside on separate machines, so we need a centralized storage of all images that are uploaded to ease.ml. ***NOTE:*** Docker images are layered and a lot of them share large common layers (e.g. the OS layer). If our storage system was aware of these layers (e.g. leverage their unique hashes to identify them and keep some sort of layer cache), both storage and transfer would have been more efficient.

The `data` directory simply stores all datasets. Currently they are stored as files directly in the file system as this provides the easiest access. However, datasets are usually made up of a large number of small files which makes them a burden for the file system. Firstly, physical storage of a single file is in increments of the block size (which is usually 4KB) which means that even a file of 20 bytes will end up taking 4KB of space. Secondly, having so many files makes it difficult to manage the whole working directory because moving it or deleting it becomes very slow as there are so many files that need to be touched. That is why storing individual datasets as TAR archives might be beneficial.

The `scheduling` directory is purely used as a temp directory for the scheduler process. We store all inputs of the optimizer Docker container in here. This allows us to make incremental changes to the input data between individual calls to the optimizer instead of recreating the whole dataset from scratch each time.

The `processes` directory is used purely to store process logs.

The `jobs` directory stores all inputs and outputs of tasks executions. Storing outputs of all stages here permits reproducibility and fault tolerance as every intermediate step of the pipeline is stored. Currently the I/O overhead of this is not a big issue as our pipeline is quite simple. However, in the future if we add more sophisticated pipeline steps (which might be very simple data-transformation operations) the I/O overhead of saving every output to disk might become problematic. One approach to reducing this overhead would be to optimistically store this data in some in-memory file structure and then have some background process to eventually commit it to disk. Another approach would be to use sockets in-pipeline data transfer but then storing intermediate results becomes more tricky. Storing intermediate results is most important after training (as this is probably the most computationally costly operation). Therefore, a good idea might also be to try some hybrid approach between sockets and in-memory file structures.

In any case, even though now we store all data on disk for simplicity, data that most benefits from this is some more ***permanent*** data that is written once and doesn't change later. Data that is more ***transient***, such as temporary inputs don't necessarily need to be stored on disk. Future improvements of the data directory structure should consider how to make a more optimal split between these two categories of data.

### Database

The database is the central repository which acts as the second pillar of the complete state of the system (the first one being the file system). We use the MongoDB database because of its declared ease of use and scalability capabilities. In the future it is highly likely that we will switch over to some relational database such as PostgreSQL.

All ease.ml data is stored in a single database (name is configurable, default is `easeml`). We provide here a description of the collection and the schema of the documents we keep inside:

#### users

All users of the system. Users can have private models and data sets that are not visible to everyone. There is one predefined user: `root`.

* `id` - Unique string identifier.
* `password-hash` - SHA1 hash of the user's password.
* `name` - Display name of the user.
* `status` - User's status.
  * Possible values: `active`, `archived`
* `api-key` - When a user logs in, they are assigned an access token which they can use to interact with the REST API. The `root` user doesn't have a password but has an access token which is written in the console each time a `control` service is started.

#### processes

All "processes" that were ever running in the system. Note that these processes don't necessarily correspond with OS-level processes. We can run a `controller`, `scheduler` and `worker` "process" all under one OS-level process.

* `id` - UUID-like identifier.
* `process-id` - OS-level process ID.
* `host-id` - Host name.
* `host-address` - IP address of the host (if available).
* `start-time` - Time when the process was started.
* `type` - Type of the process. Possible values: `controller`, `worker`, `scheduler`
* `resource` - Resource allocated to a worker process. Can be: `cpu` or `gpu`.
* `status` - Status of the process. Possible values: `idle`, `working`, `terminated`

#### datasets

All datasets.

* `id` - Unique string identifier.
* `user` - Owner of the data set. Data sets are visible only by their owner by default. Data sets created by the root user are visible for all and the root user can see all datasets.
* `name` - Display name of the dataset.
* `description` - Description of the dataset written in Markdown.
* `schema-in`, `schema-out` - Strings with serialized JSON objects representing input and output schema of the dataset.
* `source` - Source from where the data set was obtained. Possible values: `upload`, `local`, `download`
* `source-address` - If `null` then the source is a HTTP file upload. Otherwise its value depends on source type. For `local` source it is the path to a file on a mounted file system (accessible to ease.ml). For `download` source it is a URL address from which the data set can be downloaded.
* `creation-time` - Time when the dataset was crated.
* `status` - Status of the data set.
  * Possible values:
    * `created` - Dataset object created in the database.
    * `transferred` - Dataset transfer completed. Dataset resides in a temp state, possibly as a TAR or ZIP archive.
    * `unpacked` - Dataset is unpacked and all files are placed where they will permanently reside.
    * `validated` - Dataset schema inferred and validated. Ready to use.
    * `archived` - We cannot use it in future jobs.
    * `error` - Special state when an error has been encountered.
* `status-message` - In case of an error, the error message is written here.
* `process` - ID of the process that currently has a lock on the dataset and is handling it.

#### modules

All modules that represent building blocks of the pipeline or the optimizer.

* `id` - Identifier. Should be a human readable string.
* `user` - Owner of the module.
* `type` - Type of module. Possible values:
  - `model` - Machine learning model.
  - `objective` - Objective function that measures the quality of predictions.
  - `optimizer` - Black box optimizer that is plugged into the scheduler.
* `label` - User-defined label to additionally categorize modules.
* `name` - Human readable name of the model.
* `description` - Description of the module written in Markdown.
* `schema-in`, `schema-out` - Strings with serialized JSON objects representing input and output schema of the module. Can be empty when appropriate (e.g. for optimizer modules).
* `config-space` - Configuration space for `model` modules. Stored as string serialized JSON.
* `image` - Identifier of the Docker image which contains the module.
* `source` - Source from where the module image was obtained. Possible values: `upload`, `local`, `download`, `registry`
* `source-address` - If `null` then the source is a HTTP file upload. Otherwise its value depends on source type. For `local` source it is the path to a file on a mounted file system (accessible to ease.ml). For `download` source it is a URL address from which the module image can be downloaded as TAR file. For `registry` source the it is the string used to pull the image from a remote registry.
* `creation-time` - Time when the module was created.
* `status` - Status of the module.
  - Possible values:
    - `created` - Module object created in the database.
    - `transferred` - Module transfer completed.
    - `active` - Module ready to use.
    - `archived` - We cannot use it in future jobs.
    - `error` - Special state when an error has been encountered.
* `status-message` - In case of an error, the error message is written here.
* `process` - ID of the process that currently has a lock on the module and is handling it.

#### jobs

Collection of jobs that can be picked up by schedulers.

* `id` - UUID-like identifier.
* `user` - User that submitted the job.
* `dataset` - Id of the dataset used for training/evaluation.
* `models` - List of id's of all models that will be part of the model selection search space.
* `config-space` - String with serialized JSON representation of the complete search space of this job.
* `accept-new-models` - Boolean. If set to `true` (default) then whenever a new models is added, if it is applicable to the dataset it will be automatically added to the `models` list.
* `objective` - Objective to use to use when evaluating models.
* `alt-objectives` - Additional objectives to run on the models' predictions. These don't impact the optimization. Here as a placeholder, currently ignored. **TO-DO**: Consider enabling adding new alt objectives for tasks that have been completed.
* `max-tasks` - Limit the budget for a job given as the maximum number of tasks that can be completed before we declare the job to be completed.
* `creation-time` - Time when the job was created.
* `running-time` - Nested object, has two fields: `start` and `end` - times when the job started to run and when running was stopped (either due to completion, termination, cancellation or error).
* `pause-start-time` - Start time of the pause state.
* `pause-duration` - Computed field (not stored in database): `time.now() - pause-start-time`
* `prev-pause-duration` - Duration of all previous pauses. Each time we exit a paused state we increment this field by the value of `pause-duration`.
* `runnint-duration` - Computed field (not stored in database). Holds the total running time minus the total pause time.
* `status` - Status of the job.
  - Possible values: `scheduled`, `running`,  `pausing`, `paused`, `resuming`, `completed`, `terminating`, `terminated`, `error`
  - When the job is in `pausing` or `terminating` state, then it is waiting for all its tasks get transferred to the `paused` or `terminated` state. After that it transfers to `paused` or `terminated`.
  - Job is `completed` when the completion criteria was met (e.g. maximum tasks budget is fulfilled). The `terminated` state is used for jobs that are manually terminated.
* `status-message` - In case of an error, the error message is written here.
* `process` - ID of the process that currently has a lock on the module and is handling it.

#### tasks

Collection of tasks that can be picked up by workers. Its schema is denormalized as it contains come fields copied over from `jobs`.

* `id` - Unique identifier. Digit based string. (e.g. `00001`, `00002`, etc)
* `job` - Identifier of the job that this task was spawned from.
* `process` - Identifier of the process that is handling the task.
* `user` - User that created this task's job.
* `dataset` - Id of the dataset used for training/evaluation. 
* `model` - Identifier of the target model.
* `objective` - Identifier of the objective to apply.
* `config` - String serialized JSON that represents a concrete model configuration that was instantiated from the job's `config-space` by an optimizer.
* `quality` - Value of the quality metric of the trained model over the validation data set obtained from the objective function. Available after the `evaluating` stage is finished.
* `quality-train` - Value of the quality metric over the training data set obtained from the objective function.
* `quality-expected` - When a task is scheduled, the (Bayesian-based) optimizers provide an expected quality metric value that is used for making scheduling choices.
* `alt-qualities` - Quality metric values of additional objectives (if defined in the `job`). These don't impact the optimization.
* `status` - Status of the task.
  * Possible values: `scheduled`, `running`, `pausing`, `paused`, `completed`, `terminating`, `terminated`, `canceled`, `error`
  * Stages are atomic units of execution. A task can be paused or terminated in-between stages. To signal that a task should be paused/terminated it is put in the `pausing`/`terminating` state and once a stage is completed it will transfer to the `paused`/`terminated` stage.
  * Tasks are `cancelled` by the system in order to discard them if they have been scheduled, but the budget for their job has been reached before they were completed.
* `status-message` - In case of an error, the error message is written here.
* `stage` - Current execution stage. Should be ignored if the status is not `running` or `paused`.
  * Possible values: `begin`, `training`, `predicting`, `evaluating`, `end`
* `creation-time` - Time when the task was created by the scheduler.
* `stage-times` - Nested object, has three fields: `training`, `predicting` and `evaluating`. Each itself has a nested object as value with fields `start` and `end`. Times of start end end of each stage are recorded here.
* `stage-durations` - Computed field (not stored in database). Nested object, has three fields: `training`, `predicting` and `evaluating`. Each field holds the duration of the corresponding stage.
* `running-duration` - Computed field (not stored in database). Sum of all stage running durations.

### Configuring ease.ml

The approach to ease.ml configuration is influenced by the "twelve-factor app" ideology. The entry point to the ease.ml application is the `easeml` CLI command. Essentially, each invocation of the `easeml` command branches out in different subcommands, each one having a set of configuration properties along with sensible default values which intend to reduce user interaction as much as possible, at least for the most common usage scenarios. Values of these properties should (in most cases) be specifiable through the following means (subsequent ones take precedence over former ones):

1. Configuration file (default is in `<working_dir>/config.yml`), properties are stored by their dash-separated (WHICH CASE?) names
2. Environment variables, names should be prefixed with `EASEML_` and names should be underscore-separated (WHICH CASE?)
3. Explicitly specified command line flags

The configuration file location can be specified explicitly, or implicitly by specifying the working directory in which case it is assumed that the configuration file is stored in that working directory. To prevent circular logic, the working directory location cannot be specified inside a configuration file.

We don't specify all the configuration properties here. The definitive reference should be the the source code that implements all the `easeml` commands.

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