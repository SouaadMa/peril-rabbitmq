package world

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
)

var t0 = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func player(name string, units ...gamelogic.Unit) gamelogic.Player {
	p := gamelogic.Player{Username: name, Units: map[int]gamelogic.Unit{}}
	for i, u := range units {
		u.ID = i + 1
		p.Units[u.ID] = u
	}
	return p
}

func unit(rank gamelogic.UnitRank, location gamelogic.Location) gamelogic.Unit {
	return gamelogic.Unit{Rank: rank, Location: location}
}

func playerView(t *testing.T, s Snapshot, username string) PlayerView {
	t.Helper()
	for _, view := range s.Players {
		if view.Username == username {
			return view
		}
	}
	t.Fatalf("player %q not in snapshot", username)
	return PlayerView{}
}

func locationView(t *testing.T, s Snapshot, name string) LocationView {
	t.Helper()
	for _, view := range s.Locations {
		if view.Name == name {
			return view
		}
	}
	t.Fatalf("location %q not in snapshot", name)
	return LocationView{}
}

func TestApplyPlayerStateReplacesArmyWholesale(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankInfantry, "europe"), unit(gamelogic.RankCavalry, "asia")), t0)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankCavalry, "asia")), t0.Add(time.Second))

	view := playerView(t, w.Snapshot(), "alice")
	if len(view.Units) != 1 {
		t.Fatalf("units = %d, want 1", len(view.Units))
	}
	if view.Units[0].Location != "asia" {
		t.Errorf("location = %s, want asia", view.Units[0].Location)
	}
	if !view.LastSeen.Equal(t0.Add(time.Second)) {
		t.Errorf("lastSeen = %v, want %v", view.LastSeen, t0.Add(time.Second))
	}
}

func TestApplyMoveUpdatesPosition(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankInfantry, "europe")), t0)

	moved := player("alice", unit(gamelogic.RankInfantry, "australia"))
	w.ApplyMove(gamelogic.ArmyMove{Player: moved, ToLocation: "australia"}, t0.Add(time.Second))

	view := playerView(t, w.Snapshot(), "alice")
	if view.Units[0].Location != "australia" {
		t.Errorf("location = %s, want australia", view.Units[0].Location)
	}
}

func TestApplyWarRemovesLoserUnitsInContestedLocationOnly(t *testing.T) {
	w := New(10)
	attacker := player("alice", unit(gamelogic.RankArtillery, "europe"))
	defender := player("bob", unit(gamelogic.RankInfantry, "europe"), unit(gamelogic.RankInfantry, "asia"))

	result, fought := w.ApplyWar(gamelogic.RecognitionOfWar{Attacker: attacker, Defender: defender}, t0)
	if !fought {
		t.Fatal("fought = false, want true")
	}
	if result.Winner != "alice" || result.Loser != "bob" {
		t.Errorf("winner/loser = %s/%s, want alice/bob", result.Winner, result.Loser)
	}

	snap := w.Snapshot()
	bob := playerView(t, snap, "bob")
	if len(bob.Units) != 1 {
		t.Fatalf("bob units = %d, want 1", len(bob.Units))
	}
	if bob.Units[0].Location != "asia" {
		t.Errorf("bob survivor in %s, want asia", bob.Units[0].Location)
	}
	if alice := playerView(t, snap, "alice"); len(alice.Units) != 1 {
		t.Errorf("alice units = %d, want 1", len(alice.Units))
	}
}

func TestApplyWarDrawRemovesBothSides(t *testing.T) {
	w := New(10)
	attacker := player("alice", unit(gamelogic.RankCavalry, "africa"))
	defender := player("bob",
		unit(gamelogic.RankInfantry, "africa"),
		unit(gamelogic.RankInfantry, "africa"),
		unit(gamelogic.RankInfantry, "africa"),
		unit(gamelogic.RankInfantry, "africa"),
		unit(gamelogic.RankInfantry, "africa"),
	)

	result, fought := w.ApplyWar(gamelogic.RecognitionOfWar{Attacker: attacker, Defender: defender}, t0)
	if !fought || !result.Draw {
		t.Fatalf("fought=%v draw=%v, want true/true", fought, result.Draw)
	}

	snap := w.Snapshot()
	if got := playerView(t, snap, "alice"); len(got.Units) != 0 {
		t.Errorf("alice units = %d, want 0", len(got.Units))
	}
	if got := playerView(t, snap, "bob"); len(got.Units) != 0 {
		t.Errorf("bob units = %d, want 0", len(got.Units))
	}
}

func TestApplyWarWithoutOverlapLeavesArmiesIntact(t *testing.T) {
	w := New(10)
	attacker := player("alice", unit(gamelogic.RankArtillery, "europe"))
	defender := player("bob", unit(gamelogic.RankInfantry, "asia"))

	result, fought := w.ApplyWar(gamelogic.RecognitionOfWar{Attacker: attacker, Defender: defender}, t0)
	if fought {
		t.Fatal("fought = true, want false")
	}
	if result != (gamelogic.WarResult{}) {
		t.Errorf("result = %+v, want zero value", result)
	}

	snap := w.Snapshot()
	if got := playerView(t, snap, "alice"); len(got.Units) != 1 {
		t.Errorf("alice units = %d, want 1", len(got.Units))
	}
	if got := playerView(t, snap, "bob"); len(got.Units) != 1 {
		t.Errorf("bob units = %d, want 1", len(got.Units))
	}
}

func TestDropStale(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankInfantry, "asia")), t0)
	w.ApplyPlayerState(player("bob", unit(gamelogic.RankInfantry, "asia")), t0.Add(10*time.Second))
	w.ApplyPlayerState(player("carol", unit(gamelogic.RankInfantry, "asia")), t0.Add(12*time.Second))

	dropped := w.DropStale(t0.Add(15*time.Second), 10*time.Second)
	if !reflect.DeepEqual(dropped, []string{"alice"}) {
		t.Fatalf("dropped = %v, want [alice]", dropped)
	}

	snap := w.Snapshot()
	if len(snap.Players) != 2 {
		t.Fatalf("players = %d, want 2", len(snap.Players))
	}
	if snap.Players[0].Username != "bob" || snap.Players[1].Username != "carol" {
		t.Errorf("players = %s, want [bob carol]", []string{snap.Players[0].Username, snap.Players[1].Username})
	}
}

func TestApplyLogKeepsNewestWithinLimit(t *testing.T) {
	w := New(3)
	for i := 1; i <= 5; i++ {
		w.ApplyLog(LogEntry{Time: t0, Username: "alice", Message: fmt.Sprintf("m%d", i)})
	}

	log := w.Snapshot().Log
	if len(log) != 3 {
		t.Fatalf("log = %d entries, want 3", len(log))
	}
	want := []string{"m3", "m4", "m5"}
	for i, entry := range log {
		if entry.Message != want[i] {
			t.Errorf("log[%d] = %s, want %s", i, entry.Message, want[i])
		}
	}
}

func TestLocationViewsGroupAndComputePower(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankArtillery, "europe"), unit(gamelogic.RankInfantry, "europe")), t0)
	w.ApplyPlayerState(player("bob", unit(gamelogic.RankCavalry, "europe")), t0)

	snap := w.Snapshot()

	names := make([]string, 0, len(snap.Locations))
	for _, view := range snap.Locations {
		names = append(names, view.Name)
	}
	wantNames := make([]string, 0, len(gamelogic.AllLocations()))
	for _, location := range gamelogic.AllLocations() {
		wantNames = append(wantNames, string(location))
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("locations = %v, want %v", names, wantNames)
	}

	europe := locationView(t, snap, "europe")
	want := []Occupant{
		{Username: "alice", Units: 2, Power: 11},
		{Username: "bob", Units: 1, Power: 5},
	}
	if !reflect.DeepEqual(europe.Occupants, want) {
		t.Errorf("europe occupants = %+v, want %+v", europe.Occupants, want)
	}

	if got := locationView(t, snap, "asia"); len(got.Occupants) != 0 {
		t.Errorf("asia occupants = %d, want 0", len(got.Occupants))
	}
}

func TestSnapshotIsDeterministic(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("carol", unit(gamelogic.RankInfantry, "asia")), t0)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankCavalry, "europe")), t0)
	w.ApplyPlayerState(player("bob", unit(gamelogic.RankArtillery, "africa")), t0)
	w.ApplyLog(LogEntry{Time: t0, Username: "alice", Message: "hello"})

	if first, second := w.Snapshot(), w.Snapshot(); !reflect.DeepEqual(first, second) {
		t.Errorf("snapshots differ:\n%+v\n%+v", first, second)
	}
}

func TestSnapshotDoesNotAliasWorld(t *testing.T) {
	w := New(10)
	w.ApplyPlayerState(player("alice", unit(gamelogic.RankInfantry, "europe")), t0)
	w.ApplyLog(LogEntry{Time: t0, Username: "alice", Message: "original"})

	snap := w.Snapshot()
	snap.Players[0].Units[0].Location = "mars"
	snap.Players[0].Username = "mallory"
	snap.Log[0].Message = "tampered"
	snap.Log = append(snap.Log, LogEntry{Message: "injected"})
	snap.Locations[0].Occupants = nil

	after := w.Snapshot()
	if got := playerView(t, after, "alice"); got.Units[0].Location != "europe" {
		t.Errorf("location = %s, want europe", got.Units[0].Location)
	}
	if len(after.Log) != 1 {
		t.Fatalf("log = %d entries, want 1", len(after.Log))
	}
	if after.Log[0].Message != "original" {
		t.Errorf("log message = %s, want original", after.Log[0].Message)
	}
}

func TestConcurrentApplyAndSnapshot(t *testing.T) {
	w := New(50)
	var wg sync.WaitGroup

	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("p%d", i)
			for j := range 200 {
				at := t0.Add(time.Duration(j) * time.Second)
				w.ApplyPlayerState(player(name, unit(gamelogic.RankInfantry, "asia")), at)
				w.ApplyLog(LogEntry{Time: at, Username: name, Message: "spam"})
				w.ApplyWar(gamelogic.RecognitionOfWar{
					Attacker: player(name, unit(gamelogic.RankArtillery, "europe")),
					Defender: player("victim", unit(gamelogic.RankInfantry, "europe")),
				}, at)
			}
		}(i)
	}

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				snap := w.Snapshot()
				for _, view := range snap.Players {
					_ = len(view.Units)
				}
				w.DropStale(t0.Add(time.Hour), time.Hour)
			}
		}()
	}

	wg.Wait()
}
