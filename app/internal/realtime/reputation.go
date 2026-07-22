package realtime

import (
	"context"
	"time"

	"github.com/google/uuid"

	"drawo/internal/core/domain"
	"drawo/internal/core/ports/repositories"
)

type reputationEvent struct {
	UserID string
	Delta  int64
	Reason string
}

type reputationLedger struct {
	profileRepo    repositories.ProfileRepository
	reputationRepo repositories.ReputationRepository
	roomID         string
	round          int
	events         []reputationEvent
	applied        map[string]int64
}

func newReputationLedger(profileRepo repositories.ProfileRepository, reputationRepo repositories.ReputationRepository, roomID string) *reputationLedger {
	return &reputationLedger{profileRepo: profileRepo, reputationRepo: reputationRepo, roomID: roomID, applied: make(map[string]int64)}
}

func (l *reputationLedger) setContext(roomID string, round int) {
	l.roomID = roomID
	l.round = round
}

func (l *reputationLedger) add(userID string, delta int64, reason string) {
	if userID == "" || delta == 0 {
		return
	}
	l.events = append(l.events, reputationEvent{UserID: userID, Delta: delta, Reason: reason})
	l.applied[userID] += delta
}

func (l *reputationLedger) addPositiveCapped(userID string, delta int64, reason string) {
	if delta <= 0 {
		return
	}
	current := l.applied[userID]
	if current >= maxPositiveRepPerGame {
		return
	}
	if current+delta > maxPositiveRepPerGame {
		delta = maxPositiveRepPerGame - current
	}
	l.add(userID, delta, reason)
}

func (l *reputationLedger) flush() {
	if len(l.events) == 0 {
		return
	}
	for _, event := range l.events {
		if l.reputationRepo != nil {
			_ = l.reputationRepo.InsertEvent(context.Background(), &domain.ReputationEvent{
				ID:        uuid.New().String(),
				UserID:    event.UserID,
				Delta:     event.Delta,
				Reason:    event.Reason,
				RoomID:    l.roomID,
				Round:     l.round,
				CreatedAt: time.Now(),
			})
		}
		if l.profileRepo == nil {
			continue
		}
		profile, err := l.profileRepo.GetByUserID(event.UserID)
		if err != nil || profile == nil {
			continue
		}
		profile.ReputationScore += event.Delta
		if profile.ReputationScore < 0 {
			profile.ReputationScore = 0
		}
		_ = l.profileRepo.Update(profile)
	}
	l.events = nil
}
