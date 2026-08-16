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

const autoBanReputationThreshold = int64(3000)

type reputationLedger struct {
	profileRepo    repositories.ProfileRepository
	reputationRepo repositories.ReputationRepository
	userRepo       repositories.UserRepository
	sessionRepo    repositories.SessionRepository
	roomID         string
	round          int
	events         []reputationEvent
	applied        map[string]int64
}

func newReputationLedger(profileRepo repositories.ProfileRepository, reputationRepo repositories.ReputationRepository, roomID string, extraRepos ...interface{}) *reputationLedger {
	ledger := &reputationLedger{profileRepo: profileRepo, reputationRepo: reputationRepo, roomID: roomID, applied: make(map[string]int64)}
	for _, repo := range extraRepos {
		switch typed := repo.(type) {
		case repositories.UserRepository:
			ledger.userRepo = typed
		case repositories.SessionRepository:
			ledger.sessionRepo = typed
		}
	}
	return ledger
}

func (l *reputationLedger) setContext(roomID string, round int) {
	l.roomID = roomID
	l.round = round
}

func (l *reputationLedger) add(userID string, delta int64, reason string) {
	if userID == "" || delta == 0 {
		return
	}
	// Guests are ephemeral: they have no users/profiles rows and their
	// "guest:<uuid>" IDs are not valid UUIDs, so persisting reputation for
	// them is impossible (SQLSTATE 22P02) and meaningless — a new invite
	// join is a brand-new identity anyway.
	if domain.IsGuestID(userID) {
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
		if profile.ReputationScore < autoBanReputationThreshold {
			l.autoBan(profile.UserID)
		}
	}
	l.events = nil
}

func (l *reputationLedger) autoBan(userID string) {
	if l.userRepo != nil {
		if user, err := l.userRepo.GetByID(userID); err == nil && user != nil && user.IsActive {
			now := time.Now()
			user.IsActive = false
			user.Status = domain.AccountStatusBanned
			user.BanCount++
			user.BannedAt = &now
			user.UpdatedAt = now
			_ = l.userRepo.Update(user)
		}
	}
	if l.sessionRepo != nil {
		_ = l.sessionRepo.DeleteAllForUser(context.Background(), userID)
	}
}
