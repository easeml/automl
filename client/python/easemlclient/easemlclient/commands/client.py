import pluggy
import os
import argparse
import textwrap
import sys
import yaml

from easemlclient.commands import hookspecs
from easemlclient.model.core import Connection
from pkg_resources import working_set


def _get_parser():
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


def init_parser(parser, namespace, group_description='Available Commands'):
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


def get_plugin_manager(namespace):
    pm = pluggy.PluginManager("easemlclient")
    pm.add_hookspecs(hookspecs)
    pm.load_setuptools_entrypoints(namespace + ".hook")
    plugin_list = pm.list_name_plugin()
    if len(plugin_list):
        print("Running using the following plugins:",
              [l[0] for l in plugin_list])
    return pm


def _get_easeml_environment():
    easeml_env = {}
    # Populate dictionary
    return easeml_env


def _get_easeml_config_file(fname):
    with open(fname) as file:
        easeml_config_file = yaml.load(file, Loader=yaml.SafeLoader)
        easeml_config_file = dict((k.replace('-', '_'), v)
                                  for k, v in easeml_config_file.items())
    return easeml_config_file


def _extract_setup(args):
    config = {arg: None for arg in vars(args)}

    # Hard coded defaults
    config.update({"host": "http://localhost:8080"})

    # 0 priority default config file
    default_config = os.environ['HOME'] + "/.easeml/config.yaml"
    if os.path.isfile(default_config) and (
            default_config.endswith(".yaml") or default_config.endswith(".yml")):
        config.update(_get_easeml_config_file(default_config))

    # 1 priority environmental variables
    config.update(_get_easeml_environment())

    # 2 priority user specified config file from flags
    if args.config:
        if os.path.isfile(
                args.config) and (
                args.config.endswith(".yml") or args.config.endswith(".yaml")):
            config.update(_get_easeml_config_file(args.config))

    # 3 priority user specified flags
    for arg in vars(args):
        value = getattr(args, arg)
        if value is not None:
            config[arg] = value
        # print(arg,value)

    return config


def main():

    lv0_parser = _get_parser()
    init_parser(lv0_parser, 'easemlclient')

    args = lv0_parser.parse_args()

    if 'action' not in args and 'help' in args:
        args.help()
        sys.exit(1)

    if 'action' in args:
        # Add Hooks
        pm = get_plugin_manager(args.namespace)
        args.action.add_hook(pm.hook)
        config = _extract_setup(args)
        connection = Connection(host=config["host"], api_key=config["api_key"])
        connection.login()
        args.action.run_action(config, connection)
    else:
        print("Error while parsing arguments")
        sys.exit(1)


if __name__ == "__main__":
    main()
