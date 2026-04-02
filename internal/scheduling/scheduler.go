// Package scheduling provides command scheduling capabilities
package scheduling

import (
	"fmt"
	"sync"
	"time"
)

// ScheduledCommand represents a scheduled command
type ScheduledCommand struct {
	ID        string
	Command   string
	Args      []string
	Schedule  Schedule
	Enabled   bool
	LastRun   time.Time
	NextRun   time.Time
	RunCount  int
	CreatedAt time.Time
}

// Schedule represents a schedule
type Schedule struct {
	Type     ScheduleType
	Cron     string
	Interval time.Duration
	At       time.Time
}

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleOnce     ScheduleType = "once"
	ScheduleInterval ScheduleType = "interval"
	ScheduleDaily    ScheduleType = "daily"
	ScheduleWeekly   ScheduleType = "weekly"
	ScheduleCron     ScheduleType = "cron"
)

// Scheduler manages scheduled commands
type Scheduler struct {
	commands map[string]*ScheduledCommand
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		commands: make(map[string]*ScheduledCommand),
		stopCh:   make(chan struct{}),
	}
}

// AddCommand adds a scheduled command
func (s *Scheduler) AddCommand(cmd *ScheduledCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cmd.ID == "" {
		cmd.ID = fmt.Sprintf("sched-%d", time.Now().UnixNano())
	}
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}

	cmd.NextRun = s.calculateNextRun(cmd.Schedule)
	s.commands[cmd.ID] = cmd

	return nil
}

// RemoveCommand removes a scheduled command
func (s *Scheduler) RemoveCommand(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.commands[id]; !ok {
		return fmt.Errorf("command not found: %s", id)
	}

	delete(s.commands, id)
	return nil
}

// ListCommands returns all scheduled commands
func (s *Scheduler) ListCommands() []*ScheduledCommand {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cmds := make([]*ScheduledCommand, 0, len(s.commands))
	for _, cmd := range s.commands {
		cmds = append(cmds, cmd)
	}

	return cmds
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopCh)
	s.stopCh = make(chan struct{})
}

func (s *Scheduler) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndRun()
		}
	}
}

func (s *Scheduler) checkAndRun() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, cmd := range s.commands {
		if !cmd.Enabled {
			continue
		}

		if now.After(cmd.NextRun) {
			// Would execute command here
			cmd.LastRun = now
			cmd.RunCount++
			cmd.NextRun = s.calculateNextRun(cmd.Schedule)
		}
	}
}

func (s *Scheduler) calculateNextRun(schedule Schedule) time.Time {
	now := time.Now()

	switch schedule.Type {
	case ScheduleOnce:
		return schedule.At
	case ScheduleInterval:
		return now.Add(schedule.Interval)
	case ScheduleDaily:
		return now.Add(24 * time.Hour)
	case ScheduleWeekly:
		return now.Add(7 * 24 * time.Hour)
	default:
		return now.Add(1 * time.Hour)
	}
}
