# Configuration

This doc outlines the configuration available for the server and client.

## Client

Configuration for the client is provided in `.env.client` as a list of environment variables exported to the environment by the client application.

### Send Last Received Index

`SEND_LAST_RECEIVED_INDEX`

**optional, (default = false, one of 0, 1, true or false)**

Whether or not the last received index should be sent when re-connecting to the server.

### Max Reconnection Attempts

`MAX_RECONNECTION_ATTEMPTS`

**optional, (default = 100)**

The maximum number of reconnection attempts the client can make to the server in a period of disconnection.

### Log Level

`LOG_LEVEL`

**optional, (default = "info", one of "trace", "debug", "info", "warn", "error")**

The log level to be used for the logger used within the client app.

## Server

Configuration for the server is provided in `.env.server` as a list of environment variables exported to the environment by the server application.

### Sequence Message Interval

`SEQUENCE_MESSAGE_INTERVAL`

**optional, (default = 1000)**

The number of milliseconds the server should wait between each number sent in a sequence of messages.

### Session State Expiry

`SESSION_STATE_IDLE_TIME_EXPIRY`

**optional, (default = 30)**

The number of seconds that can pass during a period of disconnection before expiring/discarding session state for a client.

### Log Level

`LOG_LEVEL`

**optional, (default = "info", one of "trace", "debug", "info", "warn", "error")**

The log level to be used for the logger used within the server app.
