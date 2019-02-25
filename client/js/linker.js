// Linker-IM Websocket client.

utf8 = require('utf8')
proto = require('./proto')

function LinkerClient(url) {
    dummy = function () {}
    this.$linker = new dummy()
    this.$linker.url = url
}

LinkerClient.prototype.connect = function (namespace, credential) {
    client = this
    ws = new WebSocket(this.$linker.url)
    console.log(proto.encodeConnectRequest(namespace, credential))
    ws.onopen = function (event) {
    }
    ws.onerror = function (event) {
        client.onerror(event)
    }
    this.$linker.ws = ws
}

LinkerClient.prototype.push = function (message, group) {
}

LinkerClient.prototype.onmessage = function (message) {
}

LinkerClient.prototype.onerror = function (event) {
}

module.exports = {
    Client: LinkerClient
};
