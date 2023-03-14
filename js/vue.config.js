// vue.config.js
module.exports = {
    devServer: {
        host: '0.0.0.0',
        port: 80,
        https: false,
        hotOnly: false,
        disableHostCheck: true,
        proxy: {
            '^/api': {
                target: 'http://api',
                changeOrigin: true
            },
        }
    },
    configureWebpack: {
        devtool: 'cheap-module-source-map'
    },
    chainWebpack: config => {
        if (process.env.NODE_ENV === 'development') {
           config
            .output
            .filename('[name].[hash].js')
            .end()
        }
    }
}
