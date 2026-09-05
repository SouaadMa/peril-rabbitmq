package world

import (
	"sort"
	"sync"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
)

type World struct {
	mu       sync.RWMutex
	players  map[string]playerState
	log      []LogEntry
	logLimit int
}

type playerState struct {
	player   gamelogic.Player
	lastSeen time.Time
}

func New(logLimit int) *World {
	return &World{
		players:  map[string]playerState{},
		logLimit: logLimit,
	}
}

func (w *World) ApplyPlayerState(p gamelogic.Player, at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.setPlayer(p, at)
}

func (w *World) ApplyMove(m gamelogic.ArmyMove, at time.Time) {
	w.ApplyPlayerState(m.Player, at)
}

func (w *World) ApplyWar(r gamelogic.RecognitionOfWar, at time.Time) (gamelogic.WarResult, bool) {
	result, fought := gamelogic.ResolveWar(r.Attacker, r.Defender)

	w.mu.Lock()
	defer w.mu.Unlock()

	w.setPlayer(r.Attacker, at)
	w.setPlayer(r.Defender, at)

	if !fought {
		return gamelogic.WarResult{}, false
	}

	w.removeUnitsInLocation(result.Loser, result.Location)
	if result.Draw {
		w.removeUnitsInLocation(result.Winner, result.Location)
	}

	return result, true
}

func (w *World) ApplyLog(entry LogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.log = append(w.log, entry)
	if len(w.log) > w.logLimit {
		w.log = append([]LogEntry(nil), w.log[len(w.log)-w.logLimit:]...)
	}
}

func (w *World) DropStale(now time.Time, ttl time.Duration) []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	dropped := []string{}
	for username, state := range w.players {
		if now.Sub(state.lastSeen) > ttl {
			delete(w.players, username)
			dropped = append(dropped, username)
		}
	}
	sort.Strings(dropped)
	return dropped
}

func (w *World) setPlayer(p gamelogic.Player, at time.Time) {
	units := make(map[int]gamelogic.Unit, len(p.Units))
	for id, unit := range p.Units {
		units[id] = unit
	}
	w.players[p.Username] = playerState{
		player:   gamelogic.Player{Username: p.Username, Units: units},
		lastSeen: at,
	}
}

func (w *World) removeUnitsInLocation(username string, location gamelogic.Location) {
	state, ok := w.players[username]
	if !ok {
		return
	}
	for id, unit := range state.player.Units {
		if unit.Location == location {
			delete(state.player.Units, id)
		}
	}
}
