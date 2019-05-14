'use strict'

import sch from '../../src/schema'
import ds from '../../src/dataset'
import fsOpener from '../../src/openers/fs-opener'

import { expect } from 'chai'

import fs from 'fs'
import path from 'path'

const relTestPath = path.join(__dirname, '../../../../test-examples')
const datasetInferPositivePath = 'dataset/infer/positive'
const datasetInferNegativePath = 'dataset/infer/negative'
const datasetGenPositivePath = 'dataset/generate/positive'

describe('Dataset', function () {
  describe('#infer', function () {
    describe('#positive', function () {
      var files = fs.readdirSync(path.join(relTestPath, datasetInferPositivePath))
      for (let i = 0; i < files.length; i++) {
        let dirName = files[i]
        if (fs.lstatSync(path.join(relTestPath, datasetInferPositivePath, dirName)).isDirectory()) {
          let testName = path.join(datasetInferPositivePath, dirName)
          let exampleFile = path.join(relTestPath, datasetInferPositivePath, dirName + '.json')
          let exampleDir = path.join(relTestPath, datasetInferPositivePath, dirName)
          it(testName, testDatasetInfer(exampleFile, exampleDir, true))
        }
      }
    })
    describe('#negative', function () {
      var files = fs.readdirSync(path.join(relTestPath, datasetInferNegativePath))
      for (let i = 0; i < files.length; i++) {
        let dirName = files[i]
        if (fs.lstatSync(path.join(relTestPath, datasetInferNegativePath, dirName)).isDirectory()) {
          let testName = path.join(datasetInferNegativePath, dirName)
          let exampleDir = path.join(relTestPath, datasetInferNegativePath, dirName)
          it(testName, testDatasetInfer(null, exampleDir, false))
        }
      }
    })
  })
  describe('#infer', function () {
    describe('#positive', function () {
      var files = fs.readdirSync(path.join(relTestPath, datasetGenPositivePath))
      for (let i = 0; i < files.length; i++) {
        if (files[i].endsWith('.json')) {
          let testName = path.join(datasetGenPositivePath, files[i].slice(0, -'.json'.length))
          let exampleFile = path.join(relTestPath, datasetGenPositivePath, files[i])
          it(testName, testDatasetGenerate(exampleFile))
        }
      }
    })
  })
})

function testDatasetInfer (exampleFile, exampleDir, positiveExample) {
  return function () {
    let caughtError = null
    let srcSchema = null

    try {
      // Load the dataset.
      let dataset = ds.load(exampleDir, fsOpener.defaultOpener, true)

      // Infer the schema.
      srcSchema = dataset.inferSchema()
    } catch (error) {
      if (positiveExample) {
        throw error
      } else {
        caughtError = error
        if ((caughtError instanceof ds.DatasetException) === false) {
          throw error
        }
      }
    }

    if (positiveExample) {
      // There should have been no errors.
      expect(caughtError).to.be.null

      // Load the JSON file and the schema.
      let dstData = fs.readFileSync(exampleFile)
      let dst = JSON.parse(dstData)
      let dstSchema = sch.load(dst)

      // Match the loaded schema to the dataset.
      let match = dstSchema.match(srcSchema)
      expect(match).to.be.true
    } else {
      // The caught error should be a dataset exception.
      expect(caughtError).to.be.instanceOf(ds.DatasetException)
    }
  }
}

function testDatasetGenerate (exampleFile) {
  return function () {
    // Load the JSON file and the schema.
    let dstData = fs.readFileSync(exampleFile)
    let dst = JSON.parse(dstData)
    let dstSchema = sch.load(dst)

    // Generate random dataset from the schema.
    let dataset = ds.generateFromSchema('root', dstSchema, 10, 10)

    // Infer the schema and match.
    let srcSchema = dataset.inferSchema()
    let match = dstSchema.match(srcSchema, false)
    expect(match).to.be.true
  }
}
