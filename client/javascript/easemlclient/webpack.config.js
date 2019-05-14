const path = require('path');
const name = 'easemlclient'

module.exports = {
    mode: "development",
    entry: './src/index.js',
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: name + '.js',
        sourceMapFilename: name + '.js.map',
        library: name,
        libraryTarget: 'umd',
    },
    devtool: 'source-map'
};
