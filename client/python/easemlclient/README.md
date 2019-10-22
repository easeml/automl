# Ease.ml Client

This is the Python implementation of the ease.ml client.

## Installation

This package is available on PyPI.

```bash
pip install easemlclient
```

## Example usage

### Establishing a connection

To use the client API we first need to create a connection object that we will be using to target the running easeml instance. The connection must be inialized with a host name (here we use localhost) and either the API key or a username and password.

```python
from easemlclient.model import Connection

connection = Connection(host="localhost:8080", api_key="some-api-key")
```

### Querying Collections

Then we can query all the running jobs. To do that we need to create a `JobQuery` instance which we use to specify the parameters of our query. For example, we can query all completed jobs. To get the result we call the `run()` method of the query object and pass the connection instance.

```python
from easemlclient.model import JobQuery

query = JobQuery(status="completed")
result, next_query = query.run(connection)
```

The result will contain a list of `Job` objects taht satisfy our query. Results are paginated to limit the size of each request. If there are more pages to be loaded, then the `next_query` variable will contain a `JobQuery` instance that we can run and return the next page. The full pattern for loading all jobs is the following:

```python
from easemlclient.model import JobQuery

result, query = [], JobQuery(status="completed")

next_result, next_query = [], query
while next_query is not None:
    next_result, next_query = query.run(connection)
    result.extend(next_result)
```

We can take the first completed job and get a list of its tasks.

```python
job = result[0]
tasks = job.tasks
```

The `tasks` list actually contains "shallow" instances of the `Task` class. This means that each instance contains only the task's `id` field and no other fields. This is normal because the `Job` object has only references to tasks, not entire tasks. To get a full version of a task given a "shallow" instance, we use the `get()` method.

```python
task = tasks[0].get(connection)
```

### Querying Specific Objects

The `Task` object can also be used to query tasks by their ID. We simply create a new "shallow" instance using a task ID and call the `get()` method.

```python
from easemlclient.model import Task

task = Task(id="some-task-id").get(connection)
```

### Creating Objects

We have the ability to create certain objects, such as `Dataset`, `Module` and `Job`. We do this by initializing an instance of that object, assigning values to relevant fields and calling the `post()` method. Here is an example of creating a dataset object along with uploading of a dataset.

```python

from easemlclient.model import Dataset, DataSource

dataset = Dataset(id="test_dataset_1", source=DataSource.UPLOAD, name="Test Dataset 1").post(connection)

with open("test_dataset_1.tar", "r") as f:
    dataset.upload(connection=connection, data=f)
```

### Starting a new training Job and monitoring it

Here we show a slightly more complex example that demonstrates how to start a model selection and tuning job given a previously uploaded dataset.

We will first fetch the dataset object in order to be able to access its schema.

```python
from easemlclient.model import Dataset

dataset = Dataset(id="test_dataset_1").get(connection)
```

Then we query all models that are applicable to the given dataset. We use the `ModuleQuery` class for this.

```python

from easemlclient.model import ModuleQuery, ModuleType

query = ModuleQuery(type=ModuleType.MODEL, status=ModuleStatus.ACTIVE,
                    schema_in=dataset.schema_in, schema_out=dataset.schema_out)

# We assume that the result does not contain more than one page.
models, _ = query.run(connection)
```

We do the same for objectives.

```python

from easemlclient.model import ModuleQuery, ModuleType

query = ModuleQuery(type=ModuleType.OBJECTIVE, status=ModuleStatus.ACTIVE,
                    schema_in=dataset.schema_in, schema_out=dataset.schema_out)
objectives, _ = query.run(connection)

# We will simply pick the first objective here.
objective = objectives[0]
```

Then we are ready to create a job.

```python
from easemlclient.model import Job

job = Job(dataset=dataset, objective=objective, models=models, max_tasks=20).post(connection)
```

With `max_tasks` we specify the number of tasks to run before a job's status will become `completed`. We can keep querying the job to check the status.

```python
from time import sleep
from easemlclient.model import JobStatus

while job.get(connection).status != JobStatus.COMPLETED:
    time.sleep(10)
```

Once the job is completed, we can get the task with the best result.

```python
from easemlclient.model import TaskQuery, ApiQueryOrder

tasks, _ = TaskQuery(job=job, order_by="quality", order=ApiQueryOrder.DESC).run(connection)

best_task = tasks[0].get(connection)
```

Finally, we can download the Docker image of the best task and save it as a tar file.

```python
image = best_task.get_image(connection)
open("/output/path/to/image.tar", "wb").write(image)
```
