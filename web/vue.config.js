var webpack = require("webpack");

module.exports = {
    devServer:{
	port:8081,
    },
    configureWebpack: {
      plugins: [
        new webpack.ProvidePlugin({
            $: 'jquery',
            jquery: 'jquery',
            'window.jQuery': 'jquery',
            jQuery: 'jquery'
        })
      ]
    }
  }
