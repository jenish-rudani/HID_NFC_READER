package main

import (
	"encoding/hex"
	"fmt"
	"github.com/dumacp/smartcard/pcsc"
	"strconv"
	"strings"
	"unicode/utf16"
)

// M24LRxx represents a M24LR series RFID tag
type M24LRxx struct {
	uid    string
	reader pcsc.Card
}

// BeaconType represents the type of beacon
type BeaconType uint64

var currentBeaconType uint64

const (
	BeaconTypeNone BeaconType = iota
	BeaconTypeIBeacon
	BeaconTypeEddystone
)

// UUIDInfo contains the UUID and related information
type UUIDInfo struct {
	UUID     string
	Major    string
	Minor    string
	Instance string
}

// LoRaSettings holds all the settings read from the RFID tag
type LoRaSettings struct {
	BeaconType      int
	HardwareVersion string
	FirmwareVersion string
	SleepState      string
	MinMaxThreshold string
	RangeType       string
	SpreadingFactor string
	DownlinkBitRate int
	UplinkBitRate   int
	HighTemperature int
	LowTemperature  int
	Accelerometer   int
	GNSSMin         int
	GNSSMax         int
	DOP             float64
	RangeThreshold  int
	SensorPeriod    int
	RangeOffset     int
	MaximumRange    int
}

// ReadLoRaSettings reads all LoRa settings from the RFID tag
func (m *M24LRxx) ReadLoRaSettings(tagType int) (*LoRaSettings, error) {
	settings := &LoRaSettings{}

	blocks := make(map[int]string)
	for i := 8; i <= 15; i++ {
		block, err := m.ReadBlock(i)
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %v", i, err)
		}
		blocks[i] = block
	}

	// Parse Block 15
	settings.HardwareVersion = blocks[15][1:2]
	fwInt, _ := strconv.ParseInt(blocks[15][2:4], 16, 0)
	settings.FirmwareVersion = fmt.Sprintf("%.1f", float64(fwInt)/10)
	settings.BeaconType, _ = strconv.Atoi(blocks[15][4:6])

	// Parse Block 13
	switch blocks[13][2:3] {
	case "1":
		settings.SleepState = "Asleep"
	case "0":
		settings.SleepState = "Awake"
	}

	switch blocks[13][4:5] {
	case "0":
		settings.MinMaxThreshold = "Below"
	case "1":
		settings.MinMaxThreshold = "Above"
	}

	switch blocks[13][6:7] {
	case "0":
		settings.RangeType = "Short 1.3m"
	case "1":
		settings.RangeType = "Long 4m"
	}

	// Parse Block 8
	sfInt, _ := strconv.ParseInt(blocks[8][:2], 16, 0)
	if sfInt == 255 {
		settings.SpreadingFactor = "ADR"
	} else {
		settings.SpreadingFactor = fmt.Sprintf("%d", sfInt)
	}
	settings.DownlinkBitRate, _ = strconv.Atoi(blocks[8][2:4])
	settings.UplinkBitRate, _ = strconv.Atoi(blocks[8][4:6])
	tempHigh, _ := strconv.ParseInt(blocks[8][6:], 16, 0)
	settings.HighTemperature = int(tempHigh) - 127

	// Parse Block 9
	tempLow, _ := strconv.ParseInt(blocks[9][:2], 16, 0)
	settings.LowTemperature = int(tempLow) - 127
	settings.Accelerometer, _ = strconv.Atoi(blocks[9][2:4])

	if tagType == 3 { // Sense Range
		settings.RangeThreshold, _ = strconv.Atoi(blocks[9][4:])
		settings.SensorPeriod, _ = strconv.Atoi(blocks[10][:2])
		settings.RangeOffset, _ = strconv.Atoi(blocks[10][4:6])
		maxRange, _ := strconv.Atoi(blocks[10][6:])
		settings.MaximumRange = maxRange * 10
	}

	// Parse Block 14
	settings.GNSSMin, _ = strconv.Atoi(blocks[14][:2])
	settings.GNSSMax, _ = strconv.Atoi(blocks[14][2:4])
	dop, _ := strconv.ParseFloat(blocks[14][4:6], 64)
	settings.DOP = dop / 10

	return settings, nil
}

func printLoRaSettings(settings *LoRaSettings) {
	fmt.Println("LoRa Settings:")
	fmt.Printf("Beacon Type: %d\n", settings.BeaconType)
	fmt.Printf("Hardware Version: %s\n", settings.HardwareVersion)
	fmt.Printf("Firmware Version: %s\n", settings.FirmwareVersion)
	fmt.Printf("Sleep State: %s\n", settings.SleepState)
	fmt.Printf("Min/Max Threshold: %s\n", settings.MinMaxThreshold)
	fmt.Printf("Range Type: %s\n", settings.RangeType)
	fmt.Printf("Spreading Factor: %s\n", settings.SpreadingFactor)
	fmt.Printf("Downlink Bit Rate: %d\n", settings.DownlinkBitRate)
	fmt.Printf("Uplink Bit Rate: %d\n", settings.UplinkBitRate)
	fmt.Printf("High Temperature: %d\n", settings.HighTemperature)
	fmt.Printf("Low Temperature: %d\n", settings.LowTemperature)
	fmt.Printf("Accelerometer: %d\n", settings.Accelerometer)
	fmt.Printf("GNSS Min: %d\n", settings.GNSSMin)
	fmt.Printf("GNSS Max: %d\n", settings.GNSSMax)
	fmt.Printf("DOP: %.1f\n", settings.DOP)
	fmt.Printf("Range Threshold: %d\n", settings.RangeThreshold)
	fmt.Printf("Sensor Period: %d\n", settings.SensorPeriod)
	fmt.Printf("Range Offset: %d\n", settings.RangeOffset)
	fmt.Printf("Maximum Range: %d\n", settings.MaximumRange)
}

// BeaconInfo holds information about the beacon type
type BeaconInfo struct {
	BeaconType string
	Name       string
	Image      string // In Go, we'll just store the image name/path
}

// ReadSKU reads the SKU (beacon type) from the tag
func (m *M24LRxx) ReadSKU() (*BeaconInfo, error) {
	block, err := m.ReadBlock(15)
	if err != nil {
		return nil, fmt.Errorf("failed to read block 15: %v", err)
	}

	beaconType := block[4:6]
	currentBeaconType, err = strconv.ParseUint(beaconType, 16, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beacon type: %v", err)
	}
	return getBeaconInfo(beaconType)
}

func getBeaconInfo(beaconType string) (*BeaconInfo, error) {
	info := &BeaconInfo{BeaconType: beaconType}

	switch beaconType {
	case "00", "01", "FF":
		info.Name = "Please select the tag type"
		info.Image = ""
	case "0D":
		info.Name = "Sense Asset BLE"
		info.Image = "Sense_BLE_Small"
	case "12":
		info.Name = "Sense Asset XL"
		info.Image = "Asset_Small"
	case "09":
		info.Name = "Sense Condition Range Finder"
		info.Image = "Range_Small"
	case "08":
		info.Name = "Sense Condition Alert"
		info.Image = "Button_Small"
	case "14":
		info.Name = "Sense Shield/Badge/Lite"
		info.Image = "Social2"
	case "15":
		info.Name = "Sense Asset +"
		info.Image = "Ditto_correct_200_trans"
	case "13":
		info.Name = "Sense Asset Temp"
		info.Image = "Sense_BLE_Small"
	case "16":
		info.Name = "Sense Asset"
		info.Image = "Sense_BLE_Small"
	case "17":
		info.Name = "Sense Wirepass"
		info.Image = "Social2"
	default:
		return nil, fmt.Errorf("unknown beacon type: %s", beaconType)
	}

	return info, nil
}

func printBeaconInfo(info *BeaconInfo) {
	fmt.Println("Beacon Information:")
	fmt.Printf("Type: %s\n", info.BeaconType)
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Image: %s\n", info.Image)
}

// ReadUUID reads the UUID and related information based on the beacon type
func (m *M24LRxx) ReadUUID(beaconType uint64) (*UUIDInfo, error) {
	blocks := make([]string, 6)
	var err error

	for i := 10; i <= 15; i++ {
		blocks[i-10], err = m.ReadBlock(i)
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %v", i, err)
		}
	}

	info := &UUIDInfo{}

	switch beaconType {
	case 0:
		// Do nothing
	case 1:
		info.UUID = strings.Join(blocks[:4], "")
		info.Major = blocks[4][:4]
		info.Minor = blocks[4][4:]
	case 2:
		info.UUID = blocks[0] + blocks[1] + blocks[2][:4]
		info.Instance = blocks[2][4:] + blocks[3]
	default:
		return nil, fmt.Errorf("unknown beacon type")
	}

	return info, nil
}

// NewM24LRxx creates a new M24LRxx instance
func NewM24LRxx(reader pcsc.Reader) (*M24LRxx, error) {
	card, err := reader.ConnectCardPCSC()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to card: %v", err)
	}

	m24lr := &M24LRxx{
		reader: card,
	}

	err = m24lr.getUID()
	if err != nil {
		card.DisconnectCard()
		return nil, fmt.Errorf("failed to get UID: %v", err)
	}

	return m24lr, nil
}

// UID returns the UID of the tag
func (m *M24LRxx) UID() string {
	return m.uid
}

// ReadBlock reads a block from the tag
func (m *M24LRxx) ReadBlock(blockNumber int) (string, error) {
	cmd := fmt.Sprintf("FFB0%04X04", blockNumber)
	return m.transmit(cmd, 0x9000)
}

// WriteBlock writes a block to the tag
func (m *M24LRxx) WriteBlock(blockNumber int, block string) (string, error) {
	cmd := fmt.Sprintf("FFD6%04X04%s", blockNumber, block)
	return m.transmit(cmd, 0x9000)
}

// AFI returns the Application Family Identifier
func (m *M24LRxx) AFI() (string, error) {
	return m.transmit("FF30020001", 0x9000)
}

// DSFID returns the Data Storage Format Identifier
func (m *M24LRxx) DSFID() (string, error) {
	return m.transmit("FF30030001", 0x9000)
}

// MemorySize returns the memory size of the tag
func (m *M24LRxx) MemorySize() (uint16, error) {
	response, err := m.transmit("FF30040003", 0x9000)
	if err != nil {
		return 0, err
	}
	size, err := strconv.ParseUint(response[2:4], 16, 16)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory size: %v", err)
	}
	return uint16(size), nil
}

// ICReference is not supported
func (m *M24LRxx) ICReference() error {
	return fmt.Errorf("ICReference: Instruction not supported")
}

// DittoMacs holds the MAC addresses for LoRa and BLE
type DittoMacs struct {
	LoRaMAC string
	BleMac  string
}

// ReadDittoMacs reads the LoRa and BLE MAC addresses from the tag
func (m *M24LRxx) ReadDittoMacs() (*DittoMacs, error) {
	macs := &DittoMacs{}

	// Read LoRa MAC
	loraDword1, err := m.ReadBlock(11)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoRa block 11: %v", err)
	}
	loraDword2, err := m.ReadBlock(12)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoRa block 12: %v", err)
	}

	loraMacParts := []string{
		loraDword1[0:2], loraDword1[2:4], loraDword1[4:6], loraDword1[6:8],
		loraDword2[0:2], loraDword2[2:4], loraDword2[4:6], loraDword2[6:8],
	}
	macs.LoRaMAC = strings.Join(loraMacParts, ":")

	// Read BLE MAC
	bleDword1, err := m.ReadBlock(18)
	if err != nil {
		return nil, fmt.Errorf("failed to read BLE block 18: %v", err)
	}
	bleDword2, err := m.ReadBlock(19)
	if err != nil {
		return nil, fmt.Errorf("failed to read BLE block 19: %v", err)
	}

	BLEMacParts := []string{
		bleDword2[2:4], bleDword2[0:2],
		bleDword1[6:8], bleDword1[4:6], bleDword1[2:4], bleDword1[0:2],
	}
	macs.BleMac = strings.Join(BLEMacParts, ":")

	return macs, nil
}

func printDittoMacs(macs *DittoMacs) {
	fmt.Println("Ditto MACs:")
	fmt.Printf("LoRa MAC: %s\n", macs.LoRaMAC)
	fmt.Printf("BLE MAC: %s\n", macs.BleMac)

	if macs.BleMac == "00:00:00:00:00:00" {
		fmt.Println("Warning: Mac Not written, Erase tag and repeat")
	} else {
		fmt.Println("Mac Written")
	}
}

// ReadMACAddress reads the MAC address from blocks 11 and 12
func (m *M24LRxx) ReadMACAddress() (string, error) {
	LoRa_Dword1, err := m.ReadBlock(11)
	if err != nil {
		return "", fmt.Errorf("failed to read block 11: %v", err)
	}

	LoRa_Dword2, err := m.ReadBlock(12)
	if err != nil {
		return "", fmt.Errorf("failed to read block 12: %v", err)
	}

	if len(LoRa_Dword1) != 8 || len(LoRa_Dword2) != 8 {
		return "", fmt.Errorf("invalid block data length")
	}

	macParts := []string{
		LoRa_Dword1[0:2],
		LoRa_Dword1[2:4],
		LoRa_Dword1[4:6],
		LoRa_Dword1[6:8],
		LoRa_Dword2[0:2],
		LoRa_Dword2[2:4],
		LoRa_Dword2[4:6],
		LoRa_Dword2[6:8],
	}

	return strings.Join(macParts, ":"), nil
}

// ReadLocalName reads the local BLE name from blocks 3, 4, 5, and 6
func (m *M24LRxx) ReadLocalName() (string, error) {
	var readLocalValue strings.Builder

	for block := 3; block <= 6; block++ {
		blockData, err := m.ReadBlock(block)
		if err != nil {
			return "", fmt.Errorf("failed to read block %d: %v", block, err)
		}
		fmt.Printf("blockData: %s\n", blockData)
		readLocalValue.WriteString(blockData)
	}

	fmt.Printf("readLocalValue: %s\n", readLocalValue.String())
	// Convert hex string to UTF-16 and then to string
	hexBytes, err := hex.DecodeString(readLocalValue.String())
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %v", err)
	}

	utf16Ints := make([]uint16, len(hexBytes)/2)
	for i := 0; i < len(hexBytes); i += 2 {
		utf16Ints[i/2] = uint16(hexBytes[i])<<8 | uint16(hexBytes[i+1])
	}

	localOut := string(utf16.Decode(utf16Ints))

	// Remove null characters
	localOut = strings.TrimRight(localOut, "\x00")

	return localOut, nil
}

// Close disconnects the card
func (m *M24LRxx) Close() error {
	return m.reader.DisconnectCard()
}

func (m *M24LRxx) getUID() error {
	uid, err := m.transmit("FFCA000000", 0x9000)
	if err != nil {
		return err
	}
	m.uid = uid
	return nil
}

func (m *M24LRxx) transmit(cmdHex string, expectedSW uint16) (string, error) {
	cmd, err := hex.DecodeString(cmdHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode command: %v", err)
	}

	resp, err := m.reader.Apdu(cmd)
	if err != nil {
		return "", fmt.Errorf("transmission failed: %v", err)
	}

	if len(resp) < 2 {
		return "", fmt.Errorf("response too short")
	}

	sw := uint16(resp[len(resp)-2])<<8 | uint16(resp[len(resp)-1])
	if sw != expectedSW {
		return "", fmt.Errorf("unexpected status word: %04X", sw)
	}

	return hex.EncodeToString(resp[:len(resp)-2]), nil
}

func main() {
	// Initialize PCSC
	ctx, err := pcsc.NewContext()
	if err != nil {
		fmt.Printf("Failed to create PCSC context: %v\n", err)
		return
	}
	defer ctx.Release()

	// List readers
	readers, err := pcsc.ListReaders(ctx)
	if err != nil {
		fmt.Printf("Failed to list readers: %v\n", err)
		return
	}

	if len(readers) == 0 {
		fmt.Println("No readers found")
		return
	}

	// Use the first reader
	reader := pcsc.NewReader(ctx, readers[0])

	// Create M24LRxx instance
	m24lr, err := NewM24LRxx(reader)
	if err != nil {
		fmt.Printf("Failed to create M24LRxx: %v\n", err)
		return
	}
	defer m24lr.Close()

	// Example usage
	fmt.Printf("UID: %s\n", m24lr.UID())

	blockData, err := m24lr.ReadBlock(10)
	if err != nil {
		fmt.Printf("Failed to read block: %v\n", err)
	} else {
		fmt.Printf("Block 10 data: %s\n", blockData)
	}

	memSize, err := m24lr.MemorySize()
	if err != nil {
		fmt.Printf("Failed to get memory size: %v\n", err)
	} else {
		fmt.Printf("Memory size: %d\n", memSize)
	}

	// Read SKU (beacon type)
	beaconInfo, err := m24lr.ReadSKU()
	if err != nil {
		fmt.Printf("Failed to read SKU: %v\n", err)
	} else {
		printBeaconInfo(beaconInfo)
	}

	// Read Ditto MACs
	dittoMacs, err := m24lr.ReadDittoMacs()
	if err != nil {
		fmt.Printf("Failed to read Ditto MACs: %v\n", err)
	} else {
		printDittoMacs(dittoMacs)
	}

	// Read local name
	localName, err := m24lr.ReadLocalName()
	if err != nil {
		fmt.Printf("Failed to read local name: %v\n", err)
	} else {
		fmt.Printf("Local BLE Name: %s\n", localName)
	}

	// Example usage of ReadUUID
	uuidInfo, err := m24lr.ReadUUID(currentBeaconType)
	if err != nil {
		fmt.Printf("Failed to read UUID: %v\n", err)
	} else {
		fmt.Printf("UUID: %s\n", uuidInfo.UUID)
		if uuidInfo.Major != "" {
			fmt.Printf("Major: %s\n", uuidInfo.Major)
			fmt.Printf("Minor: %s\n", uuidInfo.Minor)
		}
		if uuidInfo.Instance != "" {
			fmt.Printf("Instance: %s\n", uuidInfo.Instance)
		}
	}
	// Read LoRa settings
	loraSettings, err := m24lr.ReadLoRaSettings(3) // Assuming tag type 3 (Sense Range)
	if err != nil {
		fmt.Printf("Failed to read LoRa settings: %v\n", err)
	} else {
		printLoRaSettings(loraSettings)
	}

}
