package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/andlabs/ui"
)

var dnsServers = []string{
	"Shecan | 178.22.122.100, 185.51.200.2",
	"Electro Team | 78.157.42.100, 78.157.42.101",
	"Radar game | 10.202.10.10, 10.202.10.11",
	"403.online | 10.202.10.202, 10.202.10.102",
	"Asiatech | 194.36.174.161, 178.22.122.100",
	"Cloudflare | 1.1.1.1, 1.0.0.1",
	"Gaming 1 | 78.157.42.100, 78.157.42.101",
	"Gaming 2 | 88.135.36.247, 0.0.0.0",
	"Gaming 3 | 178.22.122.100, 185.51.200.2",
	"Gaming 4 | 37.152.182.112, 0.0.0.0",
	"Gaming 5 | 78.157.41.100, 88.135.36.247",
	"Gaming 6 | 109.96.8.51, 78.157.42.101",
	"Gaming 7 | 78.157.42.100, 77.157.42.110",
	"Gaming 8 | 45.90.30.205, 45.90.30.193",
	"Google DNS | 8.8.8.8, 8.8.4.4",
	"OpenDNS | 208.67.222.222, 208.67.220.220",
	"Quad9 | 9.9.9.9, 149.112.112.112",
	"Comodo Secure DNS | 8.26.56.26, 8.20.247.20",
	"Norton ConnectSafe | 199.85.126.10, 199.85.127.10",
	"Yandex.DNS | 77.88.8.8, 77.88.8.1",
}

var resolvConfPath = "/etc/resolv.conf"
var backupResolvConfPath = "/etc/resolv.conf.bak"
var systemdResolvConfPath = "/run/systemd/resolve/stub-resolv.conf"

func main() {
	err := ui.Main(func() {
		window := ui.NewWindow("DNS Changer", 400, 200, true)
		window.SetMargined(true)

		vbox := ui.NewVerticalBox()
		vbox.SetPadded(true)

		combobox := ui.NewCombobox()
		for _, dns := range dnsServers {
			combobox.Append(strings.TrimSpace(strings.Split(dns, "|")[0]))
		}
		vbox.Append(combobox, false)

		pingResult := ui.NewLabel("")
		vbox.Append(pingResult, false)

		buttonConnect := ui.NewButton("Connect")
		buttonConnect.OnClicked(func(*ui.Button) {
			selectedDNS := dnsServers[combobox.Selected()]
			selectedIPs := parseIPs(selectedDNS)
			err := changeDNS(selectedIPs)
			if err != nil {
				log.Println("Error changing DNS:", err)
				ui.MsgBoxError(window, "Error", "Failed to change DNS server.")
				return
			}
			ui.MsgBox(window, "Success", fmt.Sprintf("DNS server changed to: %s", selectedDNS))
		})
		vbox.Append(buttonConnect, false)

		buttonTurnOff := ui.NewButton("Turn Off")
		buttonTurnOff.OnClicked(func(*ui.Button) {
			err := turnOffDNS()
			if err != nil {
				log.Println("Error turning off DNS:", err)
				ui.MsgBoxError(window, "Error", "Failed to turn off DNS.")
				return
			}
			ui.MsgBox(window, "Success", "DNS server turned off.")
		})
		vbox.Append(buttonTurnOff, false)

		combobox.OnSelected(func(*ui.Combobox) {
			selectedDNS := dnsServers[combobox.Selected()]
			selectedIPs := parseIPs(selectedDNS)
			go performPing(selectedIPs, pingResult)
		})

		window.SetChild(vbox)
		window.OnClosing(func(*ui.Window) bool {
			ui.Quit()
			return true
		})

		window.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

func parseIPs(dns string) []string {
	parts := strings.Split(dns, "|")
	if len(parts) != 2 {
		return nil
	}

	ips := strings.Split(strings.TrimSpace(parts[1]), ",")
	for i := range ips {
		ips[i] = strings.TrimSpace(ips[i])
	}

	return ips
}

func changeDNS(ips []string) error {
	err := backupResolvConf()
	if err != nil {
		return fmt.Errorf("failed to backup resolv.conf: %w", err)
	}

	err = replaceResolvConf(ips)
	if err != nil {
		restoreResolvConf()
		return fmt.Errorf("failed to replace resolv.conf: %w", err)
	}

	return nil
}

func backupResolvConf() error {
	err := copyFile(resolvConfPath, backupResolvConfPath)
	if err != nil {
		return fmt.Errorf("failed to backup resolv.conf: %w", err)
	}

	return nil
}

func replaceResolvConf(ips []string) error {
	content := "nameserver " + strings.Join(ips, "\nnameserver ")
	err := writeFile(resolvConfPath, content+"\n")
	if err != nil {
		return fmt.Errorf("failed to replace resolv.conf: %w", err)
	}

	return nil
}

func restoreResolvConf() {
	copyFile(backupResolvConfPath, resolvConfPath)
}

func turnOffDNS() error {
	err := exec.Command("sudo", "rm", resolvConfPath).Run()
	if err != nil {
		return fmt.Errorf("failed to remove resolv.conf: %w", err)
	}

	err = exec.Command("sudo", "ln", "-rsf", systemdResolvConfPath, resolvConfPath).Run()
	if err != nil {
		return fmt.Errorf("failed to link resolv.conf to systemd: %w", err)
	}

	err = exec.Command("sudo", "service ", "systemd-resolved", "restart").Run()
	if err != nil {
		return fmt.Errorf("failed to link resolv.conf to systemd: %w", err)
	}

	return nil
}

func copyFile(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	err = writeFile(dest, string(input))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func writeFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func performPing(ips []string, resultLabel *ui.Label) {
	var wg sync.WaitGroup

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			cmd := exec.Command("ping", "-c", "4", ip)
			out, err := cmd.Output()

			if err != nil {
				result := fmt.Sprintf("[%s] DNS is unavailable", ip)
				ui.QueueMain(func() {
					resultLabel.SetText(result)
					// resultLabel.SetColor(ui.ColorRed)
				})
			} else {
				pingOutput := string(out)
				// Parse the ping statistics
				statsRegex := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received`)
				statsMatches := statsRegex.FindStringSubmatch(pingOutput)
				if len(statsMatches) >= 3 {
					packetsTransmitted, _ := strconv.Atoi(statsMatches[1])
					packetsReceived, _ := strconv.Atoi(statsMatches[2])
					packetLoss := float64(packetsTransmitted-packetsReceived) / float64(packetsTransmitted) * 100

					// Determine the ping result based on packet loss
					var result string
					// var color ui.Color
					if packetLoss == 0 {
						result = fmt.Sprintf("[%s] DNS is %.3f ms", ip, parsePingTime(pingOutput))
						// color = ui.ColorGreen
					} else {
						result = fmt.Sprintf("[%s] DNS is unavailable", ip)
						// color = ui.ColorRed
					}

					ui.QueueMain(func() {
						resultLabel.SetText(result)
						// resultLabel.SetColor(color)
					})
				}
			}
		}(ip)
	}

	wg.Wait()
}

func parsePingTime(pingOutput string) float64 {
	// Parse the round-trip time
	rttRegex := regexp.MustCompile(`rtt min/avg/max/mdev = (\d+\.\d+)/(\d+\.\d+)/(\d+\.\d+)/(\d+\.\d+)`)
	rttMatches := rttRegex.FindStringSubmatch(pingOutput)
	if len(rttMatches) >= 5 {
		avgTime, _ := strconv.ParseFloat(rttMatches[2], 64)
		return avgTime
	}
	return 0.0
}
