"""
Implementation of the `TimeInterval` class.
"""
import pyrfc3339

from copy import deepcopy
from datetime import datetime
from typing import Dict, Optional, Any, Iterator, Tuple


class TimeInterval:
    """
    The TimeInterval class contains information about time intervals.

    # Arguments
    
    identifier (str): A unique identifier of the user (i.e. the username).
    name (str): The full name of the user.
    status (str): The current status of the user. Can be 'active' or 'archived'.
    """

    def __init__(self, input: Dict[str, Any]) -> None:
        if "start" not in input:
            raise ValueError("Invalid input dictionary: It must contain a 'start' key.")
        if "end" not in input:
            raise ValueError("Invalid input dictionary: It must contain an 'end' key.")

        self._dict: Dict[str, Any] = deepcopy(input)

    @property
    def start(self) -> Optional[datetime]:
        value = self._dict.get("start")
        return pyrfc3339.parse(value) if value is not None else None

    @property
    def end(self) -> Optional[datetime]:
        value = self._dict.get("end")
        return pyrfc3339.parse(value) if value is not None else None

    def __iter__(self) -> Iterator[Tuple[str, Any]]:
        for (k, v) in self._dict:
            yield (k, v)
