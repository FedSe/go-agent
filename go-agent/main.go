package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
)

const serverAddr = "10.0.0.3:8080"
const idFile = "agent_id.json"

var clientID string

func main() {
	fmt.Printf("Запущено на: %s\n", runtime.GOOS)

	switch runtime.GOOS {
	case "windows":
		//getHostname()
		//getCPUInfo()
		//getRAMInfo()
		//getDiskInfo()
		// getNetworkInfo()
		// getCDROMInfoWindows()
		//	getMonitorInfoWindows()
		//	getPrinterInfoWindows()
		//	getWiaDevicesWindows()
		//--getTwainDevices()
		//	getGPUInfoWindows()
		//	getWindowsServices()
		// getAVPExe()
		// getAppsWindows()
		//getRedistsWindows()
		// printPythonInstalls()
	case "linux":
		getHostname()
		getCPUInfo()
		getRAMInfo()
		getDiskInfo()
		getNetworkInfo()
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

// Имя хоста
func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("Ошибка при получении имени хоста: %s", err)
	}
	return hostname, nil
}

// Информация о CPU
func getCPUInfo() ([]map[string]string, error) {
	cpuInfos, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных о CPU: %w", err)
	}

	var result []map[string]string
	for _, info := range cpuInfos {
		result = append(result, map[string]string{
			"model":       info.ModelName,
			"mhz":         fmt.Sprintf("%.2f", info.Mhz),
			"cores":       strconv.Itoa(int(info.Cores)),
			"physical_id": info.PhysicalID,
		})
	}

	return result, nil
}

// formatCPUData преобразует []map[string]string от getCPUInfo в map[string]any для отправки
func formatCPUData(cpuData []map[string]string) map[string]any {
	response := make(map[string]any)
	for i, cpu := range cpuData {
		prefix := fmt.Sprintf("cpu%d.", i)
		for k, v := range cpu {
			response[prefix+k] = v
		}
	}
	return response
}

// Информация о RAM
func getRAMInfo() (string, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return "", fmt.Errorf("Ошибка при получении общих памяти: %s", err)
	}
	ram := fmt.Sprintf("%.2f", toGB(memInfo.Total))
	return ram, nil
}

// Информация о RAM windows
func getRAMModules() ([]map[string]string, error) {
	cmd := exec.Command("wmic", "memorychip", "get", "Manufacturer,PartNumber,SerialNumber,Capacity,Speed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения wmic: %w", err)
	}

	lines := strings.Split(string(output), "\r\n")

	var result []map[string]string
	var headers []string

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Первая непустая строка — заголовки
		if i == 0 || (headers == nil && i > 0) {
			headers = parseWMICHeaders(line)
			continue
		}

		// Парсим данные
		module := parseRAMLine(line, headers)
		if module == nil {
			continue
		}

		result = append(result, module)
	}

	return result, nil
}

func formatRAMData(totalGB string, modules []map[string]string) map[string]any {
	response := make(map[string]any)

	// Общий объём
	response["TotalGB"] = totalGB

	// Данные по каждому модулю
	for i, module := range modules {
		prefix := fmt.Sprintf("r%d.", i)
		for k, v := range module {
			response[prefix+k] = v
		}
	}

	return response
}

// Перевод байт в гигабайты
func toGB(b uint64) float64 {
	return float64(b) / 1024 / 1024 / 1024
}

// Windows: WMI через wmic
func parseRAMLine(line string, headers []string) map[string]string {
	fields := strings.Fields(line)
	if len(fields) != len(headers) {
		return nil
	}

	result := make(map[string]string)
	for i, header := range headers {
		result[header] = fields[i]
	}

	return result
}

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
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[%s] Ошибка выполнения команды.\n", title)
		return
	}
	fmt.Println(title + ":")
	fmt.Println(getRussianFromPSOut(output))
}

// Мониторы (Windows)
func getRussianFromPSOut(f []byte) string {
	//f = []byte{160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191}
	//f = []byte{128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159}
	//			 А	  Б	   В	Г	 Д	  Е	   Ж	З	 И	  Й	   К	Л	 М	  Н	   О	П	 Р	  С	   Т	У	 Ф	  Х	   Ц	Ч	 Ш	  Щ	   Ъ	Ы	 Ь	  Э	   Ю	Я
	var result []rune
	for _, b := range f {
		r := rune(b)
		if r > 191 {
			r -= 48
		}
		if r > 127 {
			r += 912
		}
		result = append(result, r)
	}
	return string(result)
}

func getWiaDevicesWindows() {
	script := `
[Console]::OutputEncoding = [Text.Encoding]::UTF8;
$deviceManager = New-Object -ComObject Wia.DeviceManager;
$deviceManager.DeviceInfos | ForEach-Object {
    $device = $_.Connect();
    "Имя: " + $device.Properties['Name'].Value;
    "ID:  " + $device.Properties['DeviceID'].Value;
    "---";
}`

	cmd := exec.Command("powershell", "/C", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Ошибка при получении WIA-устройств:", err)
		return
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		fmt.Println("WIA-устройства не найдены.")
		return
	}

	fmt.Println("\n=== WIA Устройства ===")
	fmt.Println(result)
}

// Дисководы / Картридеры (Windows)
func getCDROMInfoWindows() {
	runWMICCommand("wmic path Win32_CDROMDrive get Name,Drive", "Дисководы / Картридеры")
}

// Мониторы (Windows)
func getMonitorInfoWindows() {
	runWMICCommand("wmic path Win32_DesktopMonitor get Name,MonitorManufacturer,ScreenWidth,ScreenHeight", "Мониторы")
}

// Принтеры (Windows)
func getPrinterInfoWindows() {
	runWMICCommand("wmic printer get Name,DriverName,PortName", "Принтеры")
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

func getGPUInfoWindows() {
	fmt.Println("\n=== Информация о видеокарте (Windows) ===")

	script := `
Get-WmiObject -Class Win32_VideoController | Select Name, Description, VideoProcessor, AdapterRAM, DriverVersion`

	cmd := exec.Command("powershell", "/C", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Ошибка при получении данных:", err)
		return
	}

	result := string(output)
	if strings.TrimSpace(result) == "" {
		fmt.Println("Видеокарта не найдена.")
		return
	}

	fmt.Println(result)
}

func getWindowsServices() {
	fmt.Println("\n=== Службы Windows ===")

	script := `
[Console]::OutputEncoding = [Text.Encoding]::UTF8;
Get-WmiObject -Class Win32_Service |
Select Name, DisplayName, State, StartMode |
Format-List`

	cmd := exec.Command("powershell", "/C", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Ошибка выполнения команды:", err)
		return
	}

	result := string(output)
	if strings.TrimSpace(result) == "" {
		fmt.Println("  Не найдено служб.")
		return
	}

	fmt.Println(result)
}

// "C:\Program Files (x86)\Kaspersky Lab\Kaspersky Endpoint Security for Windows\avp.exe" UPDATE "D:\UPDTS11006499-56" /RA:"D:\soft\rpt.txt"

func getAppsWindows() {
	apps, err := getInstalledAppsFiltered()
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	fmt.Printf("Найдено %d установленных программ\n", len(apps))

	for i, app := range apps {
		fmt.Printf("\n%d. %s\n", i+1, app.DisplayName)
		if app.Version != "" {
			fmt.Printf("   Версия: %s\n", app.Version)
		}
		if app.Publisher != "" {
			fmt.Printf("   Производитель: %s\n", app.Publisher)
		}
		if app.InstallPath != "" {
			fmt.Printf("   Путь: %s\n", app.InstallPath)
		}
	}
}

func getRedistsWindows() {
	redists, err := getVCRedists()
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	printGroupedVCRedists(redists)
}
