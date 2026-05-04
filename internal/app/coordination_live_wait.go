package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evanmschultz/tillsyn/internal/domain"
)

// waitForLiveEvent waits for one broker event and treats local wait timeouts as a
// normal no-op so list surfaces can return current state without failing closed.
func (s *Service) waitForLiveEvent(ctx context.Context, eventType LiveWaitEventType, key string, afterSequence int64, waitTimeout time.Duration) (bool, error) {
	if s == nil || s.liveWait == nil || waitTimeout <= 0 {
		return false, nil
	}
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()
	if _, err := s.liveWait.Wait(waitCtx, eventType, strings.TrimSpace(key), afterSequence); err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// liveWaitBaselineSequence returns the latest known monotonic sequence for one event key.
func (s *Service) liveWaitBaselineSequence(ctx context.Context, eventType LiveWaitEventType, key string) (int64, error) {
	if s == nil || s.liveWait == nil {
		return 0, nil
	}
	event, ok, err := s.liveWait.Latest(ctx, eventType, strings.TrimSpace(key))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	return event.Sequence, nil
}

// publishAttentionChanged wakes live waiters interested in project-scoped inbox changes.
func (s *Service) publishAttentionChanged(projectID string) {
	if s == nil || s.liveWait == nil {
		return
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return
	}
	s.liveWait.Publish(LiveWaitEvent{
		Type:  LiveWaitEventAttentionChanged,
		Key:   projectID,
		Value: projectID,
	})
}

// publishActionItemChanged wakes live waiters interested in project-scoped
// action-item changes. Drop 4a Wave 2.2 wires the cascade dispatcher's broker
// subscriber against this event; the wait key is the owning project ID so a
// single wakeup fans out across every dispatcher walking that project's tree.
func (s *Service) publishActionItemChanged(projectID string) {
	if s == nil || s.liveWait == nil {
		return
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return
	}
	s.liveWait.Publish(LiveWaitEvent{
		Type:  LiveWaitEventActionItemChanged,
		Key:   projectID,
		Value: projectID,
	})
}

// publishHandoffChanged wakes live waiters interested in project-scoped handoff changes.
func (s *Service) publishHandoffChanged(projectID string) {
	if s == nil || s.liveWait == nil {
		return
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return
	}
	s.liveWait.Publish(LiveWaitEvent{
		Type:  LiveWaitEventHandoffChanged,
		Key:   projectID,
		Value: projectID,
	})
}

// publishCommentChanged wakes live waiters interested in one thread target.
func (s *Service) publishCommentChanged(target domain.CommentTarget) {
	if s == nil || s.liveWait == nil {
		return
	}
	key := commentLiveWaitKey(target)
	if key == "" {
		return
	}
	s.liveWait.Publish(LiveWaitEvent{
		Type:  LiveWaitEventCommentChanged,
		Key:   key,
		Value: key,
	})
}

// commentLiveWaitKey returns one deterministic wait key for a comment thread target.
func commentLiveWaitKey(target domain.CommentTarget) string {
	target.ProjectID = strings.TrimSpace(target.ProjectID)
	target.TargetID = strings.TrimSpace(target.TargetID)
	target.TargetType = domain.CommentTargetType(strings.TrimSpace(string(target.TargetType)))
	if target.ProjectID == "" || target.TargetID == "" || target.TargetType == "" {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s", target.ProjectID, target.TargetType, target.TargetID)
}
