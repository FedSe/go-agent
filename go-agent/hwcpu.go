package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// wmic cpu get Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed
type CPUModule struct {
	Model   string
	Cores   int
	Threads int
	Speed   int
}

// Информация о CPU windows
func getCPUModules() ([]CPUModule, error) {
	cmd := exec.Command("wmic", "cpu", "get", "Name,NumberOfCores,NumberOfLogicalProcessors,MaxClockSpeed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения wmic: %w", err)
	}

	lines := strings.Split(string(output), "\r\n")
	var modules []CPUModule
	var headers []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Заголовки
		if headers == nil {
			headers = parseWMICHeaders(line)
			continue
		}

		// Данные
		module, err := parseCPULine(line, headers)
		if err != nil {
			continue
		}
		if module != nil {
			modules = append(modules, *module)
		}
	}

	return modules, nil
}

// Форматирует данные для ответа
func formatCPUData(modules []CPUModule) map[string]any {
	response := make(map[string]any)

	for i, m := range modules {
		prefix := fmt.Sprintf("c%d.", i)
		response[prefix+"Model"] = m.Model
		response[prefix+"Cores"] = m.Cores
		response[prefix+"Threads"] = m.Threads
		response[prefix+"Speed"] = m.Speed
	}

	return response
}

// Парсит данные
func parseCPULine(line string, headers []string) (*CPUModule, error) {
	fields := regexp.MustCompile(`\s{2,}`).
		Split(strings.TrimSpace(line), -1)

	if len(fields) < len(headers) {
		return nil, nil
	}

	var module CPUModule

	for i, header := range headers {
		value := fields[i]

		switch header {
		case "Name":
			module.Model = value
		case "NumberOfCores":
			if value != "" {
				if cores, err := strconv.Atoi(value); err == nil {
					module.Cores = cores
				}
			}
		case "NumberOfLogicalProcessors":
			if value != "" {
				if threads, err := strconv.Atoi(value); err == nil {
					module.Threads = threads
				}
			}
		case "MaxClockSpeed":
			if value != "" {
				if speed, err := strconv.Atoi(value); err == nil {
					module.Speed = speed
				}
			}
		}
	}

	return &module, nil
}
