# Linker.js

LInker-IM Websocket client.



#### Build

```bash
npm install
npx webpack
```



#### Getting started

- Connect to server.

  ```json
  wsc = new linker.Client('ws://localhost:12360/ws')
  wsc.connect("myNamespacee", "myCredential")
  ```

- Subscribe

  ```json
  wsc.subscribe('room1')
  ```

