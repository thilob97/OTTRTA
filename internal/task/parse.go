package task

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thilob97/ottrta/internal/session"
)

type AgentRequest struct {
	Kind  session.AgentKind
	Count int
}

func ParseAgentRequest(input string) (AgentRequest, error) {
	if input == "" {
		return AgentRequest{}, fmt.Errorf("empty agent request")
	}

	parts := strings.Split(input, ":")
	kindStr := parts[0]
	count := 1

	if len(parts) == 2 {
		c, err := strconv.Atoi(parts[1])
		if err != nil {
			return AgentRequest{}, fmt.Errorf("invalid count: %s", parts[1])
		}
		if c <= 0 {
			return AgentRequest{}, fmt.Errorf("count must be positive")
		}
		count = c
	} else if len(parts) > 2 {
		return AgentRequest{}, fmt.Errorf("invalid format: %s", input)
	}

	kind := session.AgentKind(kindStr)
	if kind != session.AgentKindOmp {
		return AgentRequest{}, fmt.Errorf("unknown agent kind: %s", kindStr)
	}

	return AgentRequest{Kind: kind, Count: count}, nil
}
