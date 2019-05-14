'use strict'

import sch from '../../src/schema'
import { expect } from 'chai'

import fs from 'fs'
import path from 'path'

const relTestPath = path.join(__dirname, '../../../../test-examples')
const schemaValPositivePath = 'schema/validate/positive'
const schemaValNegativePath = 'schema/validate/negative'
const schemaMatchPositivePath = 'schema/match/positive'
const schemaMatchNegativePath = 'schema/match/negative'

describe('Schema', function () {
  describe('#validate', function () {
    describe('#positive', function () {
      var files = fs.readdirSync(path.join(relTestPath, schemaValPositivePath))
      for (let i = 0; i < files.length; i++) {
        if (files[i].endsWith('.json')) {
          let testName = path.join(schemaValPositivePath, files[i].slice(0, -'.json'.length))
          let exampleFile = path.join(relTestPath, schemaValPositivePath, files[i])
          it(testName, testSchemaValidate(exampleFile, true))
        }
      }
    })
    describe('#negative', function () {
      var files = fs.readdirSync(path.join(relTestPath, schemaValNegativePath))
      for (let i = 0; i < files.length; i++) {
        if (files[i].endsWith('.json')) {
          let testName = path.join(schemaValNegativePath, files[i].slice(0, -'.json'.length))
          let exampleFile = path.join(relTestPath, schemaValNegativePath, files[i])
          it(testName, testSchemaValidate(exampleFile, false))
        }
      }
    })
  })
  describe('#match', function () {
    describe('#positive', function () {
      var files = fs.readdirSync(path.join(relTestPath, schemaMatchPositivePath))
      for (let i = 0; i < files.length; i++) {
        if (files[i].endsWith('-src.json')) {
          let fileName = files[i].slice(0, -'-src.json'.length)
          let testName = path.join(schemaMatchPositivePath, fileName)
          let exampleFileSrc = path.join(relTestPath, schemaMatchPositivePath, fileName + '-src.json')
          let exampleFileDst = path.join(relTestPath, schemaMatchPositivePath, fileName + '-dst.json')
          it(testName, testSchemaMatch(exampleFileSrc, exampleFileDst, true))
        }
      }
    })
    describe('#negative', function () {
      var files = fs.readdirSync(path.join(relTestPath, schemaMatchNegativePath))
      for (let i = 0; i < files.length; i++) {
        if (files[i].endsWith('-src.json')) {
          let fileName = files[i].slice(0, -'-src.json'.length)
          let testName = path.join(schemaMatchNegativePath, fileName)
          let exampleFileSrc = path.join(relTestPath, schemaMatchNegativePath, fileName + '-src.json')
          let exampleFileDst = path.join(relTestPath, schemaMatchNegativePath, fileName + '-dst.json')
          it(testName, testSchemaMatch(exampleFileSrc, exampleFileDst, false))
        }
      }
    })
  })
})

function testSchemaValidate (exampleFile, positiveExample) {
  return function () {
    // Load the JSON file.
    let srcData = fs.readFileSync(exampleFile)
    let src = JSON.parse(srcData)

    if (positiveExample) {
      // Load the schema.
      let schema = sch.load(src)
      expect(schema).to.be.instanceof(sch.Schema)
    } else {
      let caughtError = null
      try {
        sch.load(src)
      } catch (error) {
        caughtError = error
        if ((caughtError instanceof sch.SchemaException) === false) {
          throw error
        }
      }
      expect(caughtError).to.be.instanceOf(sch.SchemaException)
    }
  }
}

function testSchemaMatch (exampleFileSrc, exampleFileDst, positiveExample) {
  return function () {
    // Load the JSON files.
    let srcData = fs.readFileSync(exampleFileSrc)
    let src = JSON.parse(srcData)
    let dstData = fs.readFileSync(exampleFileDst)
    let dst = JSON.parse(dstData)

    // Load the schemas.
    let srcSchema = sch.load(src)
    let dstSchema = sch.load(dst)

    if (positiveExample) {
      // Match the schemas.
      let match = dstSchema.match(srcSchema, true)
      expect(match).to.be.not.null
      expect(match).to.be.instanceof(sch.Schema)
    } else {
      // Match the schemas.
      let match = dstSchema.match(srcSchema, false)
      expect(match).to.be.a('boolean')
      expect(match).to.be.false
    }
  }
}
