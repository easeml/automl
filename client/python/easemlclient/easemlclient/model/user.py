"""
Implementation of the `User` class.
"""
import requests

from copy import deepcopy
from enum import Enum
from typing import Dict, Optional, Any, Iterator, Tuple, List

from .core import Connection
from .type import ApiType, ApiQuery, ApiQueryOrder


class UserStatus(Enum):
    ACTIVE = "active"
    ARCHIVED = "archived"


class User(ApiType['User']):
    """The User class contains information about users.

    ...
    Attributes:
    -----------
    id: str
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
    def create(cls, id: str, name: Optional[str] = None, password_hash: Optional[str] = None) -> 'User':
        init_dict: Dict[str, Any] = {"id": id}
        if name is not None:
            init_dict["name"] = name
        if password_hash is not None:
            init_dict["password"] = password_hash
        return User(init_dict)
    
    @classmethod
    def create_ref(cls, id: str) -> 'User':
        return User({"id": id})

    @property
    def id(self) -> str:
        return self._dict["id"]

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
    def status(self) -> Optional[UserStatus]:
        value = self._updates.get("status") or self._dict.get("status")
        return UserStatus(value) if value is not None else None

    @status.setter
    def status(self, value: Optional[UserStatus] = None) -> None:
        if value is not None:
            self._updates["status"] = value
        else:
            self._updates.pop("status")

    @property
    def password_hash(self) -> Optional[str]:
        value = self._updates.get("password") or self._dict.get("password")
        return str(value) if value is not None else None

    @password_hash.setter
    def password_hash(self, value: Optional[str] = None) -> None:
        if value is not None:
            self._updates["password"] = value
        else:
            self._updates.pop("password")

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)

    def post(self, connection: Connection) -> 'User':
        url = connection.url("users")
        return self._post(connection, url)

    def patch(self, connection: Connection) -> 'User':
        url = connection.url("users/" + self.id)
        return self._patch(connection, url)

    def get(self, connection: Connection) -> 'User':
        url = connection.url("users/" + self.id)
        return self._get(connection, url)


class UserQuery(ApiQuery['User', 'UserQuery']):

    VALID_SORTING_FIELDS = ["id", "name", "status"]

    def __init__(self, id: Optional[List[str]] = None, status: Optional[UserStatus] = None,
                 order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        super().__init__(order_by, order, limit, cursor)
        self.T = User

        if id is not None:
            self._query["id"] = id
        if status is not None:
            self._query["status"] = status.value

    def run(self, connection: Connection) -> Tuple[List[User], Optional['UserQuery']]:
        url = connection.url("users")
        return self._run(connection, url)
