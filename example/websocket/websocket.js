// Linker Websocket example.
const linker = require('../../client/js')

window.addEventListener("load", function () {
    wsc = new linker.Client('ws://localhost:12360/ws')
    wsc.connect("bbb", "user1")
})
