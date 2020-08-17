import argparse
import sys

from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import Job, JobQuery
from easemlclient.model import Task, TaskQuery
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType


from requests.exceptions import HTTPError
from typing import List, Optional, Dict


class ShowActionGroup(EasemlAction):
    """ Defines the download action group
        Uses the default action (print help)
    """

    def help_description(self) -> str:
        return "Shows an item"

    def group_description(self) -> str:
        return "Available items to show"


class ShowDatasetAction(EasemlAction):
    """ Defines the show dataset action
    """

    def help_description(self) -> str:
        return "Shows dataset(s)"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        optitem_subparser.add_argument(
            '--id', type=str, help='id, if not set shows all available items.')
        return [optitem_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,Dataset]:
        all_datasets = []
        if 'id' in config and config['id']:
            try:
                dataset = Dataset({"id": config["id"]}).get(connection)
                all_datasets.append(dataset)
            except HTTPError as e:
                if e.response.status_code == 404:
                    print("There is no dataset with id: {}".format(config["id"]))
                print(e)
                sys.exit(1)
            except Exception as e:
                print(e)
                sys.exit(1)
            print("Dataset ID: {} : Status: {}".format(
                dataset.id, dataset.status))
        else:
            all_datasets, next_query = DatasetQuery().run(connection)
            for data in all_datasets:
                print("Dataset ID: {} : Status:{}".format(data.id, data.status))
                print(data._dict)
            if len(all_datasets) == 0:
                print("No datasets found")
                print(all_datasets)
        response = {}
        for idx, dataset in enumerate(all_datasets):
            if idx==0:
                response['response']=dataset
            else:
                response['response_'+idx-1]=dataset
        return response


class ShowJobAction(EasemlAction):
    """ Defines the show job action
    """

    def help_description(self) -> str:
        return "Shows Job(s)"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        optitem_subparser.add_argument(
            '--id', type=str, help='id, if not set shows all available items.')
        return [optitem_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,Job]:
        print(config)
        all_jobs: List[Job] = []
        if config['id']:
            try:
                job = Job({'id': config['id']}).get(connection)
                all_jobs.append(job)
            except HTTPError as e:
                if e.response.status_code == 404:
                    print("There is no job with id: {}".format(config["id"]))
                print(e)
                sys.exit(1)
            except Exception as e:
                print(e)
                sys.exit(1)
            print("Job ID: {} : Status:{}".format(job.id, job.status))
        else:
            all_jobs = []
            query = JobQuery()
            next_result: List[Job]
            next_query: Optional[JobQuery] = query
            while next_query is not None:
                next_result, next_query = next_query.run(connection)
                all_jobs.extend(next_result)
            for j in all_jobs:
                print(j.id, j.status)

            if not len(all_jobs):
                print("No jobs found")
        
        response = {}
        for idx, job in enumerate(all_jobs):
            if idx==0:
                response['response']=job
            else:
                response['response_'+idx-1]=job
        return response


class ShowTaskAction(EasemlAction):
    """ Defines the show job action
    """

    def help_description(self) -> str:
        return "Shows Task(s)"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        group = optitem_subparser.add_mutually_exclusive_group()
        group.add_argument('--task-id', type=str, help='Show a specific task')
        group.add_argument('--job-id', type=str,
                           help='Show tasks for a job-id')
        optitem_subparser.add_argument(
            '--best',
            action='store_true',
            default=None,
            help='Shows only the best task.')
        return [optitem_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str,Task]:
        all_tasks = []
        if config['task_id']:
            try:
                task = Task({'id': config['task_id']}).get(connection)
                all_tasks.append(task)
            except HTTPError as e:
                if e.response.status_code == 404:
                    print("There task in job id: {}".format(config["task_id"]))
                print(e)
                exit(1)
            except Exception as e:
                print(e)
                sys.exit(1)
        else:
            if config['job_id']:
                job = Job({'id': config['job_id']})
                all_tasks, query = [], TaskQuery(
                    job=job, order_by="quality", order='desc')
            else:
                all_tasks, query = [], TaskQuery(
                    order_by="quality", order='desc')

            next_result: List[Task]
            next_query: Optional[TaskQuery] = query
            while next_query is not None:
                next_result, next_query = next_query.run(connection)
                all_tasks.extend(next_result)

        if len(all_tasks) == 0:
            print("No tasks found")
        else:
            if config['best']:
                all_tasks = [all_tasks[0]]

        for t in all_tasks:
            print("Task ID: {}, Quality: {}, Status:{}".format(
                    t.id, t.quality, t.status))
        
        response = {}
        for idx, taks in enumerate(all_tasks):
            if idx==0:
                response['response']=task
            else:
                response['response_'+idx-1]=task
        return response



show_action_group = ShowActionGroup()
show_dataset = ShowDatasetAction()
show_job = ShowJobAction()
show_task = ShowTaskAction()
