package hidNfc

import "fmt"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

func printStructuredOutput(m24lr *M24LRxx) {
	fmt.Println(colorGreen + "=== RFID Tag Reader Output ===" + colorReset)

	printSection("Basic Information", func() {
		printField("UID", m24lr.UID())
		printField("Block 10 Data", getBlockData(m24lr))
		printField("Memory Size", getMemorySize(m24lr))
	})

	printSection("Beacon Information", func() {
		beaconInfo, _ := m24lr.ReadSKU()
		printField("Type", beaconInfo.BeaconType)
		printField("Name", beaconInfo.Name)
		printField("Image", beaconInfo.Image)
	})

	printSection("Ditto MACs", func() {
		macs, _ := m24lr.ReadDittoMacs()
		printField("LoRa MAC", macs.LoRaMAC)
		printField("BLE MAC", macs.BleMac)
		if macs.BleMac != "00:00:00:00:00:00" {
			fmt.Println(colorGreen + "Mac Written" + colorReset)
		} else {
			fmt.Println(colorRed + "Warning: Mac Not written, Erase tag and repeat" + colorReset)
		}
	})

	printSection("Local BLE Name", func() {
		localName, _ := m24lr.ReadLocalName()
		printField("Name", localName)
	})

	printSection("LoRa Settings", func() {
		settings, _ := m24lr.ReadLoRaSettings(3) // Assuming tag type 3 (Sense Range)
		printField("Beacon Type", fmt.Sprintf("%d", settings.BeaconType))
		printField("Hardware Version", settings.HardwareVersion)
		printField("Firmware Version", settings.FirmwareVersion)
		printField("Sleep State", settings.SleepState)
		printField("Min/Max Threshold", settings.MinMaxThreshold)
		printField("Range Type", settings.RangeType)
		printField("Spreading Factor", settings.SpreadingFactor)
		printField("Downlink Bit Rate", fmt.Sprintf("%d", settings.DownlinkBitRate))
		printField("Uplink Bit Rate", fmt.Sprintf("%d", settings.UplinkBitRate))
		printField("High Temperature", fmt.Sprintf("%d°C", settings.HighTemperature))
		printField("Low Temperature", fmt.Sprintf("%d°C", settings.LowTemperature))
		printField("Accelerometer", fmt.Sprintf("%d", settings.Accelerometer))
		printField("GNSS Min", fmt.Sprintf("%d", settings.GNSSMin))
		printField("GNSS Max", fmt.Sprintf("%d", settings.GNSSMax))
		printField("DOP", fmt.Sprintf("%.1f", settings.DOP))
		printField("Range Threshold", fmt.Sprintf("%d", settings.RangeThreshold))
		printField("Sensor Period", fmt.Sprintf("%d", settings.SensorPeriod))
		printField("Range Offset", fmt.Sprintf("%d", settings.RangeOffset))
		printField("Maximum Range", fmt.Sprintf("%d m", settings.MaximumRange))
	})
}

func printSection(title string, content func()) {
	fmt.Printf("\n%s=== %s ===%s\n", colorCyan, title, colorReset)
	content()
}

func printField(label, value string) {
	fmt.Printf("%s%s:%s %s\n", colorYellow, label, colorReset, value)
}
