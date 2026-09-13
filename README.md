# Peril

A small multiplayer strategy game where every player is a separate process and
the only thing connecting them is RabbitMQ. There's no central authority — each
client keeps its own army, publishes its moves, and reacts to everyone else's.
A gateway watches the whole thing and streams it to a live world map in the
browser.

I built this to get properly comfortable with AMQP: exchanges, routing keys,
acknowledgements, prefetch, and dead-lettering — and then to try some frontend :D

## Running it

You'll need Go 1.22+, Docker, and Node 22+ for the map.

```sh
./rabbit.sh start
cp .env.example .env
```

Then, each in its own terminal:

```sh
go run ./cmd/server
go run ./cmd/gateway
go run ./cmd/client
```

Start a couple of clients. Pick a username, then `spawn europe infantry`,
`move europe 1`, `status`, `help`. The server can `pause` and `resume` everyone.
Management UI is at http://localhost:15672 (guest/guest).

And the map:

```sh
cd web
npm install
npm run dev
```

Open http://localhost:5173 and watch continents change hands as players go into war.

## The gateway

`cmd/gateway` rebuilds the state of the world from those four streams and serves
it as JSON. Nothing publishes the board — it's a projection, reconstructed purely
from the events players emit.

```sh
curl -s localhost:8090/api/state | jq
```

`GET /ws` streams the same shape over a WebSocket: one snapshot immediately on
connect, then another whenever the world changes. Updates are coalesced onto a
100ms tick, so a burst of events costs one frame rather than one per message, and
a browser that stops reading is dropped instead of stalling the broadcaster.

Players heartbeat their whole army every few seconds, so the gateway can be
restarted at any point and refills within one interval. A player who stops
heartbeating is dropped after `PLAYER_TTL_SECONDS`.

## The map

`web/` is a React + TypeScript app built with Vite. In development, Vite proxies
`/api` and `/ws` to the gateway, so the browser only ever talks to one origin.

Each continent takes the colour of whoever has the most power there, and every
region is labelled with its occupants.

## Credit

Peril is the project from [Boot.dev's Learn Pub/Sub course](https://www.boot.dev/courses/learn-pub-sub),
and `internal/gamelogic` is largely their starter code.

---

## How the messaging fits together

| Exchange       | Type   | Routing key         | Queue                           | Who consumes                |
| -------------- | ------ | ------------------- | ------------------------------- | --------------------------- |
| `peril_direct` | direct | `pause`             | `pause.<user>` (transient)      | every client                |
| `peril_topic`  | topic  | `army_moves.<user>` | `army_moves.<user>` (transient) | every client                |
| `peril_topic`  | topic  | `war.<user>`        | `war.<user>` (transient)        | every client                |
| `peril_topic`  | topic  | `game_logs.<user>`  | `game_logs` (durable)           | the server                  |
| `peril_topic`  | topic  | `army_moves.*`      | `gateway_moves` (transient)     | the gateway                 |
| `peril_topic`  | topic  | `war.*`             | `gateway_wars` (transient)      | the gateway                 |
| `peril_topic`  | topic  | `player_state.*`    | `gateway_state` (transient)     | the gateway                 |
| `peril_topic`  | topic  | `game_logs.*`       | `gateway_logs` (transient)      | the gateway                 |
| `peril_dlx`    | fanout | —                   | `peril_dlq` (durable)           | nobody, it's for inspection |

The gateway declares its own queues rather than sharing the players'. A topic
exchange copies each message into every queue whose binding matches, so the
gateway observes the whole game without taking a single message away from it.
