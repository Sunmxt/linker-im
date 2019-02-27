const path = require('path');
const DIST_PATH = path.resolve(__dirname, "./dist")

module.exports = {
    entry: './linker.js',
    output: {
        filename: 'linker.js',
        path: DIST_PATH
    }
};
