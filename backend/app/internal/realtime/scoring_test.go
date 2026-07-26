package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoringFormula(t *testing.T) {
	assert.Equal(t, int64(280), CalculateGuessScore(2, 40))
	assert.Equal(t, int64(100), CalculateGuessScore(0, -1))
	assert.Equal(t, int64(75), CalculateDrawerBonus(3))
}

func TestSelectMVP(t *testing.T) {
	players := []PlayerState{
		{UserID: "early", Score: 100, CorrectGuesses: 1, SuccessfulDrawings: 0, JoinedAt: 1},
		{UserID: "winner", Score: 300, CorrectGuesses: 0, SuccessfulDrawings: 0, JoinedAt: 2},
		{UserID: "late", Score: 300, CorrectGuesses: 0, SuccessfulDrawings: 0, JoinedAt: 3},
	}
	assert.Equal(t, "winner", SelectMVP(players))

	players = []PlayerState{
		{UserID: "score", Score: 100, CorrectGuesses: 1, SuccessfulDrawings: 0, JoinedAt: 1},
		{UserID: "guesses", Score: 100, CorrectGuesses: 2, SuccessfulDrawings: 0, JoinedAt: 2},
	}
	assert.Equal(t, "guesses", SelectMVP(players))

	players = []PlayerState{
		{UserID: "guess", Score: 100, CorrectGuesses: 2, SuccessfulDrawings: 0, JoinedAt: 1},
		{UserID: "drawer", Score: 100, CorrectGuesses: 2, SuccessfulDrawings: 1, JoinedAt: 2},
	}
	assert.Equal(t, "drawer", SelectMVP(players))
}
