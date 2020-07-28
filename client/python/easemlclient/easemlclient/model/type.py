"""
Implementation of the `ApiType` class.
"""
import requests

from copy import deepcopy
from enum import Enum
from typing import Dict, Any, TypeVar, Generic, Optional, Tuple, List, Type

from .core import Connection

T = TypeVar('T', bound='ApiType')

class ApiType(Generic[T]):
    """The User class contains information about users.
    """

    def __init__(self: T, input: Dict[str, Any]) -> None:
        self._dict: Dict[str, Any] = deepcopy(input)
        self._updates: Dict[str, Any] = {}
        self.T:Type[T] = type(self)

    def _post(self: T, connection: Connection, url: str) -> T:
        resp = requests.post(url, auth=connection.auth, json={**self._dict, **self._updates})
        resp.raise_for_status()
        self._dict['id']=resp.headers['Location'][resp.headers['Location'].rfind('/')+1:]
        return self.T({**self._dict, **self._updates})

    def _patch(self: T, connection: Connection, url: str) -> T:
        resp = requests.patch(url, auth=connection.auth, json=self._updates)
        resp.raise_for_status()
        return self.T({**self._dict, **self._updates})

    def _get(self: T, connection: Connection, url: str) -> T:
        resp = requests.get(url, auth=connection.auth)
        resp.raise_for_status()
        payload = resp.json()
        return self.T(payload["data"])
    
    def _download(self: T, connection: Connection, url: str) -> bytes:
        resp = requests.get(url, auth=connection.auth)
        resp.raise_for_status()
        return resp.content

    @classmethod
    def from_dict(cls, input: Dict[str, Any]) -> 'ApiType':
        """Creates an instance of User given a dictionary.

        Parameters
        ----------
        input : Dict[str, Any]
            The dictionary that can be used to reconstruct a User instance.

        Returns
        -------
        User
            Instance of the reconstructed User class.
        """
        return cls(**input)


Q = TypeVar('Q', bound='ApiQuery')


class ApiQueryOrder(Enum):
    ASC = "asc"
    DESC = "desc"


class ApiQuery(Generic[T, Q]):

    def __init__(self: Q, order_by: Optional[str] = None, order: Optional[ApiQueryOrder] = None,
                 limit: Optional[int] = None, cursor: Optional[str] = None) -> None:
        self._query: Dict[str, Any] = {}
        if order_by is not None:
            order_by = order_by.replace("_", "-")
            self._query["order-by"] = order_by
        if order is not None:
            self._query["order"] = order
        if limit is not None:
            self._query["limit"] = limit
        if cursor is not None:
            self._query["cursor"] = cursor
        self.T:Type[T] = ApiType

    def _run(self: Q, connection: Connection, url: str) -> Tuple[List[T], Optional[Q]]:
        resp = requests.get(url, auth=connection.auth, params=self._query)
        resp.raise_for_status()

        payload = resp.json()
        payload_data = payload["data"] or []

        result = [self.T(x) for x in payload_data]

        next_query: Optional[Q] = None
        next_cursor = payload.get("metadata", {}).get("next-page-cursor", "")
        if next_cursor is not None and next_cursor != "":
            next_query = deepcopy(self)
            next_query._query["cursor"] = next_cursor

        return result, next_query
