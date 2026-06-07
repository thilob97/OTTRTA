package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"

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

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			defer signal.Stop(sigChan)

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Started %d omp agent(s). Press Ctrl+C to stop.\n", count)
			return runAgentEventLoop(ctx, cancel, out, sessions, allEvents, sigChan, manager.StopPTYSession)
		},
	}

	cmd.Flags().IntVar(&count, "count", 1, "number of agents to start")
	cmd.Flags().StringVar(&workdir, "workdir", "", "working directory for agents")
	cmd.Flags().StringVar(&nameFlag, "name", "", "base name for the agents")

	return cmd
}

func runAgentEventLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	out io.Writer,
	sessions []session.Session,
	allEvents []<-chan session.PTYEvent,
	sigChan <-chan os.Signal,
	stopSession func(string) error,
) error {
	eventChan := make(chan session.PTYEvent)
	var eventsDone sync.WaitGroup
	eventsDone.Add(len(allEvents))
	for _, evCh := range allEvents {
		go func(ch <-chan session.PTYEvent) {
			defer eventsDone.Done()
			for e := range ch {
				select {
				case eventChan <- e:
				case <-ctx.Done():
					return
				}
			}
		}(evCh)
	}
	go func() {
		eventsDone.Wait()
		close(eventChan)
	}()

	completed := make(map[string]struct{}, len(sessions))
	for {
		select {
		case <-sigChan:
			cancel()
			for _, s := range sessions {
				_ = stopSession(s.ID)
			}
			return nil
		case ev, ok := <-eventChan:
			if !ok {
				return nil
			}
			if ev.Data != nil {
				fmt.Fprint(out, string(ev.Data))
				continue
			}
			if _, seen := completed[ev.SessionID]; seen {
				continue
			}
			completed[ev.SessionID] = struct{}{}
			if ev.Err != nil {
				fmt.Fprintf(out, "\n[%s exited with error: %v]\n", ev.SessionID, ev.Err)
			} else {
				fmt.Fprintf(out, "\n[%s exited]\n", ev.SessionID)
			}
			if len(completed) == len(sessions) {
				return nil
			}
		}
	}
}
