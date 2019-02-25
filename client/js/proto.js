// Javascript binding of Linker's protocol.

const protocolUnitType = {
    Dummy: 0
    , Response: 1
    , Connect: 2
    , Sub: 3
    , Push: 4
    , Message: 6
    , Error: 7
}

const connectType = {
    Basic: 1
    , Session: 2
}

function newProtocolUnit(requestID, unitType, bodyLength) {
    buffer = new ArrayBuffer(bodyLength + 6)
    view = new DataView(buffer)
    view.setUint16(0, unitType)
    view.setUint32(2, requestID)
    return new DataView(buffer, 6)
}

function encodeConnectRequest(namespace, credential, type) {
    enc = new TextEncoder()
    namespace = enc.encode(namespace)
    credential = enc.encode(credential)
    view = newProtocolUnit(0, protocolUnitType.Connect, 4 + namespace.byteLength + credential.byteLength)
    view.setUint16(0, namespace.byteLength)
    view.setUint16(2, credential.byteLength)
    console.log(namespace.map)
    return view.buffer
}

module.exports = {
    unitTypes: protocolUnitType
    , encodeConnectRequest: encodeConnectRequest
}
