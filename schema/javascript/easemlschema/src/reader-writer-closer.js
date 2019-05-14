'use strict'

function ReaderWriterCloser () {
}

ReaderWriterCloser.prototype.read = function (buffer, offset, length, position) {
  throw new Error('Not implemented')
}

ReaderWriterCloser.prototype.readLines = function () {
  throw new Error('Not implemented')
}

ReaderWriterCloser.prototype.write = function (buffer, offset, length, position) {
  throw new Error('Not implemented')
}

ReaderWriterCloser.prototype.writeLines = function (data) {
  throw new Error('Not implemented')
}

ReaderWriterCloser.prototype.close = function () {
  throw new Error('Not implemented')
}

export default ReaderWriterCloser
