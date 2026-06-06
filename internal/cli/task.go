package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/handyfun97/ottrta/internal/session"
	"github.com/handyfun97/ottrta/internal/task"
)

func newTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newTaskStartCommand())
	cmd.AddCommand(newTaskListCommand())
	return cmd
}

func newTaskStartCommand() *cobra.Command {
	var agents string
	var workdir string

	cmd := &cobra.Command{
		Use:   "start [title]",
		Short: "Start a new task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			agentReq, err := task.ParseAgentRequest(agents)
			if err != nil {
				return fmt.Errorf("invalid agent request: %w", err)
			}

			sessionMgr := session.NewManager(nil)
			taskMgr := task.NewManager(&sessionMgr)

			t, err := taskMgr.CreateTask(title, task.TaskModeRace, agentReq, workdir)
			if err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			ctx := context.Background()
			if err := taskMgr.StartTask(ctx, t.ID, 80, 24); err != nil {
				return fmt.Errorf("failed to start task: %w", err)
			}

			fmt.Printf("Started task %s: %s\n", t.ID, title)
			fmt.Printf("Sessions: %v\n", t.SessionIDs)
			return nil
		},
	}

	cmd.Flags().StringVar(&agents, "agents", "omp:1", "Agent specification (e.g., omp:3)")
	cmd.Flags().StringVar(&workdir, "workdir", "", "Working directory for agents")

	return cmd
}

func newTaskListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionMgr := session.NewManager(nil)
			taskMgr := task.NewManager(&sessionMgr)

			tasks := taskMgr.ListTasks()
			if len(tasks) == 0 {
				fmt.Println("No tasks found")
				return nil
			}

			for _, t := range tasks {
				fmt.Printf("%s: %s [%s] - %d sessions\n", t.ID, t.Title, t.Status, len(t.SessionIDs))
			}
			return nil
		},
	}
}
