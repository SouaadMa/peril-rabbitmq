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
| `peril_topic`  | topic  | `player_state.<user>` | —                               | the gateway, eventually     |
| `peril_topic`  | topic  | `game_logs.<user>`    | `game_logs` (durable)           | the server                  |
| `peril_dlx`    | fanout | —                     | `peril_dlq` (durable)           | nobody, it's for inspection |

## Credit

Peril is the project from [Boot.dev's Learn Pub/Sub course](https://www.boot.dev/courses/learn-pub-sub),
and `internal/gamelogic` is largely their starter code.
