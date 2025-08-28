package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
)

const serverAddr = "10.0.10.10:8081"
const idFile = "agent_id.json"

var clientID string

func main() {
	fmt.Printf("Запущено на: %s\n", runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		//getHostname()
		//getCPUInfo()
		//getDiskInfo()
		//getNetworkInfo()
		//getCDROMInfoWindows()   //++
		//getMonitorInfoWindows() //++
		//getPrinterInfoWindows() //++
		//getWiaDevicesWindows()  //++
		//getGPUInfoWindows()     //++
		//getWindowsServices()    //++
		//getAVPExe()
		//getAppsWindows()
		//getRedistsWindows()
		//printPythonInstalls()
	case "linux":
		getHostname()
		getCDROMInfoLinux()
		getMonitorInfoLinux()
		getPrinterInfoLinux()
	default:
		fmt.Printf("Операционная система %s не поддерживается.\n", runtime.GOOS)
	}

	id, err := getClientID()
	if err != nil {
		log.Fatal("Не удалось получить clientID:", err)
	}
	clientID = id

	fmt.Printf("Клиент запущен с ID: %s\n", clientID)

	for {
		startClient(clientID)
		time.Sleep(5 * time.Second)
	}
}

// Имя хоста
func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("ошибка при получении имени хоста: %s", err)
	}
	return hostname, nil
}

// Перевод байт в гигабайты
func toGB(b uint64) float64 {
	return float64(b) / 1024 / 1024 / 1024
}

// Парс заголовков
func parseWMICHeaders(line string) []string {
	fields := strings.Fields(line)
	return fields // список заголовков: "Capacity", "Manufacturer", ...
}

// Linux: dmidecode
func getRAMInfoLinux() {
	cmd := exec.Command("sudo", "dmidecode", "--type", "memory")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("[RAM] Не удалось выполнить dmidecode. Возможно, нужны права root.")
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Manufacturer") ||
			strings.Contains(line, "Part Number") ||
			strings.Contains(line, "Serial Number") ||
			strings.Contains(line, "Size") {
			fmt.Println(line)
		}
	}
}

// Информация о дисках
func getDiskInfo() {
	diskParts, _ := disk.Partitions(false)
	fmt.Println("\n--- Диски ---")
	for _, part := range diskParts {
		diskUsage, _ := disk.Usage(part.Mountpoint)
		fmt.Printf("  Раздел: %s (%s)\n", part.Device, part.Fstype)
		fmt.Printf("    Точка монтирования: %s\n", part.Mountpoint)
		fmt.Printf("    Всего: %.2f GB\n", toGB(diskUsage.Total))
		fmt.Printf("    Использовано: %.2f GB (%.2f%%)\n", toGB(diskUsage.Used), diskUsage.UsedPercent)
	}
}

// Сетевые интерфейсы
func getNetworkInfo() {
	netInfo, _ := net.Interfaces()
	fmt.Println("\n--- Сетевые интерфейсы ---")
	for i, intf := range netInfo {
		fmt.Printf("  %d: %s, MAC: %s, IP-адреса: %v\n", i, intf.Name, intf.HardwareAddr, intf.Addrs)
	}
}

// Выполнить команду и вывести результат
func runCommand(command string, title string) {
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[%s] Ошибка выполнения команды.\n", title)
		return
	}
	fmt.Println(title + ":")
	fmt.Println(string(output))
	fmt.Println()
}

// Выполнить WMI команду
func runWMICCommand(command string, title string) {
	script := `[Console]::OutputEncoding = [Text.Encoding]::UTF8;
` + command

	cmd := exec.Command("powershell", "/C", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("[%s] Ошибка выполнения команды.\n", title)
		return
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		fmt.Printf("Не найдено: %s\n", title)
		return
	}

	fmt.Printf("\n=== %s ===\n", title)
	fmt.Println(result)
}

// Запрос WIA устройств (Сканер, вебкамера, МФУ и т.д.)
func getWiaDevicesWindows() {
	runWMICCommand(`Get-WmiObject -Class Win32_PnPEntity | Where-Object { $_.PNPClass -eq "Image" } | ForEach-Object {
    "Имя: " + $_.Name
    "ID:  " + $_.DeviceID
    "---"
}`, "WIA устройства")
}

// Дисководы / Картридеры (Windows)
func getCDROMInfoWindows() {
	runWMICCommand(`Get-WmiObject -Class Win32_CDROMDrive | ForEach-Object {
    "Имя: " + $_.Name;
    "Диск: " + $_.Drive;
	"---";
}`, "Дисководы / Картридеры")
}

// Мониторы (Windows)
func getMonitorInfoWindows() {
	runWMICCommand(`Get-WmiObject -Class Win32_DesktopMonitor | ForEach-Object {
    "Имя: " + $_.Name;
    "Производитель: " + $_.MonitorManufacturer;
	"Разрешение: $($_.ScreenWidth)x$($_.ScreenHeight)";
	"---";
}`, "Мониторы")
}

// Принтеры (Windows)
func getPrinterInfoWindows() {
	runWMICCommand(`Get-WmiObject -Class Win32_Printer | ForEach-Object {
    "Имя:     " + $_.Name;
    "Драйвер: " + $_.DriverName;
    "Порт:    " + $_.PortName;
    "---";
}`, "Принтеры")
}

// Дисководы / Картридеры (Linux)
func getCDROMInfoLinux() {
	runCommand("lsblk -d -o NAME,TYPE,VENDOR,MODEL", "Дисководы / Картридеры")
}

// Мониторы (Linux)
func getMonitorInfoLinux() {
	runCommand("xrandr --verbose | grep -i 'connected'", "Мониторы")
}

// Принтеры (Linux)
func getPrinterInfoLinux() {
	runCommand("lpstat -v", "Принтеры")
}

// Видеокарты
func getGPUInfoWindows() {
	runWMICCommand(`Get-WmiObject -Class Win32_VideoController | ForEach-Object {
    "Имя:             " + $_.Name;
    "Описание:        " + $_.Description;
    "Видеопроцессор:  " + $_.VideoProcessor;
	"Адаптер:         " + $_.AdapterRAM;
    "Версия драйвера: " + $_.DriverVersion;
    "---";
}`, "Видеокарты")
}

func getWindowsServices() {
	runWMICCommand(`Get-WmiObject -Class Win32_Service | ForEach-Object {
    "Имя службы       : " + $_.Name
    "Отображаемое имя : " + $_.DisplayName
    "Состояние        : " + $_.State
    "Тип запуска      : " + $_.StartMode
    "---"
}`, "Службы Windows")
}

func getRedistsWindows() {
	redists, err := getVCRedists()
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	printGroupedVCRedists(redists)
}
