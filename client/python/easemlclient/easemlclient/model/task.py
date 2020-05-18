"""
Implementation of the `Task` class.
"""
from copy import deepcopy
from datetime import timedelta
from enum import Enum
from typing import Dict, Optional, Any, Iterator, Tuple, List

from .core import Connection
from .common import TimeInterval
from .dataset import Dataset
from .job import Job
from .module import Module
from .process import Process
from .user import User
from .type import ApiType, ApiQuery, ApiQueryOrder


class TaskStatus(Enum):
    SCHEDULED = "scheduled"
    RUNNING = "running"
    PAUSING = "pausing"
    PAUSED = "paused"
    COMPLETED = "completed"
    TERMINATING = "terminating"
    TERMINATED = "terminated"
    CANCELED = "canceled"
    ERROR = "error"


class TaskStage(Enum):
    BEGIN = "begin"
    TRAINING = "training"
    PREDICTING = "predicting"
    EVALUATING = "evaluating"
    END = "end"


class TaskStageIntervals:
    """The TaskStageIntervals class contains information about task stage intervals.

    ...
    Attributes:
    -----------
    identifier: str
        A unique identifier of the user (i.e. the username).
    name: str
        The full name of the user.
    status: str
        The current status of the user. Can be 'active' or 'archived'.
    """

    def __init__(self, input: Dict[str, Any]) -> None:
        self._dict: Dict[str, Any] = deepcopy(input)

    @property
    def training(self) -> Optional[TimeInterval]:
        value = self._dict.get("training")
        return TimeInterval(value) if value is not None else None

    @property
    def predicting(self) -> Optional[TimeInterval]:
        value = self._dict.get("predicting")
        return TimeInterval(value) if value is not None else None

    @property
    def evaluating(self) -> Optional[TimeInterval]:
        value = self._dict.get("evaluating")
        return TimeInterval(value) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)


class TaskStageDurations:
    """The TaskStageIntervals class contains information about task stage intervals.

    ...
    Attributes:
    -----------
    identifier: str
        A unique identifier of the user (i.e. the username).
    name: str
        The full name of the user.
    status: str
        The current status of the user. Can be 'active' or 'archived'.
    """

    def __init__(self, input: Dict[str, Any]) -> None:
        self._dict: Dict[str, Any] = deepcopy(input)

    @property
    def training(self) -> Optional[timedelta]:
        value = self._dict.get("training")
        return timedelta(milliseconds=int(value)) if value is not None else None

    @property
    def predicting(self) -> Optional[timedelta]:
        value = self._dict.get("predicting")
        return timedelta(milliseconds=int(value)) if value is not None else None

    @property
    def evaluating(self) -> Optional[timedelta]:
        value = self._dict.get("evaluating")
        return timedelta(milliseconds=int(value)) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)


class Task(ApiType['Task']):
    """The Task class contains information about datasets.

    ...
    Attributes:
    -----------
    identifier: str
        A unique identifier of the user (i.e. the username).
    name: str
        The full name of the user.
    status: str
        The current status of the user. Can be 'active' or 'archived'.
    """

    def __init__(self, input: Dict[str, Any]) -> None:
        if "id" not in input:
            raise ValueError("Invalid input dictionary: It must contain an 'id' key.")

        super().__init__(input)
    
    @classmethod
    def create_ref(cls, id: str) -> 'Task':
        return Task({"id": id})

    @property
    def id(self) -> str:
        return self._dict["id"]

    @property
    def job(self) -> Optional[Job]:
        value = self._dict.get("job")
        return Job({"id": value}) if value is not None else None

    @property
    def process(self) -> Optional[Process]:
        value = self._dict.get("process")
        return Process({"id": value}) if value is not None else None

    @property
    def user(self) -> Optional[User]:
        value = self._dict.get("user")
        return User({"id": value}) if value is not None else None

    @property
    def dataset(self) -> Optional[Dataset]:
        value = self._dict.get("dataset")
        return Dataset({"id": value}) if value is not None else None

    @property
    def model(self) -> Optional[Module]:
        value = self._dict.get("model")
        return Module({"id": value}) if value is not None else None

    @property
    def objective(self) -> Optional[Module]:
        value = self._dict.get("objective")
        return Module({"id": value}) if value is not None else None

    @property
    def alt_objectives(self) -> Optional[List[Module]]:
        value = self._dict.get("alt-objectives")
        return [Module({"id": x}) for x in value] if value is not None else None

    @property
    def config(self) -> Optional[str]:
        value = self._dict.get("config")
        return str(value) if value is not None else None

    @property
    def quality(self) -> Optional[float]:
        value = self._dict.get("quality")
        return float(value) if value is not None else None

    @property
    def quality_train(self) -> Optional[float]:
        value = self._dict.get("quality-train")
        return float(value) if value is not None else None

    @property
    def quality_expected(self) -> Optional[float]:
        value = self._dict.get("quality-expected")
        return float(value) if value is not None else None

    @property
    def alt_qualities(self) -> Optional[List[float]]:
        value = self._dict.get("alt-qualities")
        return [float(x) for x in value] if value is not None else None

    @property
    def status(self) -> Optional[TaskStatus]:
        value = self._updates.get("status") or self._dict.get("status")
        return TaskStatus(value) if value is not None else None

    @status.setter
    def status(self, value: Optional[TaskStatus] = None) -> None:
        if value is not None:
            self._updates["status"] = value.value
        else:
            self._updates.pop("status")

    @property
    def status_message(self) -> Optional[str]:
        value = self._dict.get("status-message")
        return str(value) if value is not None else None

    @property
    def stage(self) -> Optional[TaskStage]:
        value = self._dict.get("stage")
        return TaskStage(value) if value is not None else None

    @property
    def stage_times(self) -> Optional[TaskStageIntervals]:
        value = self._dict.get("stage-times")
        return TaskStageIntervals(value) if value is not None else None

    @property
    def stage_durations(self) -> Optional[TaskStageDurations]:
        value = self._dict.get("stage-durations")
        return TaskStageDurations(value) if value is not None else None

    @property
    def running_duration(self) -> Optional[timedelta]:
        value = self._dict.get("running-duration")
        return timedelta(milliseconds=int(value)) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)

    def get(self, connection: Connection) -> 'Task':
        url = connection.url("tasks/" + self.id)
        return self._get(connection, url)

    def patch(self, connection: Connection) -> 'Task':
        url = connection.url("tasks/" + self.id)
        return self._patch(connection, url)
    
    def get_predictions(self, connection: Connection) -> bytes:
        url = connection.url("tasks/" + self.id + "/predictions.tar")
        return self._download(connection, url)
    
    def get_parameters(self, connection: Connection) -> bytes:
        url = connection.url("tasks/" + self.id + "/parameters.tar")
        return self._download(connection, url)
    
    def get_image(self, connection: Connection) -> bytes:
        url = connection.url("tasks/" + self.id + "/image/download")
        return self._download(connection, url)


class TaskQuery(ApiQuery['Task', 'TaskQuery']):

    VALID_SORTING_FIELDS = ["id", "process", "job", "user", "dataset", "objective", "model", "quality", "quality-train", "quality-expected", "creation-time", "status", "stage"]

    def __init__(self, id: Optional[List[str]] = None, user: Optional[User] = None,
                 dataset: Optional[Dataset] = None, model: Optional[Module] = None,
                 objective: Optional[Module] = None, alt_objective: Optional[Module] = None,
                 process: Optional[Process] = None, job: Optional[Job] = None,
                 status: Optional[TaskStatus] = None, stage: Optional[TaskStage] = None,                
                 order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        super().__init__(order_by, order, limit, cursor)
        self.T = Task

        if id is not None:
            self._query["id"] = id
        if user is not None:
            self._query["user"] = user.id
        if dataset is not None:
            self._query["dataset"] = dataset.id
        if model is not None:
            self._query["model"] = model.id
        if objective is not None:
            self._query["objective"] = objective.id
        if alt_objective is not None:
            self._query["alt-objective"] = alt_objective.id
        if process is not None:
            self._query["process"] = process.id
        if job is not None:
            self._query["job"] = job.id
        if status is not None:
            self._query["status"] = status.value
        if stage is not None:
            self._query["stage"] = stage.value

    def run(self, connection: Connection) -> Tuple[List[Task], Optional['TaskQuery']]:
        url = connection.url("tasks")
        return self._run(connection, url)