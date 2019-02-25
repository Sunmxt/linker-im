const path = require('path');
const HTMLWebpackPlugin = require('html-webpack-plugin')

const DIST_PATH = path.resolve(__dirname, "./dist")

module.exports = {
    entry: './websocket.js',
    output: {
        filename: 'websocket.js',
        path: DIST_PATH
    },
    plugins: [
        new HTMLWebpackPlugin({
            template: 'websocket.htm'
        })
    ],
    devServer: {
        contentBase: DIST_PATH,
        compress: true,
        port: 9000
    }
};
