'use strict'

import assert from 'assert'
import ReaderWriterCloser from '../reader-writer-closer'

const DATA_TYPES = ['f8', 'f4', 'i8', 'i4', 'i2', 'i1']
const BYTES_PER_ELEMENT = {
  'f8': 8,
  'f4': 4,
  'i8': 8,
  'i4': 4,
  'i2': 2,
  'i1': 1
}

function NpyWriter (writer, shape, dtype, columnMajor = false, bigEndian = false, version = 1) {
  assert(writer instanceof ReaderWriterCloser)
  assert(DATA_TYPES.indexOf(dtype) > -1)

  this.writer = writer
  this.shape = shape
  this.dtype = dtype
  this.columnMajor = columnMajor
  this.bigEndian = bigEndian
  this.version = version

  let pos = writeHeader(writer, shape, dtype, version, columnMajor, bigEndian)
  this.pos = pos
}

function NpyReader (reader) {
  assert(reader instanceof ReaderWriterCloser)

  let header = readHeader(reader)

  this.reader = reader
  this.shape = header.shape
  this.dtype = header.dtype
  this.columnMajor = header.columnMajor
  this.bigEndian = header.bigEndian
  this.version = header.version
  this.pos = header.pos
}

function writeHeader (writer, shape, dtype, version = 1, columnMajor = false, bigEndian = false) {
  assert(writer instanceof ReaderWriterCloser)
  assert(version in [1, 2])

  // Assemble all parameters.
  const magicString = '\x93NUMPY'
  const versionString = (version === 1) ? '\x01\x00' : '\x02\x00'
  const descrString = (bigEndian ? '>' : '<') + dtype
  const shapeString = '(' + String(shape.join(',')) + ',' + ')'
  const fortranString = columnMajor ? 'True' : 'False'

  // Assemble the header.
  const header = "{'descr': '" + descrString + "', 'fortran_order': " + fortranString +
        ", 'shape': " + shapeString + ', }'

  // Compute the padding.
  const lengthBytes = (version === 1) ? 2 : 4
  const unpaddedLength = header.length
  const padMul = (version === 1) ? 16 : 16
  const padLength = (padMul - unpaddedLength % padMul) % padMul
  const padding = ' '.repeat(padLength)
  const headerLength = unpaddedLength + padLength
  const totalHeaderLength = magicString.length + versionString.length + lengthBytes + headerLength
  assert(headerLength % padMul === 0)

  // Build the array buffer.
  const BUFFER = new ArrayBuffer(totalHeaderLength)
  const view = new DataView(BUFFER)
  let pos = 0

  // Write the magic string and version.
  pos = writeStringToDataView(view, magicString + versionString, pos)

  // Write header length.
  if (version === 1) {
    view.setUint16(pos, headerLength, true)
  } else {
    view.setUint32(pos, headerLength, true)
  }
  pos += lengthBytes

  // Write header.
  pos = writeStringToDataView(view, header + padding, pos)

  // Write the buffer.
  writer.write(BUFFER, 0, totalHeaderLength, 0)

  return totalHeaderLength
}

function readHeader (reader) {
  assert(reader instanceof ReaderWriterCloser)

  // Build a buffer for the magic string and version.
  const magicStringBuffer = new ArrayBuffer(10)

  // Read the magic string and version.
  reader.read(magicStringBuffer, 0, 8, 0)
  const magicStringView = new DataView(magicStringBuffer, 0, 6)
  const magicString = readDataViewAsString(magicStringView)
  if (magicString !== '\x93NUMPY') {
    throw new Error('The given file is not a valid NUMPY file.')
  }
  const versionView = new DataView(magicStringBuffer, 0, 8)
  const [versionMajor, versionMinor] = [versionView.getUint8(6), versionView.getUint8(7)]
  if ((versionMajor in [1, 2]) === false || versionMinor !== 0) {
    throw new Error('Unknown NUMPY file version ' + versionMajor + '.' + versionMinor)
  }

  // Read header size.
  const lengthBytes = (versionMajor === 1) ? 2 : 4
  const lengthBuffer = new ArrayBuffer(lengthBytes)
  reader.read(lengthBuffer, 0, lengthBytes, 8)
  const lengthView = new DataView(lengthBuffer)
  const headerLength = (versionMajor === 1) ? lengthView.getUint16(0, true) : lengthView.getUint32(0, true)

  // Read the header.
  const headerDictLength = headerLength - lengthBytes - 8
  const headerBuffer = new ArrayBuffer(headerDictLength)
  reader.read(headerBuffer, 0, headerDictLength, lengthBytes + 8)
  const headerView = new DataView(headerBuffer)
  const headerString = readDataViewAsString(headerView)

  // Parse the header.
  const headerJson = headerString
    .replace('True', 'true')
    .replace('False', 'false')
    .replace(/'/g, `"`)
    .replace(/,\s*}/, ' }')
    .replace(/,?\)/, ']')
    .replace('(', '[')
  const header = JSON.parse(headerJson)

  // Extract properties.
  const bigEndian = header.descr[0] === '>'
  const columnMajor = header.fortran_order
  const dtype = header.descr.slice(1)
  const shape = header.shape
  const version = versionMajor

  let result = {
    'bigEndian': bigEndian,
    'columnMajor': columnMajor,
    'dtype': dtype,
    'shape': shape,
    'version': version,
    'pos': headerLength + lengthBytes + 8
  }

  return result
}

function writeStringToDataView (view, str, pos) {
  for (let i = 0; i < str.length; i++) {
    view.setInt8(pos + i, str.charCodeAt(i))
  }
  return pos + str.length
}

function readDataViewAsString (view) {
  let out = ''
  for (let i = 0; i < view.byteLength; i++) {
    const val = view.getUint8(i)
    if (val === 0) {
      break
    }
    out += String.fromCharCode(val)
  }
  return out
}

function numberOfElements (shape) {
  if (shape.length === 0) {
    return 1
  } else {
    return shape.reduce((a, b) => a * b)
  }
}

NpyWriter.prototype.write = function (data, close = true) {
  assert(data.length === numberOfElements(this.shape))

  // Build an array buffer to store the data.
  const ELEM_BYTES = BYTES_PER_ELEMENT[this.dtype]
  const BUFFER_SIZE = data.length * ELEM_BYTES
  const BUFFER = new ArrayBuffer(BUFFER_SIZE)
  const view = new DataView(BUFFER)
  let pos = 0

  // Write to the buffer in the proper format.
  switch (this.dtype) {
    case 'f8':
      for (let i = 0; i < data.length; i++) {
        view.setFloat64(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }

      break

    case 'f4':
      for (let i = 0; i < data.length; i++) {
        view.setFloat32(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }
      break
    case 'i8':
      for (let i = 0; i < data.length; i++) {
        view.setInt64(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }

      break

    case 'i4':
      for (let i = 0; i < data.length; i++) {
        view.setInt32(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }
      break
    case 'i2':
      for (let i = 0; i < data.length; i++) {
        view.setInt16(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }

      break

    case 'i1':
      for (let i = 0; i < data.length; i++) {
        view.setInt8(pos, data[i], !this.bigEndian)
        pos += ELEM_BYTES
      }
      break
  }

  // Close the writer if specified.
  if (close) {
    this.writer.close()
  }

  // Write the buffer to the file.
  this.writer.write(BUFFER, 0, BUFFER_SIZE, this.pos)

  // Shift the position by the amount of data we've just written.
  this.pos += BUFFER_SIZE
}

NpyReader.prototype.read = function (close = true) {
  // Compute the buffer size and read the data.
  const dataLength = numberOfElements(this.shape)
  const ELEM_BYTES = BYTES_PER_ELEMENT[this.dtype]
  const BUFFER_SIZE = dataLength * ELEM_BYTES
  const BUFFER = new ArrayBuffer(BUFFER_SIZE)
  this.reader.read(BUFFER, 0, BUFFER_SIZE, this.pos)
  this.pos += BUFFER_SIZE

  // Close the reader if specified.
  if (close) {
    this.reader.close()
  }

  // Feed the data into an appropriate array and return.
  switch (this.dtype) {
    case 'f8':
      return new Float64Array(BUFFER)

    case 'f4':
      return new Float32Array(BUFFER)

    case 'i8':
      // eslint-disable-next-line no-undef
      return new Int64Array(BUFFER)

    case 'i4':
      return new Int32Array(BUFFER)

    case 'i2':
      return new Int16Array(BUFFER)

    case 'i1':
      return new Int8Array(BUFFER)
  }
}

export default {
  'NpyWriter': NpyWriter,
  'NpyReader': NpyReader
}
