package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thilob97/ottrta/internal/agent"
	"github.com/thilob97/ottrta/internal/session"
)

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
	}
	cmd.AddCommand(newAgentStartCommand())
	return cmd
}

func newAgentStartCommand() *cobra.Command {
	var count int
	var workdir string
	var nameFlag string

	cmd := &cobra.Command{
		Use:   "start [kind]",
		Short: "Start agent sessions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := strings.ToLower(args[0])
			if kind != "omp" {
				return fmt.Errorf("unsupported agent kind: %s", kind)
			}

			if !agent.OmpExists() {
				return fmt.Errorf("omp executable not found in PATH")
			}

			if count < 1 {
				count = 1
			}

			var sessions []session.Session
			for i := 1; i <= count; i++ {
				name := nameFlag
				if name == "" {
					name = fmt.Sprintf("omp-%d", i)
				} else if count > 1 {
					name = fmt.Sprintf("%s-%d", nameFlag, i)
				}
				sessions = append(sessions, session.Session{
					ID:        name,
					Name:      session.RandomImpName(),
					Kind:      session.SessionKindAgent,
					AgentKind: session.AgentKindOmp,
					Status:    session.StatusStopped,
					Command:   "omp",
					WorkDir:   workdir,
					Logs:      []string{fmt.Sprintf("%s ready", name)},
				})
			}

			manager := session.NewManager(sessions)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start sessions
			var allEvents []<-chan session.PTYEvent
			for _, s := range sessions {
				events, err := manager.StartPTYSession(ctx, s.ID, 80, 24)
				if err != nil {
					return fmt.Errorf("failed to start %s: %w", s.Name, err)
				}
				allEvents = append(allEvents, events)
			}

			// Multiplex events
			eventChan := make(chan session.PTYEvent)
			for _, evCh := range allEvents {
				go func(ch <-chan session.PTYEvent) {
					for e := range ch {
						eventChan <- e
					}
				}(evCh)
			}

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)

			fmt.Printf("Started %d omp agent(s). Press Ctrl+C to stop.\n", count)

			for {
				select {
				case <-sigChan:
					for _, s := range sessions {
						manager.StopPTYSession(s.ID)
					}
					return nil
				case ev := <-eventChan:
					if ev.Data != nil {
						fmt.Print(string(ev.Data))
					} else if ev.Err != nil {
						fmt.Printf("\n[%s exited with error: %v]\n", ev.SessionID, ev.Err)
					} else {
						fmt.Printf("\n[%s exited]\n", ev.SessionID)
					}
				}
			}
		},
	}

	cmd.Flags().IntVar(&count, "count", 1, "number of agents to start")
	cmd.Flags().StringVar(&workdir, "workdir", "", "working directory for agents")
	cmd.Flags().StringVar(&nameFlag, "name", "", "base name for the agents")

	return cmd
}
