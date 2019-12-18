"""
Implementation of the `Process` class.
"""
import pyrfc3339

from copy import deepcopy
from datetime import datetime
from enum import Enum
from typing import Dict, Optional, Any, Iterator, Tuple, List

from .core import Connection
from .type import ApiType, ApiQuery, ApiQueryOrder


class ProcType(Enum):
    CONTROLLER = "controller"
    WORKER = "worker"
    SCHEDULER = "scheduler"


class ProcStatus(Enum):
    IDLE = "idle"
    WORKING = "working"
    TERMINATED = "terminated"


class Process(ApiType['Process']):
    """The Process class contains information about processes.

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

        self._dict: Dict[str, Any] = deepcopy(input)
    
    @classmethod
    def create_ref(cls, id: str) -> 'Process':
        return Process({"id": id})

    @property
    def id(self) -> str:
        return self._dict["id"]

    @property
    def process_id(self) -> Optional[int]:
        value = self._dict.get("process-id")
        return int(value) if value is not None else None

    @property
    def host_id(self) -> Optional[str]:
        value = str(self._dict.get("host-id"))
        return str(value) if value is not None else None

    @property
    def host_address(self) -> Optional[str]:
        value = self._dict.get("host-address")
        return str(value) if value is not None else None

    @property
    def start_time(self) -> Optional[datetime]:
        value = self._dict.get("start-time")
        return pyrfc3339.parse(value) if value is not None else None

    @property
    def last_keepalive(self) -> Optional[datetime]:
        value = self._dict.get("last-keepalive")
        return pyrfc3339.parse(value) if value is not None else None

    @property
    def type(self) -> Optional[ProcType]:
        value = self._dict.get("type")
        return ProcType(value) if value is not None else None

    @property
    def resource(self) -> Optional[str]:
        value = self._dict.get("resource")
        return str(value) if value is not None else None

    @property
    def status(self) -> Optional[ProcStatus]:
        value = self._dict.get("status")
        return ProcStatus(value) if value is not None else None

    @property
    def running_ordinal(self) -> Optional[int]:
        value = self._dict.get("running-ordinal")
        return int(value) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)

    def get(self, connection: Connection) -> 'Process':
        url = connection.url("processes/" + self.id)
        return self._get(connection, url)

class ProcessQuery(ApiQuery['Process', 'ProcessQuery']):

    VALID_SORTING_FIELDS = ["id", "process-id", "host-id", "host-address", "start-time", "type", "resource", "status"]

    def __init__(self, id: Optional[List[str]] = None, process_id: Optional[int] = None,
                 host_id: Optional[str] = None, host_address: Optional[str] = None,
                 type: Optional[ProcType] = None, resource: Optional[str] = None,
                 status: Optional[ProcStatus] = None,
                 order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        super().__init__(order_by, order, limit, cursor)
        self.T = Process

        if id is not None:
            self._query["id"] = id
        if process_id is not None:
            self._query["process-id"] = process_id
        if host_id is not None:
            self._query["host-id"] = host_id
        if host_address is not None:
            self._query["host-address"] = host_address
        if type is not None:
            self._query["type"] = type.value
        if resource is not None:
            self._query["resource"] = resource
        if status is not None:
            self._query["status"] = status.value

    def run(self, connection: Connection) -> Tuple[List[Process], Optional['ProcessQuery']]:
        url = connection.url("processes")
        return self._run(connection, url)
