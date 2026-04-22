# clientrpc Examples

The following `clientrpc` examples are provided:

  - [version](version/): Most basic example showing how to connect to a running
    `clientrpc` instance to query its version.
  - [multistreams](multistreams/): Example showing how to request multiple
    streams from the server.
  - [chat](chat/): Example showing how to read chat messages from stdin and
    output received messages to stdout.
  - [rtdtchat](rtdtchat/): Example showing how to join an RTDT session and read
    and write RTDT chat messages while capturing microphone audio and playing
    remote RTDT audio through `clientrpc`.
  - [rtdtaudio](rtdtaudio/): Example showing how an external app can capture
    microphone audio and send it to a live RTDT session in `brclient`, with
    optional remote-audio playback.

## `rtdtaudio` Quick Test

To test the RPC audio path against a running `brclient`:

1. Create or join an RTDT session in `brclient`.
2. Run `/realtimechat info <session-prefix>` and copy the full `RV:` value.
3. Run the example against the local `clientrpc` endpoint:

```bash
go run ./clientrpc/examples/rtdtaudio \
  -session <full-session-rv> \
  -url wss://127.0.0.1:7777/ws \
  -servercert /path/to/rpc.cert \
  -clientcert /path/to/rpc-client.cert \
  -clientkey /path/to/rpc-client.key \
  -rpcuser rpcuser \
  -rpcpass rpcpass
```

If the target `brclient` config uses a different RPC listen address or cert
paths, adjust the command accordingly.

To also receive and play remote RTDT audio through the example app, add:

```bash
-playaudio
```

To list local audio devices and quit:

```bash
go run ./clientrpc/examples/rtdtaudio -lsdev
```
