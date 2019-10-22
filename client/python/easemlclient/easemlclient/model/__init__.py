"""Types that are defined in the ease.ml REST API.
"""
from .common import TimeInterval
from .core import Connection
from .user import User, UserQuery, UserStatus
from .process import Process, ProcessQuery, ProcType, ProcStatus
from .dataset import Dataset, DatasetQuery, DatasetSource, DatasetStatus
from .module import Module, ModuleQuery, ModuleType, ModuleSource, ModuleStatus
from .job import Job, JobQuery, JobStatus
from .task import Task, TaskQuery, TaskStatus, TaskStage, TaskStageIntervals, TaskStageDurations
