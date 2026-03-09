//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const credentialName = "rickspanish/encryption-key"

// getOrCreateEncryptionKey retrieves the encryption key from Windows Credential
// Manager, creating and storing a new one if it doesn't exist yet.
func getOrCreateEncryptionKey() ([]byte, error) {
	hexKey, err := credManagerGet()
	if err != nil {
		hexKey, err = generateKey()
		if err != nil {
			return nil, err
		}
		if err := credManagerSet(hexKey); err != nil {
			// If PowerShell isn't available, fall back to env var
			envKey := os.Getenv("RICKSPANISH_ENCRYPTION_KEY")
			if envKey != "" {
				return hexToKey(envKey)
			}
			return nil, fmt.Errorf("storing key in Windows Credential Manager: %w\n"+
				"Alternatively, set RICKSPANISH_ENCRYPTION_KEY environment variable.", err)
		}
		fmt.Println("New encryption key generated and stored in Windows Credential Manager.")
	}
	return hexToKey(hexKey)
}

func credManagerGet() (string, error) {
	script := `
$cred = Get-StoredCredential -Target '` + credentialName + `' -ErrorAction SilentlyContinue
if ($cred -eq $null) { exit 1 }
$cred.GetNetworkCredential().Password
`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		// Try cmdkey approach as fallback
		return credManagerGetCmdkey()
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("no credential found")
	}
	return result, nil
}

func credManagerGetCmdkey() (string, error) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Security
$cred = [System.Net.CredentialCache]::DefaultNetworkCredentials
$wc = New-Object System.Net.WebClient
$wc.Credentials = [System.Net.CredentialCache]::DefaultCredentials
try {
    $cm = New-Object System.Management.Automation.PSCredential('%s',
        (Get-StoredCredential -Target '%s').Password)
    $cm.GetNetworkCredential().Password
} catch { exit 1 }
`, credentialName, credentialName)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return "", err
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("no credential found")
	}
	return result, nil
}

func credManagerSet(hexKey string) error {
	script := fmt.Sprintf(`
$securePass = ConvertTo-SecureString '%s' -AsPlainText -Force
$cred = New-Object System.Management.Automation.PSCredential('%s', $securePass)
if (Get-Command cmdkey -ErrorAction SilentlyContinue) {
    cmdkey /generic:'%s' /user:'rickspanish' /pass:'%s'
} else {
    $cred | Export-Clixml -Path "$env:APPDATA\rickspanish\cred.xml"
}
`, hexKey, credentialName, credentialName, hexKey)
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}
