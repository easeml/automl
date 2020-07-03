import pluggy
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from typing import List


""" Hook Specifications
        Pre hook executed before the main action
        Post hook executed after the main action
"""

hookspec = pluggy.HookspecMarker("easemlclient")


@hookspec
def easemlclient_add_pre_action(config: dict, connection: Connection) -> dict:
    """ Hook running before the action

    :param config: dictionary of parsed configuration to be used by the action
            connection: connection to be used to communicate with the server
    :return: a modified config file
    """
    return config


@hookspec
def easemlclient_add_post_action(
        config: dict,
        connection: Connection,
        response: List[ApiType]) -> None:
    """ Hook running after the action

    :param config: dictionary of parsed configuration used used by the action
            connection: connection used to communicate with the server
            response: List of ApiType objects instantiated after the action
    :return: No return value
    """
    return None
