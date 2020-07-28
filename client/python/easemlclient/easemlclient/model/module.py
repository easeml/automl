"""
Implementation of the `Module` class.
"""
import pyrfc3339

from copy import deepcopy
from datetime import datetime
from enum import Enum
from io import FileIO
from tusclient import client as tus_client
from typing import Dict, Optional, Any, Iterator, Tuple, List

from .core import Connection
from .process import Process
from .user import User
from .type import ApiType, ApiQuery, ApiQueryOrder


class ModuleType(Enum):
    MODEL = "model"
    OBJECTIVE = "objective"
    OPTIMIZER = "optimizer"


class ModuleSource(Enum):
    UPLOAD = "upload"
    LOCAL = "local"
    DOWNLOAD = "download"
    REGISTRY = "registry"


class ModuleStatus(Enum):
    CREATED = "created"
    TRANSFERRED = "transferred"
    ACTIVE = "active"
    ARCHIVED = "archived"
    ERROR = "error"


class Module(ApiType['Module']):
    """The Module class contains information about datasets.

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
    def create(cls, id: str, type: Optional[ModuleType] = None, label: Optional[str] = None,
               source: Optional[ModuleSource] = None, source_address: Optional[str] = None,
               name: Optional[str] = None, description: Optional[str] = None,) -> 'Module':
        init_dict: Dict[str, Any] = {"id": id}
        if type is not None:
            init_dict["type"] = type
        if label is not None:
            init_dict["label"] = label
        if source is not None:
            init_dict["source"] = source
        if source_address is not None:
            init_dict["source-address"] = source_address
        if name is not None:
            init_dict["name"] = name
        if description is not None:
            init_dict["description"] = description
        return Module(init_dict)
    
    @classmethod
    def create_ref(cls, id: str) -> 'Module':
        return Module({"id": id})

    @property
    def id(self) -> str:
        return self._dict["id"]

    @property
    def user(self) -> Optional[User]:
        value = self._dict.get("user")
        return User({"id": value}) if value is not None else None

    @property
    def type(self) -> Optional[ModuleType]:
        value = self._dict.get("type")
        return ModuleType(value) if value is not None else None

    @property
    def label(self) -> Optional[str]:
        value = self._dict.get("label")
        return str(value) if value is not None else None

    @property
    def name(self) -> Optional[str]:
        value = self._updates.get("name") or self._dict.get("name")
        return str(value) if value is not None else None

    @name.setter
    def name(self, value: Optional[str] = None) -> None:
        if value is not None:
            self._updates["name"] = value
        else:
            self._updates.pop("name")

    @property
    def description(self) -> Optional[str]:
        value = self._updates.get("description") or self._dict.get("description")
        return str(value) if value is not None else None

    @description.setter
    def description(self, value: Optional[str] = None) -> None:
        if value is not None:
            self._updates["description"] = value
        else:
            self._updates.pop("description")

    @property
    def schema_in(self) -> Optional[str]:
        value = self._dict.get("schema-in")
        return str(value) if value is not None else None

    @property
    def schema_out(self) -> Optional[str]:
        value = self._dict.get("schema-out")
        return str(value) if value is not None else None

    @property
    def config_space(self) -> Optional[str]:
        value = self._dict.get("config-space")
        return str(value) if value is not None else None

    @property
    def source(self) -> Optional[ModuleSource]:
        value = self._dict.get("source")
        return ModuleSource(value) if value is not None else None

    @property
    def source_address(self) -> Optional[str]:
        value = self._dict.get("source-address")
        return str(value) if value is not None else None

    @property
    def creation_time(self) -> Optional[datetime]:
        value = self._dict.get("creation-time")
        return pyrfc3339.parse(value) if value is not None else None

    @property
    def status(self) -> Optional[ModuleStatus]:
        value = self._updates.get("status") or self._dict.get("status")
        return ModuleStatus(value) if value is not None else None

    @status.setter
    def status(self, value: Optional[ModuleStatus] = None) -> None:
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

    def post(self, connection: Connection) -> 'Module':
        url = connection.url("modules")
        return self._post(connection, url)

    def patch(self, connection: Connection) -> 'Module':
        url = connection.url("modules/" + self.id)
        return self._patch(connection, url)

    def get(self, connection: Connection) -> 'Module':
        url = connection.url("modules/" + self.id)
        return self._get(connection, url)

    def upload(self, connection: Connection, data: FileIO, file_name: Optional[str] = None) -> None:
        url = connection.url("modules/%s/upload" % self.id)
        metadata = {"filename" : file_name}

        # Initialize the client for the TUS upload protocol. Apply the authentication header.
        client = tus_client.TusClient(url)
        connection.auth(client)

        uploader = client.uploader(file_stream=data, chunk_size=200)
        uploader.upload()


class ModuleQuery(ApiQuery['Module', 'ModuleQuery']):

    VALID_SORTING_FIELDS = ["id", "user", "type", "label", "source", "source-address", "creation-time", "status"]

    def __init__(self, id: Optional[List[str]] = None, user: Optional[User] = None,
                 type: Optional[ModuleType] = None, label: Optional[str] = None,
                 status: Optional[ModuleStatus] = None, source: Optional[ModuleSource] = None,
                 source_address: Optional[str] = None,
                 schema_in: Optional[str] = None, schema_out: Optional[str] = None,                 
                 order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        super().__init__(order_by, order, limit, cursor)
        self.T = Module

        if id is not None:
            self._query["id"] = id
        if user is not None:
            self._query["user"] = user.id
        if type is not None:
            self._query["type"] = type
        if label is not None:
            self._query["label"] = label
        if status is not None:
            self._query["status"] = status
        if source is not None:
            self._query["source"] = source
        if source_address is not None:
            self._query["source-address"] = source_address
        if schema_in is not None:
            self._query["schema-in"] = schema_in
        if schema_out is not None:
            self._query["schema-out"] = schema_out

    def run(self, connection: Connection) -> Tuple[List[Module], Optional['ModuleQuery']]:
        url = connection.url("modules")
        return self._run(connection, url)
