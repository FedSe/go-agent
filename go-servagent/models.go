package main

import (
	"net"
	"sync"
	"time"
)

type CommandMessage struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Target  string `json:"target"`
}

type ClientLogEntry struct {
	Time    time.Time      `json:"time"`
	Command string         `json:"command"`
	Output  map[string]any `json:"output"`
}

type ResponseMessage struct {
	ClientID string         `json:"client_id"`
	Command  string         `json:"command"` // например: "get HN", "list VC"
	Data     map[string]any `json:"data,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type ClientInfo struct {
	ID             string    `json:"id"`
	IP             string    `json:"ip"`
	Hostname       string    `json:"hostname"`
	LastActiveTime time.Time `json:"last_active_time"`
	Online         bool      `json:"-"`
}
type ClientGroup struct {
	Name      string    `json:"name"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
}
type ClientGroupForShow struct {
	Name      string
	Members   []ClientInfo
	CreatedAt time.Time
}

var (
	clientRegistry = make(map[string]ClientInfo)
	activeConns    = make(map[string]net.Conn)

	registryMu sync.RWMutex
	connsMu    sync.RWMutex
	logsMu     sync.Mutex
	logs       = make([]string, 0)
)
