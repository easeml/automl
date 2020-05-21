"""
Implementation of the `Job` class.
"""
import pyrfc3339

from copy import deepcopy
from datetime import datetime, timedelta
from enum import Enum
from typing import Dict, Optional, Any, Iterator, Tuple, List

from .core import Connection
from .common import TimeInterval
from .dataset import Dataset
from .module import Module
from .process import Process
from .user import User
from .type import ApiType, ApiQuery, ApiQueryOrder


class JobStatus(Enum):
    SCHEDULED = "scheduled"
    RUNNING = "running"
    PAUSING = "pausing"
    PAUSED = "paused"
    RESUMING = "resuming"
    COMPLETED = "completed"
    TERMINATING = "terminating"
    TERMINATED = "terminated"
    ERROR = "error"


class Job(ApiType['Job']):
    """The Job class contains information about datasets.

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
    def create(cls, dataset: Dataset, objective: Module, models: List[Module],
               accept_new_models: bool = True, max_tasks: int = 100,
               alt_objectives: Optional[List[Module]] = None, config_space: Optional[Dict[str, Any]] = None) -> 'Job':
        init_dict: Dict[str, Any] = {"id": None}
        if dataset is not None:
            init_dict["dataset"] = dataset.id
        if objective is not None:
            init_dict["objective"] = objective.id
        if models is not None:
            init_dict["models"] = [x.id for x in models]
        if accept_new_models is not None:
            init_dict["accept-new-models"] = accept_new_models
        if max_tasks is not None:
            init_dict["max-tasks"] = max_tasks
        if alt_objectives is not None:
            init_dict["alt-objectives"] = [x.id for x in alt_objectives]
        if config_space is not None:
            init_dict["config-space"] = config_space
        return Job(init_dict)
    
    @classmethod
    def create_ref(cls, id: str) -> 'Job':
        return Job({"id": id})

    @property
    def id(self) -> str:
        return self._dict["id"]

    @property
    def user(self) -> Optional[User]:
        value = self._dict.get("user")
        return User({"id": value}) if value is not None else None

    @property
    def dataset(self) -> Optional[Dataset]:
        value = self._dict.get("dataset")
        return Dataset({"id": value}) if value is not None else None

    @property
    def models(self) -> Optional[List[Module]]:
        value = self._dict.get("models")
        return [Module({"id": x}) for x in value] if value is not None else None

    @property
    def config_space(self) -> Optional[str]:
        value = self._dict.get("config-space")
        return str(value) if value is not None else None

    @property
    def accept_new_models(self) -> Optional[bool]:
        value = self._updates.get("accept-new-models") or self._dict.get("accept-new-models")
        return bool(value) if value is not None else None

    @accept_new_models.setter
    def accept_new_models(self, value: Optional[bool] = None) -> None:
        if value is not None:
            self._updates["accept-new-models"] = value
        else:
            self._updates.pop("accept-new-models")

    @property
    def objective(self) -> Optional[Module]:
        value = self._dict.get("objective")
        return Module({"id": value}) if value is not None else None

    @property
    def alt_objectives(self) -> Optional[List[Module]]:
        value = self._dict.get("alt-objectives")
        return [Module({"id": x}) for x in value] if value is not None else None

    @property
    def max_tasks(self) -> Optional[int]:
        value = self._updates.get("max-tasks") or self._dict.get("max-tasks")
        return int(value) if value is not None else None

    @max_tasks.setter
    def max_tasks(self, value: Optional[int] = None) -> None:
        if value is not None:
            self._updates["max-tasks"] = value
        else:
            self._updates.pop("max-tasks")

    @property
    def creation_time(self) -> Optional[datetime]:
        value = self._dict.get("creation-time")
        return pyrfc3339.parse(value) if value is not None else None

    @property
    def running_time(self) -> Optional[TimeInterval]:
        value = self._dict.get("running-time")
        return TimeInterval(value) if value is not None else None

    @property
    def running_duration(self) -> Optional[timedelta]:
        value = self._dict.get("running-duration")
        return timedelta(milliseconds=int(value)) if value is not None else None

    @property
    def pause_duration(self) -> Optional[timedelta]:
        value = self._dict.get("pause-duration")
        return timedelta(milliseconds=int(value)) if value is not None else None

    @property
    def status(self) -> Optional[JobStatus]:
        value = self._updates.get("status") or self._dict.get("status")
        return JobStatus(value) if value is not None else None

    @status.setter
    def status(self, value: Optional[JobStatus] = None) -> None:
        if value is not None:
            self._updates["status"] = value.value
        else:
            self._updates.pop("status")

    @property
    def status_message(self) -> Optional[str]:
        value = self._dict.get("status-message")
        return str(value) if value is not None else None

    @property
    def process(self) -> Optional[Process]:
        value = self._dict.get("process")
        return Process({"id": value}) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)

    def post(self, connection: Connection) -> 'Job':
        url = connection.url("jobs")
        return self._post(connection, url)

    def patch(self, connection: Connection) -> 'Job':
        url = connection.url("jobs/" + self.id)
        return self._patch(connection, url)

    def get(self, connection: Connection) -> 'Job':
        url = connection.url("jobs/" + self.id)
        return self._get(connection, url)

class JobQuery(ApiQuery['Job', 'JobQuery']):

    VALID_SORTING_FIELDS = ["user", "dataset", "objective", "creation-time", "running-time-start", "running-time-end", "status"]

    def __init__(self, id: Optional[List[str]] = None, user: Optional[User] = None,
                 dataset: Optional[Dataset] = None, model: Optional[Module] = None,
                 objective: Optional[Module] = None, alt_objective: Optional[Module] = None,
                 status: Optional[JobStatus] = None, accept_new_models: Optional[bool] = None,                
                 order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        super().__init__(order_by, order, limit, cursor)
        self.T = Job

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
        if status is not None:
            self._query["status"] = status.value
        if accept_new_models is not None:
            self._query["accept-new-models"] = accept_new_models

    def run(self, connection: Connection) -> Tuple[List[Job], Optional['JobQuery']]:
        url = connection.url("jobs")
        return self._run(connection, url)
