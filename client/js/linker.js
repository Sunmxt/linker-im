// Linker-IM Websocket client.

proto = require('./proto')

const CONNECTING = 1
const CONNECTED = 2
const CLOSED = 3

function LinkerClient(url) {
    this.$linker = {
        url: url
        , cursor: 0
        , pendings: {}
        , state: CONNECTING
        , session: null
    }
}

(function () {
    F = function(){}
    F.prototype = EventTarget.prototype
    LinkerClient.prototype = new F()
    LinkerClient.prototype.constructor = LinkerClient
})()

LinkerClient.prototype.connect = function (namespace, credential) {
    client = this
    ws = new WebSocket(this.$linker.url)
    ws.onopen = function (event) {
        if(ws.readyState == 1) {
            ws.send(proto.encodeConnectRequest(namespace, credential))
        }
    }
    ws.onerror = function (event) {
        client.onerror(event)
    }
    ws.onmessage = function (event) {
        reader = new FileReader()
        reader.readAsArrayBuffer(event.data)
        reader.onloadend = function() {
            client.onrawmessage(reader.result)
        }
    }
    this.$linker.ws = ws
}

LinkerClient.prototype.push = function (group, msg) {}
LinkerClient.prototype.subscribe = function (group) {}
LinkerClient.prototype.unsubscribe = function (group) {}
LinkerClient.prototype.keepalive = function () {}

LinkerClient.prototype.onmessage = function (timestamp, sequence, group, message) {
    console.log("Message --> {timestamp = " + timestamp + ", sequence = " + sequence + ", group = " + group + ", message = " + message +  "}")
}

LinkerClient.prototype.onerror = function (error) {throw error}

LinkerClient.prototype.onconnected = function () {
}

LinkerClient.prototype.onrawmessage = function (buffer) {
    fields = proto.decodeProtocol(this, buffer)
    console.log(fields)
    switch(fields.unitType){
    case proto.unitType.Connected:
        if(this.$linker.state != CONNECTING) {
            client.onerror(new proto.ProtocolError('Duplicated connecting reply.'))
            return
        }
        this.$linker.state = CONNECTED
        client.onconnected()
        break

    case proto.unitType.Message:
        break

    case proto.unitType.Error:
        break
    }
}

module.exports = {
    Client: LinkerClient
};
