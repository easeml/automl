'use strict'

import urljoin from 'url-join'
import axios from 'axios'
import datasets from './datasets'
import modules from './modules'
import jobs from './jobs'
import tasks from './tasks'
import users from './users'
import processes from './processes'

const API_PREFIX = 'api/v1'

function loadContext (input) {
  return new Context(input.serverAddress, input.userCredentials)
}

/**
 * Creates a new instance of the client context. The client context holds all information that is needed
 * to communicate with the server (server address and user credentials). All client methods need to be executed
 * on a context instance.
 *
 * @param {string} serverAddress - The URL to the easeml server that is serving the REST API.
 * @param {Object} userCredentials - Object containing either the API key or the username and password.
 * @param {string} userCredentials.apiKey - The API key that is used to authenticate the user.
 * @param {string} userCredentials.username - The username that identifies the user.
 * @param {string} userCredentials.password - The password that is used to authenticate the user.
 * @param {Object} callbacksDict - Dictionary containing callback functions.
 * @param {function} callbacksDict.unauthErrorCallback - Function callback for 401 responses.
 */
function Context (serverAddress, userCredentials, callbacksDict = {
  unauthErrorCallback: function () {}
}) {
  this.serverAddress = serverAddress
  this.userCredentials = userCredentials
  this.baseURL = urljoin(serverAddress, API_PREFIX)

  const axiosConfig = {
    timeout: 1000,
    baseURL: this.baseURL,
    headers: {}
  }
  this.authHeader = {}
  if ('apiKey' in userCredentials) {
    axiosConfig.headers['X-API-KEY'] = userCredentials.apiKey
    this.authHeader['X-API-KEY'] = userCredentials.apiKey
  } else if ('username' in userCredentials && 'password' in userCredentials) {
    axiosConfig.auth = {
      username: userCredentials.username,
      password: userCredentials.password
    }
    this.authHeader.Authorization = 'Basic ' + btoa(userCredentials.username + ':' + userCredentials.password)
  }

  this.userCredentials = userCredentials
  this.axiosInstance = axios.create(axiosConfig)
  // Register callbacks
  if (typeof callbacksDict.unauthErrorCallback === 'function') {
    this.axiosInstance.interceptors.response.use(response => {
      return response
    }, error => {
      if (error.response.status === 401) {
        // place your reentry code
        console.log('# Unauthorized access')
        callbacksDict.unauthErrorCallback()
      }
      return error
    })
  } else {
    console.log('>>>>>>>>>>>>>>> CALLBACK NOT PRESENT', callbacksDict)
  }
}

Context.prototype.getDatasets = datasets.getDatasets
Context.prototype.getDatasetById = datasets.getDatasetById
Context.prototype.validateDatasetFields = datasets.validateDatasetFields
Context.prototype.createDataset = datasets.createDataset
Context.prototype.updateDataset = datasets.updateDataset
Context.prototype.uploadDataset = datasets.uploadDataset
Context.prototype.listDatasetDirectoryByPath = datasets.listDatasetDirectoryByPath
Context.prototype.downloadDatasetByPath = datasets.downloadDatasetByPath

Context.prototype.getModules = modules.getModules
Context.prototype.getModuleById = modules.getModuleById
Context.prototype.validateModuleFields = modules.validateModuleFields
Context.prototype.createModule = modules.createModule
Context.prototype.updateModule = modules.updateModule

Context.prototype.getJobs = jobs.getJobs
Context.prototype.getJobById = jobs.getJobById
Context.prototype.validateJobFields = jobs.validateJobFields
Context.prototype.createJob = jobs.createJob
Context.prototype.updateJob = jobs.updateJob

Context.prototype.getTasks = tasks.getTasks
Context.prototype.getTaskById = tasks.getTaskById
Context.prototype.updateTask = tasks.updateTask
Context.prototype.listTaskPredictionsDirectoryByPath = tasks.listTaskPredictionsDirectoryByPath
Context.prototype.downloadTaskPredictionsByPath = tasks.downloadTaskPredictionsByPath
Context.prototype.downloadTrainedModelAsImage = tasks.downloadTrainedModelAsImage

Context.prototype.getUsers = users.getUsers
Context.prototype.getUserById = users.getUserById
Context.prototype.loginUser = users.loginUser
Context.prototype.logoutUser = users.logoutUser

Context.prototype.getProcesses = processes.getProcesses

export default {
  Context: Context,
  loadContext: loadContext
}
