# Peril

A small multiplayer strategy game where every player is a separate process and
the only thing connecting them is RabbitMQ. There's no central authority — each
client keeps its own army, publishes its moves, and reacts to everyone else's.

I built this to get properly comfortable with AMQP: exchanges, routing keys,
acknowledgements, prefetch, and dead-lettering.

## Running it

You'll need Go 1.22+ and Docker.

```sh
./rabbit.sh start
cp .env.example .env
```

Then start the server in one terminal:

```sh
go run ./cmd/server
```

...and a client in each of a few more:

```sh
go run ./cmd/client
```

Pick a username, then `spawn europe infantry`, `move asia 1`, `status`, `help`.
The server can `pause` and `resume` everyone. Management UI is at
http://localhost:15672 (guest/guest).

## How the messaging fits together

| Exchange       | Type   | Routing key           | Queue                           | Who consumes                |
| -------------- | ------ | --------------------- | ------------------------------- | --------------------------- |
| `peril_direct` | direct | `pause`               | `pause.<user>` (transient)      | every client                |
| `peril_topic`  | topic  | `army_moves.<user>`   | `army_moves.<user>` (transient) | every client                |
| `peril_topic`  | topic  | `war.<user>`          | `war.<user>` (transient)        | every client                |
| `peril_topic`  | topic  | `game_logs.<user>`    | `game_logs` (durable)           | the server                  |
| `peril_topic`  | topic  | `army_moves.*`        | `gateway_moves` (transient)     | the gateway                 |
| `peril_topic`  | topic  | `war.*`               | `gateway_wars` (transient)      | the gateway                 |
| `peril_topic`  | topic  | `player_state.<user>` | `gateway_state` (transient)     | the gateway                 |
| `peril_topic`  | topic  | `game_logs.*`         | `gateway_logs` (transient)      | the gateway                 |
| `peril_dlx`    | fanout | —                     | `peril_dlq` (durable)           | nobody, it's for inspection |

The gateway declares its own queues rather than sharing the players'. A topic
exchange copies each message into every queue whose binding matches, so the
gateway observes the whole game without taking a single message away from it.

## The gateway

`cmd/gateway` rebuilds the state of the world from those four streams and serves
it as JSON. Nothing publishes the board — it's a projection, reconstructed purely
from the events players emit.

```sh
go run ./cmd/gateway
curl -s localhost:8080/api/state | jq
```

Players heartbeat their whole army every few seconds, so the gateway can be
restarted at any point and refills within one interval. A player who stops
heartbeating is dropped after `PLAYER_TTL_SECONDS`.

## Credit

Peril is the project from [Boot.dev's Learn Pub/Sub course](https://www.boot.dev/courses/learn-pub-sub),
and `internal/gamelogic` is largely their starter code.
