package world

import (
	"sort"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
)

type Snapshot struct {
	Players   []PlayerView   `json:"players"`
	Locations []LocationView `json:"locations"`
	Log       []LogEntry     `json:"log"`
}

type PlayerView struct {
	Username string     `json:"username"`
	Units    []UnitView `json:"units"`
	LastSeen time.Time  `json:"lastSeen"`
}

type UnitView struct {
	ID       int    `json:"id"`
	Rank     string `json:"rank"`
	Location string `json:"location"`
}

type LocationView struct {
	Name      string     `json:"name"`
	Occupants []Occupant `json:"occupants"`
}

type Occupant struct {
	Username string `json:"username"`
	Units    int    `json:"units"`
	Power    int    `json:"power"`
}

type LogEntry struct {
	Time     time.Time `json:"time"`
	Username string    `json:"username"`
	Message  string    `json:"message"`
}

func (w *World) Snapshot() Snapshot {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return Snapshot{
		Players:   w.playerViews(),
		Locations: w.locationViews(),
		Log:       append([]LogEntry(nil), w.log...),
	}
}

func (w *World) playerViews() []PlayerView {
	views := make([]PlayerView, 0, len(w.players))
	for username, state := range w.players {
		units := make([]UnitView, 0, len(state.player.Units))
		for _, unit := range state.player.Units {
			units = append(units, UnitView{
				ID:       unit.ID,
				Rank:     string(unit.Rank),
				Location: string(unit.Location),
			})
		}
		sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })

		views = append(views, PlayerView{
			Username: username,
			Units:    units,
			LastSeen: state.lastSeen,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Username < views[j].Username })
	return views
}

func (w *World) locationViews() []LocationView {
	grouped := map[gamelogic.Location]map[string][]gamelogic.Unit{}
	for username, state := range w.players {
		for _, unit := range state.player.Units {
			if grouped[unit.Location] == nil {
				grouped[unit.Location] = map[string][]gamelogic.Unit{}
			}
			grouped[unit.Location][username] = append(grouped[unit.Location][username], unit)
		}
	}

	locations := gamelogic.AllLocations()
	views := make([]LocationView, 0, len(locations))
	for _, location := range locations {
		occupants := make([]Occupant, 0, len(grouped[location]))
		for username, units := range grouped[location] {
			occupants = append(occupants, Occupant{
				Username: username,
				Units:    len(units),
				Power:    gamelogic.PowerLevel(units),
			})
		}
		sort.Slice(occupants, func(i, j int) bool { return occupants[i].Username < occupants[j].Username })

		views = append(views, LocationView{
			Name:      string(location),
			Occupants: occupants,
		})
	}
	return views
}
