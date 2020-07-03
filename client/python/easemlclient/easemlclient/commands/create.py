import sys
import argparse
from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import ModuleQuery, ModuleType, ModuleStatus
from easemlclient.model import Job, JobQuery


class CreateActionGroup(EasemlAction):
    """ Defines the download action group
        Uses the default action (print help)
    """

    def help_description(self):
        return "Creates an item, e.g Dataset, Job"

    def group_description(self):
        return "Available items"


class CreateJobActionGroup(EasemlAction):
    """ Defines the create job action group
        Uses the default action (print help)
    """

    def help_description(self):
        return "Creates a Job"

    def group_description(self):
        return "Available job types"


class CreateDatasetAction(EasemlAction):
    """ Defines the create dataset action
    """

    def help_description(self):
        return "Creates a Dataset"

    def action_flags(self):
        # item single id
        item_subparser = argparse.ArgumentParser(add_help=False)
        item_subparser.add_argument('--id', type=str, help='id', required=True)
        # dataset create
        dataset_subparser = argparse.ArgumentParser(add_help=False)
        dataset_subparser.add_argument(
            '--description', help='Dataset description.', type=str, default="")
        dataset_subparser.add_argument(
            '--name', type=str, help='Dataset name.', default="")
        dataset_subparser.add_argument(
            '--source',
            type=DatasetSource,
            help='Dataset source',
            choices=list(DatasetSource),
            required=True)
        dataset_subparser.add_argument(
            '--source-address',
            type=str,
            help='Dataset source address.',
            required=True)
        dataset_subparser.add_argument(
            '--access-key',
            type=str,
            help='Dataset source address',
            default="")
        return [item_subparser, dataset_subparser]

    def action(self, config, connection):
        response = []
        try:
            dataset = Dataset.create(
                id=config["id"],
                source=config["source"],
                name=config["name"],
                source_address=config["source_address"],
                description=config["description"]).post(connection)
            response.append(dataset)
            print("Dataset with id {} succesfully created".format(dataset.id))
        except Exception as e:
            print("Unable to create dataset")
            if e.response.status_code == 409:
                print("There is an exisiting dataset with the id: {}".format(
                    config["id"]))
            print(e)
        return response


class CreateNewJobAction(EasemlAction):
    """ Defines the create new job from scratch action
    """

    def help_description(self):
        return "Downloads model"

    def action_flags(self):
        # job create
        job_subparser = argparse.ArgumentParser(add_help=False)
        job_subparser.add_argument(
            '--dataset-id',
            type=str,
            help='Dataset ID to be used for the Job.',
            required=True)
        # job create new
        new_job_subparser = argparse.ArgumentParser(add_help=False)
        new_job_subparser.add_argument(
            '--models',
            type=str,
            nargs='+',
            default=[],
            help='Models to apply to the job. If not set denotes all applicable models.')
        new_job_subparser.add_argument(
            '--objective', type=str, help='Job objective.')
        new_job_subparser.add_argument(
            '--accept-new-models',
            action='store_true',
            default=None,
            help='Set to indicate that new models applicable to the job will also be added.')
        new_job_subparser.add_argument(
            '--alt-objectives',
            type=str,
            nargs='+',
            default=[],
            help='Job alternative objectives.')
        new_job_subparser.add_argument(
            '--max-tasks',
            type=int,
            default=1,
            help='Maximum number of tasks to spawn from this job. (default 1)')
        return [job_subparser, new_job_subparser]

    def action(self, config, connection):
        try:
            dataset = Dataset({"id": config["dataset_id"]}).get(connection)
        except Exception as e:
            if e.response.status_code == 404:
                print("There is no dataset with id: {}".format(
                    config["dataset_id"]))
            print(e)
            sys.exit(1)
        if dataset.status != DatasetStatus.VALIDATED:
            print(
                "The dataset with id {} is not ready to be used!".format(
                    dataset.id))
            sys.exit(1)

        try:
            all_models_temp, next_query = ModuleQuery(
                type='model', status='active', schema_in=dataset.schema_in, schema_out=dataset.schema_out).run(connection)
        except Exception as e:
            print(e)
            sys.exit(1)

        if not len(all_models_temp):
            print("No available models to be used!")
            print(len(all_models_temp))
            sys.exit(1)

        def print_available(module_type, module_list):
            print("Available {} are:".format(module_type))
            if len(module_list) == 0:
                print("None")
            for m in module_list:
                print("\t- " + m.name)

        used_models = []
        if config['models']:
            available_models = {}
            for m in all_models_temp:
                available_models[m.name] = m
            for model in config['models']:
                if model in available_models:
                    used_models.append(available_models[model])
                else:
                    print("Model {} not found!".format(model))
                    print_available("Model", all_models_temp)
                    sys.exit(1)
        else:
            used_models = all_models_temp

        try:
            all_objectives, next_query = ModuleQuery(
                type='objective', status='active', schema_in=dataset.schema_out).run(connection)
        except Exception as e:
            print(e)
            sys.exit(1)

        if not len(all_objectives):
            print("No available objectives to be used!")
            sys.exit(1)

        alt_objectives = []
        available_objectives = {}
        for o in all_objectives:
            available_objectives[o.name] = o
        if config['alt_objectives']:
            for objective in config['alt_objectives']:
                if objective in available_objectives:
                    alt_objectives.append(available_objectives[objective])
                else:
                    print("Alt-objective {} not found!".format(objective))
                    print_available("objectives", available_objectives)
                    sys.exit(1)
        else:
            alt_objectives = all_objectives
        print(all_objectives)
        print(config)
        if config['objective']:
            if config['objective'] in available_objectives:
                objective = available_objectives[config['objective']]
            else:
                print("Objective {} not found!".format(config['objective']))
                print_available("objectives", all_objectives)
                sys.exit(1)
        else:
            objective = all_objectives[0]

        try:
            job = Job.create(
                dataset=dataset,
                objective=objective,
                models=used_models,
                alt_objectives=alt_objectives,
                max_tasks=config['max_tasks']).post(connection)
        except Exception as e:
            print(e)
            sys.exit(1)
        print("Job id: {} created sucessfully".format(job.id))
        return [job]


class CreateJobFromTaskAction(EasemlAction):
    """ Defines the create job from task action
    """

    def help_description(self):
        return "Creates a Job from an existing task"

    def action_flags(self):
        # item single id
        item_subparser = argparse.ArgumentParser(add_help=False)
        item_subparser.add_argument('--id', type=str, help='id', required=True)
        # dataset create
        dataset_subparser = argparse.ArgumentParser(add_help=False)
        dataset_subparser.add_argument(
            '--description', help='Dataset description.', type=str, default="")
        dataset_subparser.add_argument(
            '--name', type=str, help='Dataset name.', default="")
        dataset_subparser.add_argument(
            '--source',
            type=DatasetSource,
            help='Dataset source',
            choices=list(DatasetSource),
            required=True)
        dataset_subparser.add_argument(
            '--source-address',
            type=str,
            help='Dataset source address.',
            required=True)
        dataset_subparser.add_argument(
            '--access-key',
            type=str,
            help='Dataset source address',
            default="")
        return [item_subparser, dataset_subparser]

    # TODO
    def action(self, config, connection):
        job = Job()
        return [job]


create_action_group = CreateActionGroup()
create_job_action_group = CreateJobActionGroup()
create_dataset = CreateDatasetAction()
create_new_job = CreateNewJobAction()
create_job_from_task = CreateJobFromTaskAction()
