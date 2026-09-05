package gamelogic

import "testing"

func player(name string, units ...Unit) Player {
	p := Player{Username: name, Units: map[int]Unit{}}
	for i, u := range units {
		u.ID = i + 1
		p.Units[u.ID] = u
	}
	return p
}

func unit(rank UnitRank, loc Location) Unit {
	return Unit{Rank: rank, Location: loc}
}

func TestResolveWar(t *testing.T) {
	tests := []struct {
		name     string
		attacker Player
		defender Player
		wantWar  bool
		want     WarResult
	}{
		{
			name:     "no overlapping location",
			attacker: player("alice", unit(RankInfantry, "europe")),
			defender: player("bob", unit(RankInfantry, "asia")),
			wantWar:  false,
		},
		{
			name:     "attacker wins",
			attacker: player("alice", unit(RankArtillery, "europe")),
			defender: player("bob", unit(RankInfantry, "europe")),
			wantWar:  true,
			want:     WarResult{Location: "europe", Winner: "alice", Loser: "bob"},
		},
		{
			name:     "defender wins",
			attacker: player("alice", unit(RankInfantry, "europe")),
			defender: player("bob", unit(RankCavalry, "europe")),
			wantWar:  true,
			want:     WarResult{Location: "europe", Winner: "bob", Loser: "alice"},
		},
		{
			name:     "equal power is a draw",
			attacker: player("alice", unit(RankCavalry, "africa")),
			defender: player("bob", unit(RankInfantry, "africa"), unit(RankInfantry, "africa"), unit(RankInfantry, "africa"), unit(RankInfantry, "africa"), unit(RankInfantry, "africa")),
			wantWar:  true,
			want:     WarResult{Location: "africa", Winner: "alice", Loser: "bob", Draw: true},
		},
		{
			name:     "units outside the contested location do not count",
			attacker: player("alice", unit(RankInfantry, "asia"), unit(RankArtillery, "americas")),
			defender: player("bob", unit(RankCavalry, "asia")),
			wantWar:  true,
			want:     WarResult{Location: "asia", Winner: "bob", Loser: "alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveWar(tt.attacker, tt.defender)
			if ok != tt.wantWar {
				t.Fatalf("ResolveWar war=%v, want %v", ok, tt.wantWar)
			}
			if !tt.wantWar {
				return
			}
			if got != tt.want {
				t.Errorf("ResolveWar = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestHandleWarRemovesLosingUnits(t *testing.T) {
	attacker := player("alice", unit(RankArtillery, "europe"))
	defender := player("bob", unit(RankInfantry, "europe"), unit(RankInfantry, "asia"))

	gs := NewGameState("bob")
	for _, u := range defender.Units {
		gs.addUnit(u)
	}

	outcome, winner, loser := gs.HandleWar(RecognitionOfWar{Attacker: attacker, Defender: defender})
	if outcome != WarOutcomeOpponentWon {
		t.Fatalf("outcome = %v, want WarOutcomeOpponentWon", outcome)
	}
	if winner != "alice" || loser != "bob" {
		t.Errorf("winner/loser = %s/%s, want alice/bob", winner, loser)
	}

	remaining := gs.getUnitsSnap()
	if len(remaining) != 1 {
		t.Fatalf("remaining units = %d, want 1", len(remaining))
	}
	if remaining[0].Location != "asia" {
		t.Errorf("surviving unit in %s, want asia", remaining[0].Location)
	}
}
