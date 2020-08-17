
import sys
import argparse
import traceback
from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import ModuleQuery, ModuleType, ModuleStatus, ModuleSource, Module
from easemlclient.model import Job, JobQuery, User
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from requests.exceptions import HTTPError


from typing import List, Dict


class CreateActionGroup(EasemlAction):
    """ Defines the download action group
        Uses the default action (print help)
    """

    def help_description(self) -> str:
        return "Creates an item, e.g Dataset, Job"

    def group_description(self) -> str:
        return "Available items"


class CreateJobActionGroup(EasemlAction):
    """ Defines the create job action group
        Uses the default action (print help)
    """

    def help_description(self) -> str:
        return "Creates a Job"

    def group_description(self) -> str:
        return "Available job types"

class CreateDatasetAction(EasemlAction):
    """ Defines the create dataset action
    """
    def help_description(self) -> str:
        return "Creates a Dataset"

    def action_flags(self) -> List[argparse.ArgumentParser]:
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
            help='Data-source specific accessKey, i.e. oauth token.',
            default="")
        return [item_subparser, dataset_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,Dataset]:
        response = {}
        try:
            if config["source"]==DatasetSource.UPLOAD:
                with open(config["source_address"], "rb") as f:
                    dataset = Dataset.create(
                        id=config["id"],
                        source=config["source"],
                        access_key=config["access_key"],
                        name=config["name"]
                    ).post(connection)
                    dataset.upload(connection=connection, data=f)
                    dataset.status = DatasetStatus.TRANSFERRED
                    dataset.patch(connection)
            else:
                dataset = Dataset.create(
                    id=config["id"],
                    source=config["source"],
                    name=config["name"],
                    source_address=config["source_address"],
                    access_key=config["access_key"],
                    description=config["description"]
                ).post(connection)

            response["response"]=dataset
            print("Dataset with id {} successfully created".format(dataset.id))
        except HTTPError as e:
            if e.response.status_code == 409:
                print("There is an existing dataset with the id: {}".format(
                    config["id"]))
            print(e)
        except Exception as e:
            print("Unable to create Dataset")
            print(e)
        print(response)
        return response

class CreateModuleAction(EasemlAction):
    """ Defines the create module action
    """
    def help_description(self) -> str:
        return "Creates a Module"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # item single id
        item_subparser = argparse.ArgumentParser(add_help=False)
        item_subparser.add_argument('--id', type=str, help='id', required=True)
        # dataset create
        module_subparser = argparse.ArgumentParser(add_help=False)
        module_subparser.add_argument(
            '--description', help='Module description.', type=str, default="")
        module_subparser.add_argument(
            '--name', type=str, help='Module name.', default="")
        module_subparser.add_argument(
            '--source',
            type=ModuleSource,
            help='Module source',
            choices=list(ModuleSource),
            required=True)
        module_subparser.add_argument(
            '--source-address',
            type=str,
            help='Module source address.',
            required=True)
        module_subparser.add_argument(
            '--access-key',
            type=str,
            help='Module-source specific accessKey, i.e. oauth token.',
            default="")
        module_subparser.add_argument(
            '--type',
            type=ModuleType,
            help='Module type',
            choices=list(ModuleType),
            required=True)
        return [item_subparser, module_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,Module]:

        response = {}
        try:
            if config["source"] == ModuleSource.UPLOAD:
                if config["source_address"].endswith(".tar"):
                    with open(config["source_address"], "rb") as f:
                        module = Module.create(
                            id=config["id"],
                            source=config["source"],
                            type=config["type"],
                            name=config["name"],
                            access_key=config["access_key"],
                        ).post(connection)
                        module.upload(connection=connection, data=f)
                        module.status = DatasetStatus.TRANSFERRED
                        module.patch(connection)
                else:
                    msg = "Module image needs to be provided as .tar file for upload"
                    print(msg)
                    raise Exception(msg)
            else:
                module = Module.create(
                    id=config["id"],
                    source=config["source"],
                    name=config["name"],
                    type=config["type"],
                    source_address=config["source_address"],
                    access_key=config["access_key"],
                    description=config["description"]
                ).post(connection)

            response["response"]=module
            print("Module with id {} successfully created".format(module.id))
        except HTTPError as e:
            if e.response.status_code == 409:
                print("There is an existing Module with the id: {}".format(
                    config["id"]))
            print(e)
        except Exception as e:
            print(e)
        return response

class CreateNewJobAction(EasemlAction):
    """ Defines the create new job from scratch action
    """

    def help_description(self) -> str:
        return "Downloads model"

    def action_flags(self) -> List[argparse.ArgumentParser]:
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
        new_job_subparser.add_argument(
            '--pipeline',
            type=str,
            nargs='+',
            default=[],
            help='Sequence of operations to be applied to the data')

        group = new_job_subparser.add_mutually_exclusive_group()
        group.add_argument(
            '--models',
            type=str,
            nargs='+',
            default=['all'],
            help='Models to apply to the job. If not set denotes all applicable models.')
        group.add_argument(
            '--tasks',
            type=str,
            nargs='+',
            default=[],
            help='Tasks ids to run on the dataset.')

        return [job_subparser, new_job_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,ApiType]:
        response={}
        # TODO handle the mutual exclusion
        try:
            dataset = Dataset({"id": config["dataset_id"]}).get(connection)
        except HTTPError as e:
            if e.response.status_code == 404:
                print("There is no dataset with id: {}".format(
                    config["dataset_id"]))
            print(e)
            raise e
        except Exception as e:
            print(e)
            raise e
        if dataset.status != DatasetStatus.VALIDATED:
            print(
                "The dataset with id {} is not ready to be used!".format(
                    dataset.id))
        response['dataset']=dataset

        def print_available(module_type, module_list):
            print("Available {} are:".format(module_type))
            if len(module_list) == 0:
                print("None")
            for m in module_list:
                print("\t- " + m.name)
        if config['pipeline']:
            pipeline = config['pipeline']
        else:
            if config['tasks']:
                pipeline = ['predict', 'evaluate']
            else:
                pipeline = ['train', 'predict', 'evaluate']

        used_models = []
        if not config['tasks']:
            try:
                all_models_temp, next_query = ModuleQuery(
                    module_type='model', status='active', schema_in=dataset.schema_in,
                    schema_out=dataset.schema_out).run(
                    connection)
            except Exception as e:
                print(e)
                raise e

            if not len(all_models_temp):
                print("No available models to be used!")
                print(len(all_models_temp))
                raise Exception("No available models to be used!")
            if config['models']:
                available_models = {}
                for m in all_models_temp:
                    available_models[m.name] = m
                for model in config['models']:
                    if model == 'all':
                        used_models = all_models_temp
                        break

                    if model in available_models:
                        used_models.append(available_models[model])
                    else:
                        print("Model {} not found!".format(model))
                        print_available("Model", all_models_temp)
                        raise Exception("Model {} not found!".format(model))
            else:
                used_models = all_models_temp
        for idx, models in enumerate(used_models):
            response["models_"+str(idx)]=models
            
        try:
            all_objectives, next_query = ModuleQuery(
                module_type='objective', status='active', schema_in=dataset.schema_out).run(connection)
        except Exception as e:
            print(e)
            raise e

        if not len(all_objectives):
            raise Exception("No available objectives to be used!")

        for idx, objective in enumerate(all_objectives):
            response["objective_"+str(idx)]=objective                
            
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
                    raise Exception("Alt-objective {} not found!".format(objective))
        else:
            alt_objectives = all_objectives
        if config['objective']:
            if config['objective'] in available_objectives:
                objective = available_objectives[config['objective']]
            else:
                print("Objective {} not found!".format(config['objective']))
                print_available("objectives", all_objectives)
                raise Exception("Objective {} not found!".format(config['objective']))
        else:
            objective = all_objectives[0]

        try:
            job = Job.create(
                dataset=dataset,
                task_ids=config['tasks'],
                models=used_models,
                objective=objective,
                alt_objectives=alt_objectives,
                max_tasks=config['max_tasks'],
                pipeline=pipeline,
            ).post(connection)
        except Exception as e:
            print(e)
            raise e
        print("Job id: {} created sucessfully".format(job.id))
        response["response"]=job 
        return response


create_action_group = CreateActionGroup()
create_module = CreateModuleAction()
create_dataset = CreateDatasetAction()
create_new_job = CreateNewJobAction()
