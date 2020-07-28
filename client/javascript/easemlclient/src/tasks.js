'use strict'

// let common = require("./common");
import common from './common'
import decamelizeKeys from 'decamelize-keys'

import moment from 'moment'
const momentDurationFormatSetup = require('moment-duration-format')
momentDurationFormatSetup(moment)

function transformDataItem (input) {
  const runningDuration = moment.duration(input['running-duration'], 'milliseconds')

  return {
    id: input.id,
    intId: parseInt(input.id.split('/')[1]),
    job: input.job,
    process: input.process,
    user: input.user,
    dataset: input.dataset,
    model: input.model,
    objective: input.objective,
    altObjectives: input['alt-objectives'],
    config: input.config,
    quality: input.quality,
    qualityTrain: input['quality-train'],
    qualityExpected: input['quality-expected'],
    altQualities: input['alt-qualities'],
    status: input.status,
    statusMessage: input['status-message'],
    stage: input.stage,
    runningDuration: runningDuration,
    runningDurationString: runningDuration.format()
  }
}

function getTasks (query) {
  // This allows us to accept camel case keys.
  query = decamelizeKeys(query || {}, '-')

  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    common.runGetQuery(this.axiosInstance, '/tasks', query)
      .then(data => {
        const items = []

        if (data) {
          for (let i = 0; i < data.length; i++) {
            items.push(transformDataItem(data[i]))
          }
        }

        resolve(items)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function getTaskById (id) {
  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.get('/tasks/' + id)
      .then(response => {
        const result = transformDataItem(response.data.data)
        resolve(result)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function updateTask (id, updates) {
  // Collect fields of interest.
  updates = decamelizeKeys(updates, '-')
  const data = {}
  if ('status' in updates) {
    data.status = updates.status
  }

  // Run patch request as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.patch('/tasks/' + id, data)
      .then(result => {
        resolve()
      })
      .catch(e => {
        reject(e)
      })
  })
}

function listTaskPredictionsDirectoryByPath (id, relPath) {
  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.get('/tasks/' + id + '/predictions' + relPath)
      .then(response => {
        const result = response.data
        resolve(result)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function downloadTaskPredictionsByPath (id, relPath, inBrowser = false) {
  if (inBrowser) {
    const url = this.axiosInstance.defaults.baseURL + '/tasks/' + id + '/predictions' + relPath
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', id + '.tar') // or any other extension
    document.body.appendChild(link)
    link.click()
  } else {
    // Run query and collect results as a promise. The result passed to the promise is a Blob.
    return new Promise((resolve, reject) => {
      this.axiosInstance.get('/tasks/' + id + '/predictions' + relPath)
        .then(response => {
          const contentType = response.headers['Content-Type'] || ''
          const result = new Blob([response.data], { type: contentType })
          resolve(result)
        })
        .catch(e => {
          reject(e)
        })
    })
  }
}

function downloadTrainedModelAsImage (id, inBrowser = false) {
  if (inBrowser) {
    const url = this.axiosInstance.defaults.baseURL + '/tasks/' + id + '/image/download' + '?api-key=' + this.userCredentials.apiKey
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', id + '.tar') // or any other extension
    document.body.appendChild(link)
    link.click()
  } else {
    // Run query and collect results as a promise. The result passed to the promise is a Blob.
    return new Promise((resolve, reject) => {
      this.axiosInstance.get('/tasks/' + id + '/image/download')
        .then(response => {
          const contentType = response.headers['Content-Type'] || ''
          const result = new Blob([response.data], { type: contentType })
          resolve(result)
        })
        .catch(e => {
          reject(e)
        })
    })
  }
}

export default {
  getTasks: getTasks,
  getTaskById: getTaskById,
  updateTask: updateTask,
  listTaskPredictionsDirectoryByPath: listTaskPredictionsDirectoryByPath,
  downloadTaskPredictionsByPath: downloadTaskPredictionsByPath,
  downloadTrainedModelAsImage: downloadTrainedModelAsImage
}
