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
const OP_PUSH_RESULT = 9

const OP_SUB_ADD = 0
const OP_SUB_CANCEL = 1

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
    , PushResult: OP_PUSH_RESULT
}

const connectType = {
    Basic: 1
    , Session: 2
}

const subscribeOp = {
    Sub: 0
    , Unsub: 1
}

function inhert(c, p) {
    F = function (){}
    F.prototype = p.prototype
    c.prototype = new F()
    c.prototype.constructor = c
}

function putErrorCommonInfo(message, data, fields) {
    if ('captureStackTrace' in Error) {
        Error.captureStackTrace(this, ProtocolError)
    } else {
        this.stack = (new Error).stack
    }
    this.data = data
    this.fields = fields
    this.message = message
}

function ProtocolError(message, data, fields) {
    putErrorCommonInfo.call(message, data, fields)
}

function AuthError(message, data, fields) {
    putErrorCommonInfo.call(message, data, fields)
}

function OperationError(message, data, fields) {
    putErrorCommonInfo.call(message, data, fields)
}

inhert(ProtocolError, Error)
inhert(AuthError, Error)
inhert(OperationError, Error)

AuthError.prototype.name = 'AuthError'
ProtocolError.prototype.name = 'ProtocolError'
OperationError.prototype.name = 'OperationError'


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
        fields.session = view.buffer.slice(3 + view.byteOffset + fields.authErrorLength, 3 + view.byteOffset + fields.authErrorLength + fields.sessionLength)
    }
    , 6: function (fields, view) { // Message
    }
    , 7: function (fields, view) { // Error
        if (view.byteOffset < 1) {
            throw new ProtocolError('Unin too short', view.buffer, fields)
        }
        errLen = view.getUint8(0)
        if (view.byteOffset < errLen + 1) {
            throw new ProtocolError('Unin too short', view.buffer, fields)
        }
        dec = new TextDecoder()
        fields.message = dec.decode(view.buffer.slice(1, 1 + errLen))
    }
}


u8sEncode = (new TextEncoder).encode

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

function encodeSubscription(session, id, namespace, group, type) {
    if(session instanceof ArrayBuffer) {
        session = new Uint8Array(session)
    }
    enc = new TextEncoder()
    namespace = enc.encode(namespace)
    group = enc.encode(group)
    view = newProtocolUnit(id, protocolUnitType.Sub, 7 + namespace.byteLength + group.byteLength + session.byteLength)
    view.setUint8(0, type)
    view.setUint16(1, namespace.byteLength)
    view.setUint16(3, session.byteLength)
    view.setUint16(5, group.byteLength)
    putProtocolUnitBody(7, view, [namespace, session, group])
    return view.buffer
}

function encodeMessageBody(group, raw) {
    enc = new TextEncoder()
    if(raw instanceof String) {
        raw = enc.encode(raw)
    }
    group = enc.encode(group)
    buffer = new ArrayBuffer(4)
    view = new DataView(buffer)
    view.setUint16(0, group.byteLength, false)
    view.setUint16(2, raw.byteLength, false)
    return [buffer, group, raw]
}

function encodePushRequest(session, id, namespace, messages) {
    enc = new TextEncoder()
    namespace = enc.encode(namespace)
    if(session instanceof String) {
        session = enc.encode(session)
    }
    frags = []
    len = 0
    for(msg in messages) {
        encodedMsg = encodeMessageBody(msg.group, msg.data)
        frags = frags.concat(encodedMsg)
        len += encodedMsg[0].bodyLength + encodedMsg[1].bodyLength + encodedMsg[2].bodyLength
    }
    view = newProtocolUnit(id, protocolUnitType.Push, 2 + len)
    view.setUint16(0, namespace.byteLength, false)
    view.setUint16(2, session.byteLength, false)
    view.setUint16(4, messages.length, false)
    putProtocolUnitBody(6, view, [namespace, session].concat(frags))
    return view.buffer
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
    , encodeSubscription: encodeSubscription
    , decodeProtocol: decodeProtocol
    , ProtocolError: ProtocolError
    , AuthError: AuthError
    , bodyDecoder: bodyDecoder
    , unitType: protocolUnitType
    , subscribeOp: subscribeOp
}
