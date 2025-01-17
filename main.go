package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// KMS server https://www.coolhub.top/tech-articles/kms_list.html
var scriptTpl = `
@echo off
setlocal enabledelayedexpansion


:: Set the KMS server
slmgr -skms skms.netnr.eu.org
if %%ERRORLEVEL%% neq 0 (
    echo Error: Failed to set KMS server. Exiting...
    exit /b %%ERRORLEVEL%%
)

:: Set the product key
slmgr -ipk %s
if %%ERRORLEVEL%% neq 0 (
    echo Error: Failed to set product key. Exiting...
    exit /b %%ERRORLEVEL%%
)

:: Activate the system
slmgr -ato
if %%ERRORLEVEL%% neq 0 (
    echo Error: Failed to activate Windows. Exiting...
    exit /b %%ERRORLEVEL%%
)

echo Your windows system has been successfully activated.
echo The activation program will automatically exit after five seconds...
exit /b 0
`

// GVLK KMS code copy from https://learn.microsoft.com/en-us/windows-server/get-started/kms-client-activation-keys?tabs=server2025%2Cwindows1110ltsc%2Cversion1803%2Cwindows81
var editionKeyMap = map[string]string{
	"Windows 11 Pro":                    "W269N-WFGWX-YVC9B-4J6C9-T83GX",
	"Windows 10 Pro":                    "W269N-WFGWX-YVC9B-4J6C9-T83GX",
	"Windows 11 Pro N":                  "MH37W-N47XK-V7XM9-C7227-GCQG9",
	"Windows 10 Pro N":                  "MH37W-N47XK-V7XM9-C7227-GCQG9",
	"Windows 11 Pro for Workstations":   "NRG8B-VKK3Q-CXVCJ-9G2XF-6Q84J",
	"Windows 10 Pro for Workstations":   "NRG8B-VKK3Q-CXVCJ-9G2XF-6Q84J",
	"Windows 11 Pro for Workstations N": "9FNHH-K3HBT-3W4TD-6383H-6XYWF",
	"Windows 10 Pro for Workstations N": "9FNHH-K3HBT-3W4TD-6383H-6XYWF",
	"Windows 11 Pro Education":          "6TP4R-GNPTD-KYYHQ-7B7DP-J447Y",
	"Windows 10 Pro Education":          "6TP4R-GNPTD-KYYHQ-7B7DP-J447Y",
	"Windows 11 Pro Education N":        "YVWGF-BXNMC-HTQYQ-CPQ99-66QFC",
	"Windows 10 Pro Education N":        "YVWGF-BXNMC-HTQYQ-CPQ99-66QFC",
	"Windows 11 Education":              "NW6C2-QMPVW-D7KKK-3GKT6-VCFB2",
	"Windows 10 Education":              "NW6C2-QMPVW-D7KKK-3GKT6-VCFB2",
	"Windows 11 Education N":            "2WH4N-8QGBV-H22JP-CT43Q-MDWWJ",
	"Windows 10 Education N":            "2WH4N-8QGBV-H22JP-CT43Q-MDWWJ",
	"Windows 11 Enterprise":             "NPPR9-FWDCX-D2C8J-H872K-2YT43",
	"Windows 10 Enterprise":             "NPPR9-FWDCX-D2C8J-H872K-2YT43",
	"Windows 11 Enterprise N":           "DPH2V-TTNVB-4X9Q3-TJR4H-KHJW4",
	"Windows 10 Enterprise N":           "DPH2V-TTNVB-4X9Q3-TJR4H-KHJW4",
	"Windows 11 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
	"Windows 10 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
	"Windows 11 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",
	"Windows 10 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",
}

// getWindowsEdition retrieves the edition of the current Windows system.
func getWindowsEdition() (string, error) {
	// Define the RtlGetVersion function from NTDLL
	var mod = syscall.NewLazyDLL("ntdll.dll")
	var proc = mod.NewProc("RtlGetVersion")

	type OSVERSIONINFOEX struct {
		DwOSVersionInfoSize uint32
		DwMajorVersion      uint32
		DwMinorVersion      uint32
		DwBuildNumber       uint32
		DwPlatformId        uint32
		SzCSDVersion        [128]uint16
		WServicePackMajor   uint16
		WServicePackMinor   uint16
		WReserved           [2]uint16
	}

	// Allocate memory for the OSVERSIONINFOEX structure
	osvi := OSVERSIONINFOEX{
		DwOSVersionInfoSize: uint32(unsafe.Sizeof(OSVERSIONINFOEX{})),
	}

	// Call the RtlGetVersion function
	ret, _, err := proc.Call(uintptr(unsafe.Pointer(&osvi)))
	if ret != 0 {
		return "", err
	}

	// Detect Windows edition (e.g., Home, Pro, Enterprise)
	edition := "Unknown Edition"

	// Query the Windows Product Name using Registry
	k, err := registry.OpenKey(windows.HKEY_LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, windows.KEY_READ)
	if err != nil {
		return "", fmt.Errorf("failed to open registry: %v", err)
	}
	defer k.Close()

	productName, _, err := k.GetStringValue("ProductName")
	if err != nil {
		return "", fmt.Errorf("failed to get ProductName from registry: %v", err)
	}

	edition = productName

	return edition, nil
}

func runScript(script string) {
	// Write the batch script to a temporary file
	tmpFile, err := os.CreateTemp("", "script-*.bat")
	if err != nil {
		fmt.Printf("Error creating temporary file: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(script)
	if err != nil {
		fmt.Printf("Error writing to temporary file: %v\n", err)
		return
	}

	// Close the file before executing it
	if err := tmpFile.Close(); err != nil {
		fmt.Printf("Error closing temporary file: %v\n", err)
		return
	}

	// Execute the batch file
	cmd := exec.Command("cmd.exe", "/C", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running batch file: %v\n", err)
		return
	}

	// Print the output of the batch script
	fmt.Printf("\n%s\n", string(output))

	time.Sleep(time.Second * 5)
}

func main() {
	edition, err := getWindowsEdition()
	if err != nil {
		fmt.Printf("Get Windows Edition err: %v\n", err)
		time.Sleep(time.Second * 3)
		return
	}

	fmt.Printf("Your Windows Edition is: %v\n", edition)
	fmt.Printf("Trying to activate your Windows if it is not activated...\n")

	key, ok := editionKeyMap[edition]
	if !ok {
		fmt.Printf("Not found the activation code for: %v\n", edition)
		time.Sleep(time.Second * 3)
		return
	}

	script := fmt.Sprintf(scriptTpl, key)

	runScript(script)
}
