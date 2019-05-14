module.exports = function (config) {
    config.set({
        frameworks: ['mocha', 'chai'],
        files: [
            'dist/**/*.js',
            'test/**/*.js'
        ],
        preprocessors: {
            'test/**/*.js': ['webpack', 'sourcemap']
        },
        reporters: ['progress'],
        port: 9876,  // karma web server port
        colors: true,
        logLevel: config.LOG_INFO,
        browsers: ['ChromeHeadless', 'Firefox', 'FirefoxDeveloper', 'FirefoxNightly', 'IE'],
        autoWatch: false,
        concurrency: Infinity,
        customLaunchers: {
            FirefoxHeadless: {
                base: 'Firefox',
                flags: ['-headless'],
            },
        },
    })
}
