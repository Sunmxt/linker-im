// Javascript binding of Linker's protocol.

const OP_DUMMY = 0
const OP_RESPONSE = 1
const OP_CONNECT = 2
const OP_SUB = 3
const OP_KEEPALIVE = 4
const OP_PUSH = 5
const OP_MESSAGE = 6
const OP_ERROR = 7
const OP_CONNECTED = 8

function ProtocolError(string, data, fields) {
    Error.call(this, string)
    this.data = data
    this.fields = fields
}

function AuthError(string) {
    Error.call(this, string)
}

function OperationError(string) {
    Error.call(this, string)
}

const protocolUnitType = {
    Dummy: OP_DUMMY
    , Response: OP_RESPONSE
    , Connect: OP_CONNECT
    , Sub: OP_SUB
    , Push: OP_PUSH
    , Message: OP_MESSAGE
    , Error: OP_ERROR
    , Keepalive: OP_KEEPALIVE
    , Connected: OP_CONNECTED
}

const connectType = {
    Basic: 1
    , Session: 2
}

const bodyDecoder = {
    8: function (fields, view) { // Connected
        if (view.byteLength < 3) {
            throw new ProtocolError('Unit too short', view.buffer, fields)
        } 
        fields.authErrorLength = view.getUint8(0)
        fields.sessionLength = view.getUint16(1, false)
        if (view.byteLength < 3 + fields.authErrorLength + fields.sessionLength) {
            throw new ProtocolError('Unit too short', view.buffer, fields)
        }
        dec = new TextDecoder()
        fields.authError = dec.decode(view.buffer.slice(3 + view.byteOffset, 3 + view.byteOffset + fields.authErrorLength))
        fields.session = dec.decode(view.buffer.slice(3 + view.byteOffset + fields.authErrorLength, 3 + view.byteOffset + fields.authErrorLength + fields.sessionLength))
    }
    , 6: function (fields, view) { // Message
    }
}

function inhert(c, p) {
    F = function (){}
    F.prototype = p.prototype
    c.prototype = new F()
    AuthError.prototype.constructor = AuthError.prototype
}

inhert(ProtocolError, Error)
inhert(AuthError, Error)

function newProtocolUnit(requestID, unitType, bodyLength) {
    buffer = new ArrayBuffer(bodyLength + 6)
    view = new DataView(buffer)
    view.setUint16(0, unitType, false)
    view.setUint32(2, requestID, false)
    return new DataView(buffer, 6)
}

function putProtocolUnitBody(offset, view, byteArrays) {
    for(idx in byteArrays) {
        byteArrays[idx].map(function (v, i, a) {
            view.setUint8(offset + i, v)
        })
        offset += byteArrays[idx].byteLength
    }
}

function encodeConnectRequest(namespace, credential, type) {
    enc = new TextEncoder()
    namespace = enc.encode(namespace)
    credential = enc.encode(credential)
    view = newProtocolUnit(0, protocolUnitType.Connect, 5 + namespace.byteLength + credential.byteLength)
    view.setUint8(0, connectType.Basic)
    view.setUint16(1, namespace.byteLength, false)
    view.setUint16(3, credential.byteLength, false)
    putProtocolUnitBody(5, view, [namespace, credential])
    return view.buffer
}

function encodeSubscription(namespace, group, type) {
}

function decodeProtocol(client, buffer) {
    fields = {}
    view = new DataView(buffer)
    try {
        if(buffer.byteLength < 6) {
            throw new ProtocolError('Unit too short', buffer)
            return fields
        }
        fields.unitType = view.getUint16(0, false)
        fields.requestID = view.getUint32(2, false)
        decode = bodyDecoder[fields.unitType]
        if(!decode) {
            throw new ProtocolError('Invalid unit type')
            return fields
        }
        decode(fields, new DataView(buffer, 6))
    } catch(err) {
        client.onerror(err)
    }
    return fields
}

module.exports = {
    unitTypes: protocolUnitType
    , encodeConnectRequest: encodeConnectRequest
    , decodeProtocol: decodeProtocol
    , ProtocolError: ProtocolError
    , AuthError: AuthError
    , bodyDecoder: bodyDecoder
    , unitType: protocolUnitType
}
