package task

import "time"

type TaskStatus string

const (
	TaskStopped TaskStatus = "stopped"
	TaskRunning TaskStatus = "running"
	TaskFailed  TaskStatus = "failed"
	TaskDone    TaskStatus = "done"
)

type TaskMode string

const (
	TaskModeRace TaskMode = "race"
)

type Task struct {
	ID         string
	Title      string
	Mode       TaskMode
	Status     TaskStatus
	SessionIDs []string
	CreatedAt  time.Time
}
