// Linker Websocket example.
const linker = require('../../client/js')

wsc = new linker.Client('ws://localhost:8005/ws')
wsc.connect("bbb", "user1")
