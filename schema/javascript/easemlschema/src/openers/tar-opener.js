'use strict'

import untar from 'js-untar'
import ReaderWriterCloser from '../reader-writer-closer'

function loadTarFile (file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = function (event) {
      console.log('reader load')

      untar(reader.result).then(
        function (extractedFiles) { // onSuccess
          // Build a hierarchy from files.
          let result = {}

          for (let i = 0; i < extractedFiles.length; i++) {
            let nameSplits = extractedFiles[i].name.split('/')
            let curObj = result

            for (let j = 0; j < nameSplits.length - 1; j++) {
              if ((nameSplits[j] in curObj) === false) {
                curObj[nameSplits[j]] = {}
              }
              curObj = curObj[nameSplits[j]]
            }
            let name = nameSplits[nameSplits.length - 1]

            if (name) {
              if (extractedFiles[i].type === '0') {
                curObj[name] = extractedFiles[i].buffer
              } else if (extractedFiles[i].type === '5') {
                curObj[name] = {}
              }
            }
          }

          resolve(result)
        },
        function (err) {
          reject(err)
        }
      )
    }
    reader.onerror = function (event) {
      console.log('FileReader Error', event)
      reject(event)
    }

    reader.readAsArrayBuffer(file)
  })
}

function new_opener (filestruct) {
  return function (root, rel_path, directory = false, read_only = true) {
    let pathSplits = (root + rel_path).split('/')
    let parent = null
    let element = filestruct
    let elementName = ''
    for (let i = 0; i < pathSplits.length; i++) {
      let name = pathSplits[i]
      if (name) {
        parent = element

        // If we are building a directory tree.
        if (directory && !read_only && !(name in element)) {
          element[name] = {}
        }

        element = element[name]
        elementName = name
      }
    }

    if (directory) {
      return Object.keys(element)
    } else {
      if (read_only) {
        if (element instanceof ArrayBuffer) {
          return new TarReaderWriterCloser(element)
        } else {
          return null
        }
      } else {
        parent[elementName] = new ArrayBuffer()
        return new TarReaderWriterCloser(parent[elementName])
      }
    }
  }
}

function TarReaderWriterCloser (arrayBuffer) {
  ReaderWriterCloser.call()
  this.arrayBuffer = arrayBuffer
}
TarReaderWriterCloser.prototype = Object.create(ReaderWriterCloser.prototype)
TarReaderWriterCloser.prototype.constructor = TarReaderWriterCloser

ReaderWriterCloser.prototype.read = function (buffer, offset, length, position) {
  // let dst = new Uint8Array(new Buffer(buffer));
  let dst = new Uint8Array(buffer)
  let src = new Uint8Array(this.arrayBuffer, position, length)
  dst.set(src, offset)

  // Returns number of bytes read.
  return length
}

ReaderWriterCloser.prototype.readLines = function () {
  // Retrieve file lines.
  let view = new DataView(this.arrayBuffer)
  let lines = readLinesFromDataView(view, this.arrayBuffer.byteLength)
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

/* ReaderWriterCloser.prototype.write = function(buffer, offset, length, position) {
    // Returns number of bytes read.
    return fs.writeSync(this.fd, new Buffer(buffer), offset, length, position);
}

ReaderWriterCloser.prototype.writeLines = function(data) {
    // Returns number of bytes read.
    return fs.writeSync(this.fd, "\n".join(data) + "\n");
} */

ReaderWriterCloser.prototype.close = function () {

}

export default {
  loadTarFile: loadTarFile,
  new_opener: new_opener
}
