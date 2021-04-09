var through2 = require('through2').obj;
window.onload = function () {
    const term = new Terminal({
        cols: 190,
        rows: 40,
        useStyle: true,
        screenKeys: true,
        cursorBlink: true
    });

    term.on('title', function(title) {
        document.title = title;
    });

    term.open(document.body);

    const socket = new WebSocket("ws://localhost:8080/ws")
    socket.binaryType = "arraybuffer"

    socket.addEventListener('open', function (event) {
        console.log('Opened websocket')
    });

    let stdin = through2((data, enc, cb) => {
        socket.send(Buffer.concat([new Buffer([0]), new Buffer(data)]))
        cb();
    }, (cb) => {});

    let stdout = through2();
    let stderr = through2();

    stdout.on('data', function (data) {
        term.write(String.fromCharCode.apply(null, data));
    });
    stderr.on('data', function (data) {
        term.write(String.fromCharCode.apply(null, data));
    });


    socket.addEventListener('message', function (event) {
        let message = new Buffer(new Uint8Array(event.data));
        switch (message[0]) {
            case 1:
                stdout.write(message.slice(1))
                break;
            case 2:
                stderr.write(message.slice(1))
                break;
        }
    });

    // as soon as the terminal receives any data, it writes it on the terminal.
    term.on('data', function(data) {
        stdin.write(data)
    });
}