package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type InstalledApp struct {
	Name        string
	DisplayName string
	Version     string
	Publisher   string
	InstallPath string
}

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

func getInstalledAppsFiltered() ([]InstalledApp, error) {
	keyPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var apps []InstalledApp

	for _, keyPath := range keyPaths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.READ|registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		defer k.Close()

		names, err := k.ReadSubKeyNames(-1)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			subKeyPath := keyPath + `\` + name
			appKey, err := registry.OpenKey(registry.LOCAL_MACHINE, subKeyPath, registry.READ)
			if err != nil {
				continue
			}

			displayName, _, _ := appKey.GetStringValue("DisplayName")
			version, _, _ := appKey.GetStringValue("DisplayVersion")
			publisher, _, _ := appKey.GetStringValue("Publisher")
			//uninstallString, _, _ := appKey.GetStringValue("UninstallString")
			systemComponent, _, _ := appKey.GetIntegerValue("SystemComponent")
			parentKeyName, _, _ := appKey.GetStringValue("ParentKeyName")

			appKey.Close()

			// Пропуск системных компонентов
			if systemComponent == 1 {
				continue
			}

			// Пропуск элементов без DisplayName
			if displayName == "" { // || uninstallString == ""
				continue
			}

			// Пропуск обновлений и системных пакетов
			if strings.Contains(displayName, "Hotfix") ||
				strings.Contains(displayName, "Update") ||
				(strings.Contains(displayName, "Microsoft Visual C++") && strings.Contains(displayName, "Redistributable")) {
				continue
			}

			// Пропуск "Компоненты", а не программы
			if parentKeyName != "" {
				continue
			}

			apps = append(apps, InstalledApp{
				Name:        name,
				DisplayName: displayName,
				Version:     version,
				Publisher:   publisher,
			})
		}
	}

	return apps, nil
}
