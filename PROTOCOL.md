# Protocol Specification

This is the protocol specification over WebSockets for streaming n numbers to a client from a server.

_A future improvement would be to provide support over gRPC or SSE._

## Establishing a connection

### The Client

The client must establish a connection to the server providing specific parameters as query string parameters in the initial HTTP handshake that will be upgraded to WebSockets.

The format is the following:

```
ws(s)://{host}:{port}?clientId={uuid}&sequenceCount={n}&lastReceived={n}
```

Example for an initial connection:

`ws://localhost:4039?clientId=7fe2d7a5-c044-41da-bbae-7a86870f6562&sequenceCount=1000`

Example for a re-connection:

`ws://localhost:4039?clientId=7fe2d7a5-c044-41da-bbae-7a86870f6562&sequenceCount=1000&lastReceived=50`

#### Client ID

`clientId` (query string)

**required**

The ID to use to identify a client session that the server will use to keep track of progress
and be able to continue when a client disconnects and re-connects.

#### Sequence Count

`sequenceCount` (query string, default = random number between 0x1 and 0xffff)

**optional**

The number of messages that the client expects to receive from the server where the primary payload from the server message is a pseudo-randomly generated number.

#### Last Received Index

`lastReceived` (query string)

**optional**

The last received number in the sequence that the client received before disconnecting.
This is optional, the server will use acknowledgements stored in session state to determine where
to continue from if this is not provided.
When provided, this will be treated as the source of truth by the server and acknowledgements in session
state will not be used.

### The Server

Upon receiving a client identifier, an optional sequence count and last received index, the server must initialise a pseudo-random sequence of numbers of `sequenceCount` numbers and store in persistent state keyed by the provided `clientId` that lives through multiple connections up to a pre-configured deadline during a period of disconnection.

The server must upgrade the connection from HTTP to Sockets to establish a long-lived connection allowing for low latency delivery of messages from the server to client along with bi-directional communication via the underlying WebSockets protocol.

## Sequence Delivery & Acknowledgements

### Server

Once the connection has been upgraded and the server has initialised the sequence of numbers, it must begin delivering each number in the sequence at a pre-configured interval.

The format of the message for all but the last number in the sequence is as follows:

```
[NumberInSequencePrefix][number]
```

(e.g. `0x10xffff`)

The format of the message representing the last number in the sequence is the following:

```
[LastNumberInSequencePrefix]{"number":[lastNumberInSequence],"checksum":[checksumOfSequence]}
```

(e.g. `0x3{"number":430,"checksum":"4945twwe9r8wery2srfsdf8425897fsfsdfsdfwer2345wdrwdf=sadsa3das"}`)

See [Message Prefixes](#message-prefixes) for the prefix name to value mapping.

checksumOfSequence is a SHA1 checksum of the serialised JSON array representation of the sequence.

The server must also handle acknowledgements from the client for every number in the sequence by updating session state to reflect that a particular number in the sequence has been acknowledged.

In this version of the protocol acknowledgements are not used for any server-side retry functionality for individual messages, it is used as a part of the strategy for handling client re-connections.

### Client

Upon receiving a message from the server, the client must parse the message based on the [Message Prefix](#message-prefixes).

Once this is complete, the client must send an acknowledgement to the server that it has received the number in the sequence to help the server in ensuring the sequence is delivered when there are disconnections.

## Sequence Verification

### Client

Once the final message in the sequence has been received, the client must create a SHA1 checksum of the serialised JSON array representation of the sequence and compare with the checksum to determine success or failure in receiving the sequence of numbers expected.

## Re-connecting

### Client

When the client disconnects, it must re-connect as soon as possible within the parameters of a retry/backoff strategy.

The client must use an exponential backoff strategy or similar when attempting to reconnect to the server to avoid a busy loop.

The client can optionally provide a `lastReceived` query string parameter in the connection URL when re-connecting. This should hold the last recieved index in the sequence.
When `lastReceived` is not provided by the client, the server will decide where to pick up based on the acknowledgements received from the client.

### Server

The server is responsible for utilising the information it has stored in session state to continue to deliver messages to the client.

Upon receiving a `lastReceived` query parameter as part of the re-connection, the server will use `lastReceived + 1` as the starting index to deliver the rest of the sequence, otherwise it will look for the first number in session state that does not have an acknowledgement.

_As the nature of message delivery on a WebSocket over TCP connection is sequential, the challenge of missing out messages when looking at acknowledgements is not present. However, if this protocol was to be expanded to support underlying protocols that do not have these guarantees, more resilience would need to be added to the protocol._

## Message Prefixes

- NumberInSequencePrefix (0x1) - A number in a sequence sent from the server to the client.
- AcknowledgementPrefix (0x2) - An acknowledgement from the client to the server that a number in the sequence has been received by client.
- LastNumberInSequencePrefix (0x3) - The message containing the final number in the sequence along with a checksum.
