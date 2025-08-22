package main

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// MsiExec.exe /X{aac9fcc4-dd9e-4add-901c-b5496a07ab2e} /quiet /norestart /l C:\Users\Operator\FullLog.txt
// MsiExec.exe /X{a0fe116e-9a8a-466f-aee0-625cb7c207e3} /quiet /norestart /l C:\Users\Operator\JustLog.txt

type VCRedist struct {
	Name       string
	Version    string
	Arch       string
	Year       int
	IsDisabled bool
	GUID       string // {1F1C2DFC-28FF-3835-B233-9D0B47CE80AA}
}

func getVCRedists() ([]VCRedist, error) {
	keyPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	var redists []VCRedist

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
			//uninstallString, _, _ := appKey.GetStringValue("UninstallString")
			appKey.Close()

			if displayName == "" || !strings.Contains(displayName, "Microsoft Visual C++") || !strings.Contains(displayName, "Redistributable") {
				continue
			}

			arch := "x86"
			if strings.Contains(displayName, "x64") {
				arch = "x64"
			}

			year := extractVCYear(displayName)

			isDisabled := strings.Contains(displayName, "False")
			displayName = strings.Replace(displayName, " False", "", 1)

			guid := ""
			if strings.HasPrefix(name, "{") && strings.Contains(name, "-") {
				guid = name
			}

			redists = append(redists, VCRedist{
				Name:       displayName,
				Version:    version,
				Arch:       arch,
				Year:       year,
				IsDisabled: isDisabled,
				GUID:       guid,
			})
		}
	}

	sort.Slice(redists, func(i, j int) bool {
		return redists[i].Name > redists[j].Name
	})

	return redists, nil
}

func extractVCYear(name string) int {
	if strings.Contains(name, "2005") {
		return 2005
	} else if strings.Contains(name, "2008") {
		return 2008
	} else if strings.Contains(name, "2010") {
		return 2010
	} else if strings.Contains(name, "2012") {
		return 2012
	} else if strings.Contains(name, "2013") {
		return 2013
	} else if strings.Contains(name, "2015") || strings.Contains(name, "2017") || strings.Contains(name, "2019") || strings.Contains(name, "2022") {
		return 2015
	}

	return -1
}

func printGroupedVCRedists(redists []VCRedist) {
	grouped := make(map[int]map[string][]VCRedist)
	for _, r := range redists {
		if r.Year < 2005 {
			continue
		}

		if _, ok := grouped[r.Year]; !ok {
			grouped[r.Year] = map[string][]VCRedist{
				"x86": {},
				"x64": {},
			}
		}

		if r.Arch == "x64" {
			grouped[r.Year]["x64"] = append(grouped[r.Year]["x64"], r)
		} else {
			grouped[r.Year]["x86"] = append(grouped[r.Year]["x86"], r)
		}
	}

	years := []int{2005, 2008, 2010, 2012, 2013, 2015}
	for _, year := range years {
		x86List, x86Ok := grouped[year]["x86"]
		x64List, x64Ok := grouped[year]["x64"]

		if !x86Ok && !x64Ok {
			continue
		}

		fmt.Printf("\n%d:\n", year)

		if len(x64List) > 0 {
			fmt.Println("  x64:")
			for i, r := range x64List {
				fmt.Printf("    %d. %s\t ---\t%s,\t%t,\t%s\n", i+1, r.Name, r.Version, r.IsDisabled, r.GUID)
			}
		}

		if len(x86List) > 0 {
			fmt.Println("  x86:")
			for i, r := range x86List {
				fmt.Printf("    %d. %s\t ---\t%s,\t%t,\t%s\n", i+1, r.Name, r.Version, r.IsDisabled, r.GUID)
			}
		}
	}
}
