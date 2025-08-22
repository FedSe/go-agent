package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type PythonInstall struct {
	Name            string
	Version         string
	InstallPath     string
	GUID            string
	SystemComponent bool
}

func getPythonInstalls() ([]PythonInstall, error) {
	keyPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var pythonInstalls []PythonInstall

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
			installPath, _, _ := appKey.GetStringValue("InstallLocation")
			systemComponentRaw, _, _ := appKey.GetIntegerValue("SystemComponent")
			appKey.Close()

			if !strings.Contains(displayName, "Python") || displayName == "" {
				continue
			}

			isSystemComponent := systemComponentRaw == 1

			pythonInstalls = append(pythonInstalls, PythonInstall{
				Name:            displayName,
				Version:         version,
				InstallPath:     installPath,
				GUID:            name,
				SystemComponent: isSystemComponent,
			})
		}
	}

	return pythonInstalls, nil
}

func printPythonInstalls() {
	installs, err := getPythonInstalls()
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	fmt.Printf("Найдено %d установок Python:\n", len(installs))

	for i, p := range installs {
		status := ""
		if p.SystemComponent {
			status = " (SystemComponent)"
		}

		fmt.Printf("\n%d. %s - Версия: %s%s\n", i+1, p.Name, p.Version, status)
		if p.InstallPath != "" {
			fmt.Printf("   Путь: %s\n", p.InstallPath)
		}
		fmt.Printf("   GUID: %s\n", p.GUID)
	}
}
