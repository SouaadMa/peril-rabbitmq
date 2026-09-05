package gamelogic

import (
	"fmt"
)

type WarOutcome int

const (
	WarOutcomeNotInvolved WarOutcome = iota
	WarOutcomeNoUnits
	WarOutcomeYouWon
	WarOutcomeOpponentWon
	WarOutcomeDraw
)

type WarResult struct {
	Location Location
	Winner   string
	Loser    string
	Draw     bool
}

func ResolveWar(attacker, defender Player) (WarResult, bool) {
	location := getOverlappingLocation(attacker, defender)
	if location == "" {
		return WarResult{}, false
	}

	attackerPower := PowerLevel(unitsInLocation(attacker, location))
	defenderPower := PowerLevel(unitsInLocation(defender, location))

	result := WarResult{
		Location: location,
		Winner:   attacker.Username,
		Loser:    defender.Username,
	}
	switch {
	case attackerPower > defenderPower:
	case defenderPower > attackerPower:
		result.Winner = defender.Username
		result.Loser = attacker.Username
	default:
		result.Draw = true
	}
	return result, true
}

func (gs *GameState) HandleWar(rw RecognitionOfWar) (outcome WarOutcome, winner string, loser string) {
	defer fmt.Println("------------------------")
	fmt.Println()
	fmt.Println("==== War Declared ====")
	fmt.Printf("%s has declared war on %s!\n", rw.Attacker.Username, rw.Defender.Username)

	username := gs.GetUsername()
	if username != rw.Attacker.Username && username != rw.Defender.Username {
		fmt.Printf("%s, you are not involved in this war.\n", username)
		return WarOutcomeNotInvolved, "", ""
	}

	result, ok := ResolveWar(rw.Attacker, rw.Defender)
	if !ok {
		fmt.Println("Error! No units are in the same location. No war will be fought.")
		return WarOutcomeNoUnits, "", ""
	}

	if result.Draw {
		fmt.Printf("The war in %s ended in a draw!\n", result.Location)
		fmt.Printf("Your units in %s have been killed.\n", result.Location)
		gs.removeUnitsInLocation(result.Location)
		return WarOutcomeDraw, result.Winner, result.Loser
	}

	fmt.Printf("%s has won the war in %s!\n", result.Winner, result.Location)
	if result.Loser == username {
		fmt.Println("You have lost the war!")
		fmt.Printf("Your units in %s have been killed.\n", result.Location)
		gs.removeUnitsInLocation(result.Location)
		return WarOutcomeOpponentWon, result.Winner, result.Loser
	}
	return WarOutcomeYouWon, result.Winner, result.Loser
}

func unitsInLocation(p Player, loc Location) []Unit {
	units := []Unit{}
	for _, unit := range p.Units {
		if unit.Location == loc {
			units = append(units, unit)
		}
	}
	return units
}

func PowerLevel(units []Unit) int {
	power := 0
	for _, unit := range units {
		switch unit.Rank {
		case RankArtillery:
			power += 10
		case RankCavalry:
			power += 5
		case RankInfantry:
			power += 1
		}
	}
	return power
}
