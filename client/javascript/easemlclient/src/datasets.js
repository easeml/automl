'use strict'

// let common = require("./common");
import urljoin from 'url-join'
import common from './common'
import decamelizeKeys from 'decamelize-keys'
import tus from 'tus-js-client'

function transformDataItem (input) {
  return {
    id: input.id,
    name: input.name,
    description: input.description,
    user: input.user,
    source: input.source,
    schemaIn: input['schema-in'],
    schemaOut: input['schema-out'],
    sourceAddress: input['source-address'],
    creationTime: new Date(input['creation-time']),
    status: input.status
  }
}

function getDatasets (query) {
  // This allows us to accept camel case keys.
  query = decamelizeKeys(query || {}, '-')

  // Reformat schema fields.
  if ('schema-in' in query && typeof (query['schema-in']) !== 'string') {
    query['schema-in'] = JSON.stringify(query['schema-in'])
  }
  if ('schema-out' in query && typeof (query['schema-out']) !== 'string') {
    query['schema-out'] = JSON.stringify(query['schema-out'])
  }

  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    common.runGetQuery(this.axiosInstance, '/datasets', query)
      .then(data => {
        let items = []

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

function getDatasetById (id) {
  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.get('/datasets/' + id)
      .then(response => {
        let result = transformDataItem(response.data.data)
        resolve(result)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function validateDatasetFields (input) {
  input = decamelizeKeys(input, '-')
  let errors = {}

  if (!input.id) {
    errors['id'] = 'The id must be specified.'
  } else if (!common.ID_FORMAT.test(input.id)) {
    errors['id'] = 'The id can contain only letters, numbers and underscores.'
  }

  if (!input.source) {
    errors['source'] = 'The source must be specified.'
  } else if (!['upload', 'download', 'local'].includes(input.source)) {
    errors['source'] = 'The source must be either "upload", "download" or "local".'
  }

  if (!input['source-address'] && input.source !== 'upload') {
    errors['source-address'] = 'The source address is expected for "download" and "local" datasets.'
  }

  if (input.name && !common.NAME_FORMAT.test(input.name)) {
    errors['name'] = 'The name can contain only printable ASCII characters.'
  }

  return errors
}

function createDataset (input) {
  // Collect fields of interest.
  input = decamelizeKeys(input, '-')
  let data = {
    'id': input['id'],
    'source': input['source'],
    'source-address': input['source-address'],
    'name': input['name'] || '',
    'description': input['description'] || ''
  }

  // Run post request as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.post('/datasets', data)
      .then(result => {
        let id = result.headers.location
        id = id.substr(id.lastIndexOf('/') + 1)

        // We return the ID of the new created object.
        resolve(id)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function updateDataset (id, updates) {
  // Collect fields of interest.
  updates = decamelizeKeys(updates, '-')
  let data = {}
  if ('name' in updates) {
    data['name'] = updates['name']
  }
  if ('description' in updates) {
    data['description'] = updates['description']
  }
  if ('status' in updates) {
    data['status'] = updates['status']
  }

  // Run patch request as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.patch('/datasets/' + id, data)
      .then(result => {
        resolve()
      })
      .catch(e => {
        reject(e)
      })
  })
}

function uploadDataset (id, data, filename, onProgress) {
  // Run upload request as a promise.
  return new Promise((resolve, reject) => {
    var upload = new tus.Upload(data, {
      endpoint: urljoin(this.baseURL, 'datasets', id, 'upload'),
      retryDelays: [0, 1000, 3000, 5000],
      metadata: {
        filename: filename
      },
      headers: this.authHeader,
      onProgress: onProgress,
      // chunkSize: 10000,
      onSuccess: () => resolve(),
      onError: (e) => reject(e)
    })

    // Start the upload
    upload.start()
  })
}

function listDatasetDirectoryByPath (id, relPath) {
  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.get('/datasets/' + id + '/data' + relPath)
      .then(response => {
        let result = response.data
        resolve(result)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function downloadDatasetByPath (id, relPath, inBrowser = false) {
  if (inBrowser) {
    const url = this.axiosInstance.defaults.baseURL + '/datasets/' + id + '/data' + relPath
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', id + '.tar') // or any other extension
    document.body.appendChild(link)
    link.click()
  } else {
    // Run query and collect results as a promise. The result passed to the promise is a Blob.
    return new Promise((resolve, reject) => {
      this.axiosInstance.get('/datasets/' + id + '/data' + relPath)
        .then(response => {
          let contentType = response.headers['Content-Type'] || ''
          let result = new Blob([response.data], { type: contentType })
          resolve(result)
        })
        .catch(e => {
          reject(e)
        })
    })
  }
}

export default {
  getDatasets: getDatasets,
  getDatasetById: getDatasetById,
  validateDatasetFields: validateDatasetFields,
  createDataset: createDataset,
  updateDataset: updateDataset,
  uploadDataset: uploadDataset,
  listDatasetDirectoryByPath: listDatasetDirectoryByPath,
  downloadDatasetByPath: downloadDatasetByPath
}
