import pluggy  # type: ignore
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from typing import List, TypeVar, Callable, Any, Dict, cast, Optional


""" Hook Specifications
        Pre hook executed before the main action
        Post hook executed after the main action
"""

# Typed as per: https://stackoverflow.com/questions/54674679/how-can-i-annotate-types-for-a-pluggy-hook-specification
# hookspec = pluggy.HookspecMarker("easemlclient")

# Improvement suggested by @oremanj on python/typing gitter
F = TypeVar("F", bound=Callable[..., Any])
hookspec = cast(Callable[[F], F], pluggy.HookspecMarker("easemlclient"))

class EasemlPlugin:
    @staticmethod
    @hookspec
    def easemlclient_add_pre_action(config: dict, connection: Connection) -> dict:
        """ Hook running before the action

        :param config: dictionary of parsed configuration to be used by the action
                connection: connection to be used to communicate with the server
        :return: a modified config file
        """
        return config

    @staticmethod
    @hookspec
    def easemlclient_add_post_action(
            config: dict,
            connection: Connection,
            response: Dict[str,ApiType]) -> Optional[Dict[str,ApiType]]:
        """ Hook running after the action

        :param config: dictionary of parsed configuration used used by the action
                connection: connection used to communicate with the server
                response: List of ApiType objects instantiated after the action
        :return: No return value
        """
        return response
