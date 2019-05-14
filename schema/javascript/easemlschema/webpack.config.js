const path = require('path')
const name = 'easemlschema'

module.exports = {
  mode: 'development',
  entry: './src/index.js',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: name + '.js',
    sourceMapFilename: name + '.js.map',
    library: name,
    libraryTarget: 'umd'
  },
  devtool: 'source-map',
  externals: {
    fs: 'commonjs fs',
    path: 'commonjs path'
  }
}
