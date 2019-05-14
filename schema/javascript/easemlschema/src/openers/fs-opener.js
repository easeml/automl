'use strict'

import fs from 'fs'
import path from 'path'
import ReaderWriterCloser from '../reader-writer-closer'

// Source: https://stackoverflow.com/a/40686853
function mkDirByPathSync (targetDir, { isRelativeToScript = false } = {}) {
  const sep = path.sep
  const initDir = path.isAbsolute(targetDir) ? sep : ''
  const baseDir = isRelativeToScript ? __dirname : '.'

  return targetDir.split(sep).reduce((parentDir, childDir) => {
    const curDir = path.resolve(baseDir, parentDir, childDir)
    try {
      fs.mkdirSync(curDir)
    } catch (err) {
      if (err.code === 'EEXIST') { // curDir already exists!
        return curDir
      }

      // To avoid `EISDIR` error on Mac and `EACCES`-->`ENOENT` and `EPERM` on Windows.
      if (err.code === 'ENOENT') { // Throw the original parentDir error on curDir `ENOENT` failure.
        throw new Error(`EACCES: permission denied, mkdir '${parentDir}'`)
      }

      const caughtErr = ['EACCES', 'EPERM', 'EISDIR'].indexOf(err.code) > -1
      if (!caughtErr || caughtErr && targetDir === curDir) {
        throw err // Throw if it's just the last created dir.
      }
    }

    return curDir
  }, initDir)
}

function default_opener (root, rel_path, directory = false, read_only = true) {
  let abs_path = path.join(root, rel_path)

  if (directory) {
    if (read_only === false) {
      mkDirByPathSync(abs_path)
    }
    return fs.readdirSync(abs_path)
  } else {
    let flags = read_only ? 'r' : 'w'
    let fd = fs.openSync(abs_path, flags)
    return new FsReaderWriterCloser(fd)
  }
}

function FsReaderWriterCloser (fd) {
  ReaderWriterCloser.call()
  this.fd = fd
}
FsReaderWriterCloser.prototype = Object.create(ReaderWriterCloser.prototype)
FsReaderWriterCloser.prototype.constructor = FsReaderWriterCloser

ReaderWriterCloser.prototype.read = function (buffer, offset, length, position) {
  // Returns number of bytes read.
  return fs.readSync(this.fd, new Buffer(buffer), offset, length, position)
}

ReaderWriterCloser.prototype.readLines = function () {
  // Get file size from stats, create a buffer and read the file.
  let stats = fs.fstatSync(this.fd)
  let buffer = new ArrayBuffer(stats.size)
  let size = fs.readSync(this.fd, new Buffer(buffer), 0, stats.size, 0)

  // Retrieve file lines.
  let view = new DataView(buffer)
  let lines = readLinesFromDataView(view, size)
  return lines
}

function readLinesFromDataView (view, size) {
  let data = []
  let lineEnd = false
  let curLine = ''
  for (let i = 0; i < size; i++) {
    const val = view.getUint8(i)
    if (val === 0) {
      break
    }
    let char = String.fromCharCode(val)

    if (char === '\n' || char === '\r') {
      lineEnd = true
    } else {
      if (lineEnd) {
        data.push(curLine)
        curLine = ''
      }
      lineEnd = false
    }
    curLine += char
  }

  if (curLine.length > 0) {
    data.push(curLine)
  }

  return data
}

ReaderWriterCloser.prototype.write = function (buffer, offset, length, position) {
  // Returns number of bytes read.
  return fs.writeSync(this.fd, new Buffer(buffer), offset, length, position)
}

ReaderWriterCloser.prototype.writeLines = function (data) {
  // Returns number of bytes read.
  return fs.writeSync(this.fd, '\n'.join(data) + '\n')
}

ReaderWriterCloser.prototype.close = function () {
  fs.closeSync(this.fd)
}

export default {
  default_opener: default_opener
}
