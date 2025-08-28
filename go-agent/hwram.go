package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// wmic memorychip get Manufacturer,PartNumber,SerialNumber,Capacity,Speed
type RAMModule struct {
	Manufacturer  string
	PartNumber    string
	SerialNumber  string
	CapacityBytes uint64
	Speed         int
}

// Информация о RAM windows
func getRAMModules() ([]RAMModule, error) {
	cmd := exec.Command("wmic", "memorychip", "get", "Manufacturer,PartNumber,SerialNumber,Capacity,Speed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения wmic: %w", err)
	}

	lines := strings.Split(string(output), "\r\n")
	var modules []RAMModule
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
		module, err := parseRAMLine(line, headers)
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
func formatRAMData(modules []RAMModule) map[string]any {
	response := make(map[string]any)

	var total uint64
	for _, m := range modules {
		total += m.CapacityBytes
	}

	for i, m := range modules {
		prefix := fmt.Sprintf("r%d.", i)
		response[prefix+"Manufacturer"] = m.Manufacturer
		response[prefix+"PartNumber"] = m.PartNumber
		response[prefix+"SerialNumber"] = m.SerialNumber
		response[prefix+"Capacity"] = formatBytes(m.CapacityBytes)
		response[prefix+"Speed"] = m.Speed
	}

	response["total"] = formatBytes(total)

	return response
}

// Парсит данные
func parseRAMLine(line string, headers []string) (*RAMModule, error) {
	fields := regexp.MustCompile(`\s{2,}`).
		Split(strings.TrimSpace(line), -1)

	if len(fields) < len(headers) {
		return nil, nil
	}

	var module RAMModule

	for i, header := range headers {
		value := fields[i]

		switch header {
		case "Manufacturer":
			module.Manufacturer = value
		case "PartNumber":
			module.PartNumber = value
		case "SerialNumber":
			module.SerialNumber = value
		case "Capacity":
			if value != "" {
				if cap, err := strconv.ParseUint(value, 10, 64); err == nil {
					module.CapacityBytes = cap
				}
			}
		case "Speed":
			if value != "" {
				if speed, err := strconv.Atoi(value); err == nil {
					module.Speed = speed
				}
			}
		}
	}

	return &module, nil
}
