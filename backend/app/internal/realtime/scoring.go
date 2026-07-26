package realtime

const (
	baseScoreMultiplier   = int64(100)
	speedBonusMultiplier  = int64(1)
	drawerBonusMultiplier = int64(25)
)

// CalculateGuessScore is the canonical scoring formula for a correct guess.
// It is intentionally simple and predictable for players:
//
//	score = word_points*100 + remaining_seconds*word_points
//
// Private/custom games can still display this in-room score, but only ranked
// public dictionary games should persist it to global statistics.
func CalculateGuessScore(wordPoints int, remainingSeconds int64) int64 {
	if wordPoints <= 0 {
		wordPoints = defaultWordPoints
	}
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	points := int64(wordPoints)
	return points*baseScoreMultiplier + remainingSeconds*points*speedBonusMultiplier
}

// CalculateDrawerBonus rewards the drawer when another player guesses the word.
func CalculateDrawerBonus(wordPoints int) int64 {
	if wordPoints <= 0 {
		wordPoints = defaultWordPoints
	}
	return int64(wordPoints) * drawerBonusMultiplier
}

// SelectMVP returns the user ID for the match MVP using a common score-first
// formula. Tie breakers are intentionally deterministic:
//  1. highest final score
//  2. most correct guesses
//  3. most successful drawings
//  4. earliest joined player
func SelectMVP(players []PlayerState) string {
	if len(players) == 0 {
		return ""
	}
	best := players[0]
	for _, candidate := range players[1:] {
		if candidate.Score > best.Score ||
			(candidate.Score == best.Score && candidate.CorrectGuesses > best.CorrectGuesses) ||
			(candidate.Score == best.Score && candidate.CorrectGuesses == best.CorrectGuesses && candidate.SuccessfulDrawings > best.SuccessfulDrawings) ||
			(candidate.Score == best.Score && candidate.CorrectGuesses == best.CorrectGuesses && candidate.SuccessfulDrawings == best.SuccessfulDrawings && candidate.JoinedAt < best.JoinedAt) {
			best = candidate
		}
	}
	return best.UserID
}
