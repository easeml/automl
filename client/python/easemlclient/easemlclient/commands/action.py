import argparse
import abc
from typing import List, Any, Optional, Dict
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from easemlclient.commands.hookspecs import EasemlPlugin

# From https://stackoverflow.com/questions/44542605/python-how-to-get-all-default-values-from-argparse
def get_argparse_defaults(parser):
    defaults = {}
    for action in parser._actions:
        if not action.required and action.dest != "help":
            defaults[action.dest] = action.default
    return defaults

def get_argparse_required(parser):
    required = []
    for action in parser._actions:
        if action.required:
            required.append(action.dest)
    return required

def non_runnable(funcobj: Any) -> Any:
    """A decorator indicating non runnable action
        This attribute remains unless overridden by the implemented action
    """
    funcobj.__isnotrunnable__ = True
    return funcobj

class EasemlAction(metaclass=abc.ABCMeta):
    """ Base class to define actions

        If the child object doesn't implement the action method, print help is assumed
        Defines base fags and methods that can be overridden by the children
    """

    def __init__(self) -> None:
        self._hook: Optional[EasemlPlugin] = None

    @non_runnable
    def action(self, config: dict, connection: Connection) -> Optional[Dict[str,ApiType]]:  # needs to be implemented in the child objects, else
        pass

    @abc.abstractmethod
    def help_description(self) -> Optional[str]:
        raise NotImplementedError

    def action_flags(self) -> List[argparse.ArgumentParser]:
        return []

    def group_description(self) -> Optional[str]:
        return None

    def register_subparsers(self) -> List[argparse.ArgumentParser]:
        # define common shared arguments
        base_subparser = argparse.ArgumentParser(add_help=False)
        base_subparser.add_argument(
            '--api-key', type=str, help='API key of the user.')
        base_subparser.add_argument(
            '--config', type=str, help='name to be used')
        base_subparser.add_argument(
            '--host', type=str, help='Specify Ease.ml host')

        additional_parsers = self.action_flags()
        additional_parsers.append(base_subparser)
        return additional_parsers
    
    def get_config_parameters(self):
        all_parsers = self.register_subparsers()
        defaults={}
        required=[]
        for parser in all_parsers:
            defaults.update(get_argparse_defaults(parser))
            required=required+get_argparse_required(parser)
        return (required, defaults)

    def add_hook(self, hook: EasemlPlugin) -> None:
        self._hook = hook

    def run_action(self, config: dict, connection: Connection) -> Optional[Dict[str,ApiType]]:

        config = self.pre_action(config, connection)
        response = self.action(config, connection)
        hook_response = self.post_action(config, connection, response)

        return response, hook_response

    def pre_action(self, config: dict, connection: Connection) -> dict:
        if self._hook:
            configs = self._hook.easemlclient_add_pre_action(
                config=config, connection=connection)
            final_config = {}
            for c in configs:
                final_config.update(c)
                return final_config
        return config

    def post_action(self, config: dict, connection: Connection, response: List[ApiType[Any]]) -> Optional[Dict[str,ApiType]]:
        if self._hook:
            response=self._hook.easemlclient_add_post_action(
                config=config, connection=connection, response=response)
        return response
