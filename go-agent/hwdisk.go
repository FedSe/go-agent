package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Логический диск
type Partition struct {
	Letter      string  `json:"Letter"`
	FileSystem  string  `json:"FileSystem"`
	Total       uint64  `json:"Total"`
	Used        uint64  `json:"Used"`
	UsedPercent float64 `json:"UsedPercent"`
}

// Физический диск
type DiskModule struct {
	Index      int         `json:"Index"`
	Model      string      `json:"Model"`
	Serial     string      `json:"Serial"`
	Size       uint64      `json:"Size"`
	Partitions []Partition `json:"Partitions"`
}

// Информация о Локальных дисках windows
func getDiskModules() ([]DiskModule, error) {
	psScript := `
        $disks = Get-WmiObject -Class Win32_DiskDrive
        foreach ($disk in $disks) {
            $diskObj = @{
				Index=$disk.Index
				Model=$disk.Model.Trim()
				Serial=$disk.SerialNumber.Trim()
				Size=$disk.Size
				Partitions=@()
			}
            $partitions = $disk.GetRelated("Win32_DiskPartition")
            if ($partitions -eq $null) { $partitions = @() }
            elseif ($partitions.GetType().Name -ne "Object[]") { $partitions = @($partitions) }
            foreach ($partition in $partitions) {
                $logicalDisks = $partition.GetRelated("Win32_LogicalDisk")
                if ($logicalDisks -eq $null) { $logicalDisks = @() }
                elseif ($logicalDisks.GetType().Name -ne "Object[]") { $logicalDisks = @($logicalDisks) }
                foreach ($ld in $logicalDisks) {
                    if ($ld.DriveType -eq 3) {
                        $total = $ld.Size
                        $free = $ld.FreeSpace
                        $used = $total - $free
                        $pct = if ($total -gt 0) { [math]::Round($used / $total * 100, 2) } else { 0 }
                        $partObj = @{ Letter=$ld.DeviceID; FileSystem=$ld.FileSystem; Total=[UInt64]$total; Used=[UInt64]$used; UsedPercent=[double]$pct }
                        $diskObj.Partitions += $partObj
                    }
                }
            }
            $diskObj | ConvertTo-Json -Compress
        }
    `

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения powershell: %v | output: %s", err, string(output))
	}

	return parseDiskJSON(string(output))
}

// Парсит данные
func parseDiskJSON(output string) ([]DiskModule, error) {
	var disks []DiskModule
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var disk DiskModule
		if err := json.Unmarshal([]byte(line), &disk); err != nil {
			fmt.Printf("ошибка парсинга строки: %s\n", line)
			continue
		}
		disks = append(disks, disk)
	}

	return disks, nil
}

// Форматирует данные для ответа
func formatDiskData(disks []DiskModule) map[string]any {
	response := make(map[string]any)

	for i, disk := range disks {
		prefix := fmt.Sprintf("d%d.", i)
		response[prefix+"Index"] = disk.Index
		response[prefix+"Model"] = disk.Model
		response[prefix+"Serial"] = disk.Serial
		response[prefix+"Size"] = formatBytes(disk.Size)

		for j, part := range disk.Partitions {
			pprefix := fmt.Sprintf("%sp%d.", prefix, j)
			response[pprefix+"Letter"] = part.Letter
			response[pprefix+"FileSystem"] = part.FileSystem
			response[pprefix+"Total"] = formatBytes(part.Total)
			response[pprefix+"Used"] = formatBytes(part.Used)
			response[pprefix+"UsedPercent"] = part.UsedPercent
		}
	}

	return response
}
