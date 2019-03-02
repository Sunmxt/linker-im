// Linker-IM Websocket client.

proto = require('./proto')

const CLOSED = 0
const CONNECTING = 1
const CONNECTED = 2

function LinkerClient(url) {
    this.url = url
    this.cursor = 0
    this.pendings = {}
    this.state = CLOSED
}

LinkerClient.prototype.connect = function (namespace, credential) {
    client = this
    ws = new WebSocket(this.url)
    ws.onopen = function (event) {
        if(ws.readyState == 1) {
            ws.send(proto.encodeConnectRequest(namespace, credential))
            client.ws = ws
            client.state = CONNECTING
            client.namespace = namespace
            client.onconnecting()
        }
    }

    ws.onmessage = function (event) {
        reader = new FileReader()
        reader.readAsArrayBuffer(event.data)
        reader.onloadend = function() {
            client.onrawmessage(reader.result)
        }
    }

    ws.onclose = function (event) {
        client.state = CLOSED
        delete client.namespace
        delete client.ws
        delete client.session
        client.cursor = 0
        client.onclosed()
    }
}

LinkerClient.prototype.throwIfNotConnected = function () {
    if(!this.session || !this.namespace || !this.ws ) {
        throw new proto.OperationError('Not connected.')
        return
    }
}

LinkerClient.prototype.push = function (msgs, onReply) {
    this.sendOp(function (session, id, namespace) {
        return proto.encodePushRequest(session, id, namespace, msgs)
    }, onReply)
}

LinkerClient.prototype.subscribe = function (group, onReply) {
    this.sendOp(function (session, id, namespace) {
        return proto.encodeSubscription(session, id, namespace, group, proto.subscribeOp.Sub)
    }, onReply)
}

LinkerClient.prototype.unsubscribe = function (group, onReply) {
    this.sendOp(function (session, id, namespace) {
        return proto.encodeSubscription(session, id, namespace, group, proto.subscribeOp.Unsub)
    }, onReply)
}

LinkerClient.prototype.sendOp = function (getBinary, onReply) {
    this.throwIfNotConnected()
    id = this.cursor
    ws.send(getBinary(this.session, id, this.namespace))
    if (onReply) {
        this.pendings[id] = onReply
    }
    this.cursor ++ 
    return id
}

LinkerClient.prototype.close = function () {
    ws.close()
}

LinkerClient.prototype.onmessage = function (timestamp, sequence, group, message) {
    console.log("Message --> {timestamp = " + timestamp + ", sequence = " + sequence + ", group = " + group + ", message = " + message +  "}")
}

LinkerClient.prototype.onerror = function (error) { throw error }
LinkerClient.prototype.onconnected = function () {}
LinkerClient.prototype.onconnecting = function () {}
LinkerClient.prototype.onclosed = function() {}

LinkerClient.prototype.onrawmessage = function (buffer) {
    fields = proto.decodeProtocol(this, buffer)
    onReply = this.pendings[fields.requestID]
    if (onReply) {
        delete this.pendings[fields.requestID]
        onReply(fields)
    }
    switch(fields.unitType){
    case proto.unitType.Connected:
        if(this.state != CONNECTING) {
            client.onerror(new proto.ProtocolError('Duplicated connecting reply.', buffer, fields))
            return
        }
        this.state = CONNECTED
        client.onconnected()
        this.session = fields.session
        this.cursor = 1
        break

    case proto.unitType.Message:
        break

    case proto.unitType.Error:
        if (fields.message.length > 0) {
            client.onerror(new proto.OperationError(fields.message, buffer, fields))
        }
        break

    case proto.unitType.PushResult:
        break
    }
}

module.exports = {
    Client: LinkerClient
};
