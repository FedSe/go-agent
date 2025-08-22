package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// "C:\Program Files (x86)\Kaspersky Lab\Kaspersky Endpoint Security for Windows\avp.exe" UPDATE "D:\UPDTS11006499-56" /RA:"D:\soft\rpt.txt"

// Получение установки
func getAVPExe() (string, error) {
	script := `
$service = Get-WmiObject -Class Win32_Service -Filter "Name='AVP'"
$service.PathName
`
	cmd := exec.Command("powershell", "/C", script)
	output, _ := cmd.CombinedOutput()

	if len(output) < 5 {
		fmt.Println("KES не найден")
		return "", nil
	}
	kespath := strings.TrimSpace(string(extractQuotedString(output)))
	fmt.Printf("Установка KES: %s\n", kespath)

	script = fmt.Sprintf(`(Get-Item "%s").VersionInfo.ProductVersion;`, kespath)

	cmd = exec.Command("powershell", "/C", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения: %v", err)
	}
	ver := strings.TrimSpace(string(output))
	fmt.Printf("Версия KES: %s\n", ver)
	return ver, nil
}

func extractQuotedString(data []byte) []byte {
	start := bytes.IndexByte(data, '"')
	if start == -1 {
		return nil
	}

	end := bytes.IndexByte(data[start+1:], '"')
	if end == -1 {
		return nil
	}

	return data[start+1 : start+1+end]
}
