import argparse
import sys
from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import Task, TaskQuery, Job
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from easemlclient.commands.client import Client

from typing import List, Dict
import json


def _load_config(arg: str) -> dict:
    args_dict = json.loads(arg)
    return args_dict


class ExecuteBatchAction(EasemlAction):
    """ Defines the download model action
    """

    def help_description(self) -> str:
        return "Initializes an easeml server by exectuting a batch of commands"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # Task id or Job id
        opttaskjob_subparser = argparse.ArgumentParser(add_help=False)
        opttaskjob_subparser.add_argument(
            '--actions',
            nargs='+',
            type=_load_config,
            help='List of strings representing dict[dict], outer key is the action namespace, ' + \
                 'inner dict is the action config ')
        return [opttaskjob_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str, ApiType]:
        response = {}
        if config.get('actions', None) and len(config['actions']):
            cl = Client(connection)
            for batch_action in config['actions']:
                for k, sub_config in batch_action.items():
                    print("Executing {}, with config {}".format(k, sub_config))
                    easeml_config = dict((k.replace('-', '_'), v) for k, v in sub_config.items())
                    # Propagate defaults
                    default_keys = ["api_key", "host", "config"]
                    for dk in default_keys:
                        if not easeml_config.get(dk, None) and config.get(dk, None):
                            easeml_config[dk] = config.get(dk, None)
                    resp = cl.run_namespace_action(easeml_config, k)
                    #print("Response: {}".format(resp))
                    for key, r in resp[0].items():
                        response[k+"_"+key] = r
                    # TODO Register hook responses too
        else:
            print("Please specify the actions to be executed")
        return response


easeml_init = ExecuteBatchAction()
