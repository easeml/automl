'use strict'

// let common = require("./common");
import common from './common'
import decamelizeKeys from 'decamelize-keys'

function transformDataItem (input) {
  return {
    id: input.id,
    processId: input['process-id'],
    hostId: input['host-id'],
    hostAddress: input['host-address'],
    startTime: new Date(input['start-time']),
    lastKeepalive: new Date(input['last-keepalive']),
    type: input.type,
    status: input.status
  }
}

function getProcesses (query) {
  // This allows us to accept camel case keys.
  query = decamelizeKeys(query || {}, '-')

  // Run query and collect results as a promise.
  return new Promise((resolve, reject) => {
    common.runGetQuery(this.axiosInstance, '/processes', query)
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

export default {
  getProcesses: getProcesses
}
