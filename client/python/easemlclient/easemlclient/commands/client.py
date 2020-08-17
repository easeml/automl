import pluggy  # type: ignore
import os
import argparse
import textwrap
import sys
import yaml
import inspect
from functools import partial

from easemlclient.commands.hookspecs import EasemlPlugin
from easemlclient.model.core import Connection
from easemlclient.commands.action import EasemlAction
from easemlclient.model.type import ApiType

from pkg_resources import working_set
from requests.exceptions import ConnectionError

from typing import Any, Sequence, Dict, Optional, Union, List, Callable, Tuple


def _get_parser() -> argparse.ArgumentParser:
    # LV0/Base Parser
    lv0_parser = argparse.ArgumentParser(
        description=textwrap.dedent('''
        Lightweight python client for ease.ml.
        --------------------------------
        Ease.ml is a declarative machine learning service platform.
        It enables users to upload their datasets and start model selection and tuning jobs.
        Given the schema of the dataset, ease.ml does an automatic search for applicable
        models and performs training, prediction and evaluation. All models are stored as
        Docker images which allows greater portability and reproducibility."
         '''))
    lv0_parser.set_defaults(help=lv0_parser.print_help)
    return lv0_parser


def init_parser(parser: argparse.ArgumentParser, namespace: str, group_description: str = 'Available Commands') -> None:
    if len(list(working_set.iter_entry_points(namespace))):
        subparser = parser.add_subparsers(help=group_description)
    for entry_point in working_set.iter_entry_points(namespace):
        if "hook" not in entry_point.name:
            # load can raise exception due to missing imports or error in
            # object creation
            subcommand = entry_point.load()
            command_parser = subparser.add_parser(
                entry_point.name,
                help=subcommand.help_description(),
                parents=subcommand.register_subparsers())
            action = None
            if not getattr(subcommand.action, '__isnotrunnable__', False):
                action = getattr(subcommand, "action", None)
            if callable(action):
                command_parser.set_defaults(
                    action=subcommand,
                    help=command_parser.print_help,
                    namespace=namespace + "." + entry_point.name)
            else:
                command_parser.set_defaults(
                    help=command_parser.print_help,
                    namespace=namespace + "." + entry_point.name)

            group_description = subcommand.group_description()
            if group_description:
                init_parser(command_parser, namespace + "." +
                            entry_point.name, group_description)
            else:
                init_parser(command_parser, namespace + "." + entry_point.name)


def get_plugin_manager(namespace: str) -> pluggy.PluginManager:
    pm = pluggy.PluginManager("easemlclient")
    pm.add_hookspecs(EasemlPlugin)
    namespace_list = namespace.split(".")

    current_namespace = ""
    for space in namespace_list:
        current_namespace += space + "."
        pm.load_setuptools_entrypoints(current_namespace + "hook.tree")
    pm.load_setuptools_entrypoints(current_namespace + "hook")
    plugin_list = pm.list_name_plugin()
    if len(plugin_list):
        print("Running using the following plugins:",
              [l[0] for l in plugin_list])
    return pm


def _get_easeml_environment(args_keys: Sequence[str], namespace: str) -> dict:
    """Extracts environmental variables
        Two Levels of environmental variables are possible
        General: EASEML_VARIABLE_NAME e.g. export EASEML_API_KEY="AAA" (As viper does in server/golang version)
        Namespaced: EASEMLCLIENT_NA_MES_PACE_VARIABLE_NAME e.g. export EASEMLCLIENT_CREATE_DATASET_API_KEY="AAA"

        Namespaced have priority over General environment variables. General environmental variables
        will be used across actions that require that parameter, Namespaced ones only in the specific namespace

    """
    namespace = namespace.replace(".", "_").upper()

    easeml_env: dict = {}
    for key in args_keys:
        # Read Namespaced environmental variables
        value = os.environ.get(namespace + key.upper())
        if value:
            easeml_env[key] = value
            continue
        # Read General environmental variables
        value = os.environ.get("EASEML_" + key.upper())
        if value:
            easeml_env[key] = value
    return easeml_env


def _get_nested(dct: Dict[str, Any], keys: List[str]) -> Optional[Union[Dict[str, Any], str]]:
    for key in keys:
        try:
            dct = dct[key]
        except KeyError:
            return None
    return dct


def _get_easeml_config_file(fname: str, namespace: str) -> dict:
    """Extracts variables from config file
    Two Levels of config file variables are possible
    api-key: AAA # GENERAL API KEY
    easemlclient:
        create:
            module:
                api-key: BBB # NAMESPACED API KEY

    Namespaced have priority over General variables. General variables
    will be used across actions that require that parameter, Namespaced ones only in the specific namespace

    """
    namespace = namespace.split(".")
    with open(fname) as file:
        easeml_config_file = yaml.load(file, Loader=yaml.SafeLoader)
        easeml_config_full_dict = dict((k.replace('-', '_'), v)
                                       for k, v in easeml_config_file.items())

    easeml_config_dict = {}
    # Populate Generic Variables
    for k, v in easeml_config_full_dict.items():
        if k != namespace[0]:
            easeml_config_dict[k] = v

    # Extract Namespaced Variables
    namespaced_config = _get_nested(easeml_config_full_dict, namespace)
    if namespaced_config:
        namespaced_config = dict((k.replace('-', '_'), v) for k, v in namespaced_config.items())
        easeml_config_dict.update(namespaced_config)

    return easeml_config_dict


def _extract_setup(raw_config: dict, namespace) -> dict:
    config: dict = {key: None for key in raw_config.keys()}

    # Hard coded defaults
    config.update({"host": "http://localhost:8080"})

    # 0 priority default config file
    default_config = os.environ['HOME'] + "/.easeml/config.yaml"
    if os.path.isfile(default_config) and (
            default_config.endswith(".yaml") or default_config.endswith(".yml")):
        config.update(_get_easeml_config_file(default_config, namespace))

    # 1 priority environmental variables
    config.update(_get_easeml_environment(config.keys(), namespace))

    # 2 priority user specified config file from flags
    if raw_config.get('config', None):
        if os.path.isfile(
                raw_config['config']) and (
                raw_config['config'].endswith(".yml") or raw_config['config'].endswith(".yaml")):
            config.update(_get_easeml_config_file(raw_config['config'], namespace))

    # 3 priority user specified flags
    for key, value in raw_config.items():
        if value is not None:
            config[key] = value
    return config


def _filter_config_signature(f: Callable, full_config: dict) -> dict:
    config = {}
    for param in inspect.signature(f).parameters:
        value = full_config.get(param, None)
        config[param] = value
    return config


def _filter_config(action_params: Tuple[List[str], Dict[str, str]], full_config: dict) -> dict:
    config = {}
    for param in action_params[0]:
        value = full_config.get(param, None)
        config[param] = value
    for param, default_value in action_params[1].items():
        value = full_config.get(param, None)
        config[param] = value
    return config


class Client():
    def __init__(self, connection=None):
        self.connection = connection
        self.connection_config = None
        self.available_actions = {}
        self.action_default_params = {}
        self.init_actions()

    def connect(self,
                host: str,
                user_id: Optional[str] = None,
                user_password: Optional[str] = None,
                api_key: Optional[str] = None,
                force: Optional[bool] = False) -> None:

        # If the configuration hasn't changed do nothing
        if self.connection_config != locals() or force:
            self.connection_config = locals()
            self.connection = Connection(host, user_id, user_password, api_key)
            try:
                self.connection.login()
            except ConnectionError as e:
                raise Exception(
                    "Unable to establish a connection to the ease.ml server, please check your input parameters")
            except Exception as e:
                raise e

    def init_actions(self, namespace: str = "easemlclient"):
        for entry_point in working_set.iter_entry_points(namespace):
            if "hook" not in entry_point.name:
                next_namespace = namespace + "." + entry_point.name
                subcommand = entry_point.load()
                action = None
                if not getattr(subcommand.action, '__isnotrunnable__', False):
                    action = getattr(subcommand, "action", None)
                if callable(action):
                    setattr(self, next_namespace.replace('easemlclient.', '').replace(".", "_"),
                            partial(self.run_action, action=subcommand, namespace=next_namespace))

                    self.available_actions[next_namespace] = "(" + ", ".join([str(v) for k, v in
                                                                              inspect.signature(
                                                                                  action).parameters.items() if
                                                                              k != 'connection']) + ")"
                    self.action_default_params[next_namespace] = subcommand.get_config_parameters()
                self.init_actions(next_namespace)

    def print_actions(self):
        if len(self.available_actions):
            for action, parameters in self.available_actions.items():
                print("{}{} \n Parameters: \n \t required: {} \n \t default:{}\n".format(
                    action.replace('easemlclient.', '').replace(".", "_"), parameters,
                    self.action_default_params[action][0], self.action_default_params[action][1]))
        else:
            print("No actions initialized")

    def run_namespace_action(self, raw_config: dict, full_namespace: str) -> Optional[Dict[str, ApiType]]:
        end_node = full_namespace.split(".")[-1]
        namespace = '.'.join(full_namespace.split(".")[:-1])
        for entry_point in working_set.iter_entry_points(namespace):
            if end_node == entry_point.name:
                action = entry_point.load()
                response = self.run_action(raw_config, action, namespace)
                return response
        print("Unable to find action: {}".format(full_namespace))
        return None

    def run_action(self, raw_config: dict, action: EasemlAction, namespace) -> Optional[Dict[str, ApiType]]:
        # Add Hooks
        if namespace:
            pm = get_plugin_manager(namespace)
            action.add_hook(pm.hook)
        config = _extract_setup(raw_config, namespace)
        filtered_config = _filter_config_signature(self.connect, config)
        self.connect(**filtered_config)
        filtered_config = _filter_config(action.get_config_parameters(), config)
        response = action.run_action(filtered_config, self.connection)

        return response


def main() -> None:
    lv0_parser: argparse.ArgumentParser = _get_parser()
    init_parser(lv0_parser, 'easemlclient')

    args = lv0_parser.parse_args()

    if 'action' not in args and 'help' in args:
        args.help()
        sys.exit(1)

    if 'action' in args:
        try:
            cl = Client()
            cl.run_action(vars(args), args.action, args.namespace)
        except Exception as e:
            print(e)
            sys.exit(1)
    else:
        print("Error while parsing arguments")
        sys.exit(1)


if __name__ == "__main__":
    main()
