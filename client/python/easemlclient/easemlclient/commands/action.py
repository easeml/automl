import argparse

from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType


class EasemlAction:
    """ Base class to define actions

        If the child object doesn't implement the action method, print help is assumed
        Defines base fags and methods that can be overridden by the children
    """

    def __init__(self):
        self._hook = None

    # def action(self): needs to be implemented in the child objects, else
    # print help is assumed

    def action_flags(self):
        return []

    def help_description(self):
        raise NotImplementedError

    def group_description(self):
        return None

    def register_subparsers(self):
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

    def add_hook(self, hook):
        self._hook = hook

    def run_action(self, config: dict, connection: Connection):
        config = self.pre_action(config, connection)
        response = self.action(config, connection)
        self.post_action(config, connection, response)

    def pre_action(self, config: dict, connection: Connection):
        if self._hook:
            configs = self._hook.easemlclient_add_pre_action(
                config=config, connection=connection)
            final_config = {}
            for c in configs:
                final_config.update(c)
                return final_config
        return config

    def post_action(
            self,
            config: dict,
            connection: Connection,
            response: ApiType):
        if self._hook:
            self._hook.easemlclient_add_post_action(
                config=config, connection=connection, response=response)
