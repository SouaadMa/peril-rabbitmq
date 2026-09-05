package gamelogic

type Player struct {
	Username string
	Units    map[int]Unit
}

type UnitRank string

const (
	RankInfantry  UnitRank = "infantry"
	RankCavalry   UnitRank = "cavalry"
	RankArtillery UnitRank = "artillery"
)

type Unit struct {
	ID       int
	Rank     UnitRank
	Location Location
}

type ArmyMove struct {
	Player     Player
	Units      []Unit
	ToLocation Location
}

type RecognitionOfWar struct {
	Attacker Player
	Defender Player
}

type Location string

func AllLocations() []Location {
	return []Location{
		"americas",
		"europe",
		"africa",
		"asia",
		"australia",
		"antarctica",
	}
}

func getAllRanks() map[UnitRank]struct{} {
	return map[UnitRank]struct{}{
		RankInfantry:  {},
		RankCavalry:   {},
		RankArtillery: {},
	}
}

func getAllLocations() map[Location]struct{} {
	locations := map[Location]struct{}{}
	for _, location := range AllLocations() {
		locations[location] = struct{}{}
	}
	return locations
}
