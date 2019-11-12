from setuptools import setup, find_packages, Command
from setuptools.command.sdist import sdist
from setuptools.command.build_py import build_py
from setuptools.command.egg_info import egg_info

from subprocess import check_call
from codecs import open
from os.path import join as pjoin
import distutils
import os
import glob
import sys
import copy


from distutils import log

here = os.path.dirname(os.path.abspath(__file__))
is_repo = os.path.exists(pjoin(here, '.git'))
default_included_files=[('etc/jupyter/jupyter_notebook_config.d', ['config/jupyterlab_easeml.json'])]

log.info('setup.py entered')
log.info('$PATH=%s' % os.environ['PATH'])

with open(os.path.join(here, 'README.md'), encoding='utf-8') as f:
    long_description = f.read()

with open(os.path.join(here, 'requirements.txt'), encoding='utf-8') as f:
    requires = f.read().split()

def install_additional_NPM(command, strict=False):
    """decorator for and packing additional NPM packages"""
    class DecoratedCommand(command):
        def run(self):
            self.includeNPMproject('npm_labextension',"js_extension","lib",["pack"],'share/jupyter/lab/extensions',["js_extension/*.tgz"])
            self.includeNPMproject('npm_webui',"../../web/","jupyterlab_easeml/web",["run","build"],'share/jupyter/lab/extensions',["../../web/dist"])
            command.run(self)
            update_package_data(self.distribution)
        def includeNPMproject(self,cmd_name,rel_proj_path,temp_proj_path,npm_command,output_path,target):
            jsdeps = self.distribution.get_command_obj(cmd_name)
            jsdeps.set_npm_include_options(rel_proj_path,temp_proj_path,npm_command,output_path,target)
            if not is_repo and os.path.exists(temp_proj_path):
                # sdist, nothing to do
                log.info("# Nothing to do.. no targets provided")
                command.run(self)
                return
            try:
                self.distribution.run_command(cmd_name)

            except Exception as e:
                missing = [t for t in jsdeps.targets if not os.path.exists(t)]
                if strict or missing:
                    log.warn('Rebuilding NPM project failed')
                    if missing:
                        log.error('missing files: %s' % missing)
                    raise e
                else:
                    log.warn('Rebuilding NPM project failed(not a problem)')
                    log.warn(str(e))
    return DecoratedCommand


class NPM(Command):
    def __init__(self, dist, **kw):
        super().__init__(dist, **kw)

    def set_npm_include_options(self,rel_proj_path,temp_proj_path,npm_command,output_path,targets):
        self.temp_proj_path=temp_proj_path
        self.npm_command=npm_command
        self.rel_proj_path=rel_proj_path
        self.node_root = pjoin(here,rel_proj_path)
        self.npm_path = os.pathsep.join([
            pjoin(self.node_root, 'node_modules', '.bin'),
            os.environ.get('PATH', os.defpath),
        ])

        self.description = 'install package.json dependencies using npm'
        self.user_options = []
        self.node_modules = pjoin(self.node_root, 'node_modules')
        self.output_path=output_path

        self.targets = targets

    def initialize_options(self):
        pass

    def finalize_options(self):
        pass

    def has_npm(self):
        try:
            check_call(['npm', '--version'])
            return True
        except Exception:
            return False

    def should_run_npm_install(self):
        node_modules_exists = os.path.exists(self.node_modules)
        return self.has_npm() and not node_modules_exists

    def should_run_npm_command(self):
        return self.has_npm() and self.npm_command

    def run(self):
        log.info("%%%%%%%%%%%%% HERE")
        has_npm = self.has_npm()
        if not has_npm:
            log.error("`npm` unavailable.  If you're running this command using sudo, make sure `npm` is available to sudo")

        if os.path.exists(self.temp_proj_path):
            msg="ERROR: Using an exisiting folder to hold temporary build data"
            log.error(msg)
            #raise Exception(msg)
        else:
            distutils.dir_util.mkpath(self.temp_proj_path)


        env = os.environ.copy()
        env['PATH'] = self.npm_path

        if self.should_run_npm_install():
            log.info("Installing build dependencies with npm.  This may take a while...")
            check_call(['npm', 'install'], cwd=self.node_root, stdout=sys.stdout, stderr=sys.stderr)
            os.utime(self.node_modules, None)

        if self.should_run_npm_command():
            log.info("Running "+str(['npm']+self.npm_command))
            log.info("Node root "+self.node_root)
            log.info("WD "+pjoin(here,self.rel_proj_path))
            check_call(['npm']+self.npm_command, cwd=pjoin(here,self.rel_proj_path), stdout=sys.stdout, stderr=sys.stderr)

        tmp=[]
        log.info("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@## TARGETS "+str(self.targets))
        for t in self.targets:
            outside=[os.path.relpath(f, '.') for f in glob.glob(t)]
            log.info("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@## OUTSIDE "+str(outside))
            for f in outside:
                log.info("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@## f "+str(f))
                if os.path.isdir(f):
                    distutils.dir_util.copy_tree(f,self.temp_proj_path)
                    for root, dirs, files in os.walk(self.temp_proj_path, topdown = False):
                        for name in files:
                            tmp.append(os.path.join(root, name))
                else:
                    distutils.file_util.copy_file(f,os.path.join(self.temp_proj_path,os.path.basename(f)))
                    tmp.append(os.path.join(self.temp_proj_path,os.path.basename(f)))
        self.targets=tmp

        print("$$$ TARGETS ",self.targets)
        for t in self.targets:
            if not os.path.exists(t):
                msg = 'Missing file: %s' % t
                if not has_npm:
                    msg += '\nNPM is required to build the extension'
                raise ValueError(msg)

        self.distribution.data_files+=[(self.output_path, self.targets)]
        log.info("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@## INCLUDED DATA FILES "+str(self.distribution.data_files))

        # update package data in case this created new files
        update_package_data(self.distribution)

def get_data_files(output_files_args):
    """Get the data files for the package.
    """
    retval=[]
    for arg in output_files_args:
        output_path=arg[0]
        files=arg[1]
        tmp=[]
        for f in files:
            if os.path.isdir(f):
                for root, dirs, files in os.walk(f, topdown = False):
                    for name in files:
                        tmp.append(os.path.join(root, name))
            else:
                tmp.append(f)
        retval+=[(output_path,copy.deepcopy(tmp))]
    log.info("\n\n\n\n\n########### RETURN VALUE "+str(retval)+"\n\n\n\n\n####")
    return retval

def update_package_data(distribution):
    """update package_data to catch changes during setup"""
    build_py = distribution.get_command_obj('build_py')
    # distribution.package_data = find_package_data()
    # re-init build_py options which load package_data
    build_py.finalize_options()

name = 'jupyterlab_easeml'
here = os.path.abspath(os.path.dirname(__file__))

setup(
     name='jupyterlab_easeml',
     version='0.0.1',
     author="DS3 lab",
     author_email="easeml@ease.ml",
     description="A Docker and AWS utility package",
     long_description=long_description,
     long_description_content_type="Jupyter lab easeml extension", #long_description,
     url="https://github.com/DS3Lab/easeml",
     packages=find_packages(exclude=['tests']),
    include_package_data=True,
     install_requires=requires,
     keywords=[
        'jupyter',
        'widgets',
     ],
     classifiers=[
         'Development Status :: 4 - Beta',
         'Framework :: IPython',
         'Intended Audience :: Developers',
         'Intended Audience :: Science/Research',
         "Topic :: Scientific/Engineering",
         "Topic :: Scientific/Engineering :: Information Analysis",
         "Programming Language :: Python :: 3",
         "Programming Language :: JavaScript",
         "Programming Language :: Python :: 3",
         "License :: OSI Approved :: MIT License",
         "Operating System :: OS Independent",
     ],
    zip_safe=False,
    cmdclass={
        'build_py': install_additional_NPM(build_py),
        'egg_info': install_additional_NPM(egg_info),
        'sdist': install_additional_NPM(sdist, strict=True),
        'npm_labextension': NPM,
        'npm_webui': NPM
    },
    data_files=get_data_files(default_included_files+[('share/jupyter/lab/extensions',["lib"])])
 )
