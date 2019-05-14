'use strict'

// let common = require("./common");
import common from './common'
import decamelizeKeys from 'decamelize-keys'

function transformDataItem (input) {
  return {
    id: input.id,
    type: input.type,
    label: input.label,
    name: input.name,
    description: input.description,
    schemaIn: input['schema-in'],
    schemaOut: input['schema-out'],
    configSpace: input['config-space'],
    user: input.user,
    source: input.source,
    sourceAddress: input['source-address'],
    creationTime: new Date(input['creation-time']),
    status: input.status
  }
}

function getModules (query) {
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
    common.runGetQuery(this.axiosInstance, '/modules', query)
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

function getModuleById (id) {
  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.get('/modules/' + id)
      .then(response => {
        let result = transformDataItem(response.data.data)
        resolve(result)
      })
      .catch(e => {
        reject(e)
      })
  })
}

function validateModuleFields (input) {
  input = decamelizeKeys(input, '-')
  let errors = {}

  if (!input.id) {
    errors['id'] = 'The id must be specified.'
  } else if (!common.ID_FORMAT.test(input.id)) {
    errors['id'] = 'The id can contain only letters, numbers and underscores.'
  }

  if (!input.source) {
    errors['type'] = 'The type must be specified.'
  } else if (!['model', 'objective', 'optimizer'].includes(input.source)) {
    errors['source'] = 'The type must be either "model", "objective" or "optimizer".'
  }

  if (!input.source) {
    errors['source'] = 'The source must be specified.'
  } else if (!['upload', 'download', 'local', 'registry'].includes(input.source)) {
    errors['source'] = 'The source must be either "registry", "upload", "download" or "local".'
  }

  if (!input['source-address'] && input.source !== 'upload') {
    errors['source-address'] = 'The source address is expected for "download" and "local" modules.'
  }

  if (input.name && !common.NAME_FORMAT.test(input.name)) {
    errors['name'] = 'The name can contain only printable ASCII characters.'
  }

  return errors
}

function createModule (input) {
  // Collect fields of interest.
  input = decamelizeKeys(input, '-')
  let data = {
    'id': input['id'],
    'type': input['type'],
    'label': input['label'] || '',
    'source': input['source'],
    'source-address': input['source-address'],
    'name': input['name'] || '',
    'description': input['description'] || ''
  }

  // Run post request as a promise.
  return new Promise((resolve, reject) => {
    this.axiosInstance.post('/modules', data)
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

function updateModule (id, updates) {
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
    this.axiosInstance.patch('/modules/' + id, data)
      .then(result => {
        resolve()
      })
      .catch(e => {
        reject(e)
      })
  })
}

export default {
  getModules: getModules,
  getModuleById: getModuleById,
  validateModuleFields: validateModuleFields,
  createModule: createModule,
  updateModule: updateModule
}
