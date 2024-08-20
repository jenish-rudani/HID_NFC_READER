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

// ReadDittoSettings reads all settings from an Asset+ tag
func (m *M24LRxx) ReadDittoSettings() (*DittoSettings, error) {
	settings := &DittoSettings{}

	// Read required blocks
	blocks := make(map[int]string)
	for _, blockNum := range []int{7, 8, 9, 13, 14, 15, 19, 20, 21, 24, 25, 26, 27, 28, 29, 30, 31} {
		block, err := m.ReadBlock(blockNum)
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %v", blockNum, err)
		}
		blocks[blockNum] = block
	}

	// Parse Block 15: 034A1504
	settings.HardwareVersion = blocks[15][1:2]
	fwInt, _ := strconv.ParseInt(blocks[15][2:4], 16, 0)
	settings.FirmwareVersion = fmt.Sprintf("%.1f", float64(fwInt)/10)
	settings.BeaconType, _ = strconv.Atoi(blocks[15][4:6])

	/*
		Block 0: 53706563
		Block 1: 74726531
		Block 2: 00000000
		Block 3: 56432040
		Block 4: 38065F39
		Block 5: 2E361753
		Block 6: 4502330C
		Block 7: 01080000
		Block 10: 00000000
		Block 11: 0C1EF700
		Block 12: 00000D27
		Block 14: 000F3200
		Block 16: FA00A60E
		Block 17: 8E122C01
		Block 18: E3423564
		Block 19: A6CE00F4
		Block 20: 00000200
		Block 21: 05000A04
		Block 22: 53503430
		Block 23: 36360000
		Block 24: C4091027
		Block 25: A64F6D6E
		Block 26: 692D4944
		Block 27: 00000000
		Block 28: 00000000
		Block 29: 00000007
		Block 30: 3C000002
		Block 31: 01000000
	*/

	// Parse Block 13: 10100000
	var err error
	blocks[13], err = m.ReadBlock(13)
	if err != nil {
		return nil, fmt.Errorf("failed to read block %d: %v", 13, err)
	}
	settings.SleepState = parseSleepState(blocks[13][2:3])
	settings.DebugOption = parseDebugOption(blocks[13][3:4])
	settings.MACOption = parseMACOption(blocks[13][4:5])

	// Parse Block 8: 00010000
	settings.SpreadingFactor = parseSpreadingFactor(blocks[8][:2])
	settings.DownlinkBitRate, _ = strconv.Atoi(blocks[8][2:4])
	settings.UplinkBitRate, _ = strconv.Atoi(blocks[8][4:6])
	tempHigh, _ := strconv.ParseInt(blocks[8][6:], 16, 0)
	settings.HighTemperature = int(tempHigh) - 127

	// Parse Block 9: 00050000
	tempLow, _ := strconv.ParseInt(blocks[9][:2], 16, 0)
	settings.LowTemperature = int(tempLow) - 127
	settings.Accelerometer, _ = strconv.Atoi(blocks[9][2:4])

	// Parse Block 14
	settings.GNSSMin, _ = strconv.Atoi(blocks[14][:2])
	settings.GNSSMax, _ = strconv.Atoi(blocks[14][2:4])
	dop, _ := strconv.ParseFloat(blocks[14][4:6], 64)
	settings.DOP = dop / 10
	settings.OperationalMode, _ = strconv.Atoi(blocks[14][6:])

	// Parse Block 7
	settings.LoRaEnable = parseLoRaEnable(blocks[7][:2])
	settings.LoRaRegion = parseLoRaRegion(blocks[7][2:4])

	// Parse Block 19
	settings.ABR2, _ = strconv.Atoi(blocks[19][4:6])
	settings.BLEGain = parseBLEGain(blocks[19][6:])

	// Parse Block 20
	motionMoved, _ := strconv.ParseInt(blocks[20][4:]+blocks[20][6:], 16, 0)
	settings.MotionMoved = int(motionMoved)

	// Parse Block 21
	motionStationary, _ := strconv.ParseInt(blocks[21][:2]+blocks[21][2:4], 16, 0)
	settings.MotionStationary = int(motionStationary)
	settings.MotionAccelActivity, _ = strconv.Atoi(blocks[21][4:6])
	settings.MotionAccelActivityThreshold, _ = strconv.Atoi(blocks[21][6:])

	// Parse Blocks 24-29
	settings.BLEAdvertisingInterval, _ = strconv.ParseInt(blocks[24][:4], 16, 0)
	settings.BLERefScanInterval, _ = strconv.ParseInt(blocks[24][4:], 16, 0)
	settings.BLERefRSSI = complementToDec(blocks[25][:2])
	settings.BLERefFilter = parseRefFilter(blocks[25][2:] + blocks[26] + blocks[27] + blocks[28] + blocks[29][:2])
	settings.BLEAdvertisingType = parseBLEAdvertisingType(blocks[29][2:4])
	settings.PressUplink = parsePressUplink(blocks[29][4:6])
	settings.PingSlotPeriod = parsePingSlotPeriod(blocks[29][6:])

	// Parse Block 30
	settings.Timeout, _ = strconv.Atoi(blocks[30][:2])

	// Parse Flags
	flags1, _ := strconv.ParseUint(blocks[30][6:], 16, 8)
	flags2, _ := strconv.ParseUint(blocks[31][:2], 16, 8)
	settings.BLERefMode = parseBLERefMode(flags1)
	settings.ClassSelect = parseClassSelect(flags1)
	settings.ConfirmedUplinks = parseConfirmedUplinks(flags2)
	settings.Hopping = parseHopping(flags2)

	return settings, nil
}

type DittoSettings struct {
	BeaconType                   int
	HardwareVersion              string
	FirmwareVersion              string
	SleepState                   string
	DebugOption                  string
	MACOption                    string
	SpreadingFactor              string
	DownlinkBitRate              int
	UplinkBitRate                int
	HighTemperature              int
	LowTemperature               int
	Accelerometer                int
	GNSSMin                      int
	GNSSMax                      int
	DOP                          float64
	OperationalMode              int
	LoRaEnable                   string
	LoRaRegion                   string
	ABR2                         int
	BLEGain                      string
	MotionMoved                  int
	MotionStationary             int
	MotionAccelActivity          int
	MotionAccelActivityThreshold int
	BLEAdvertisingInterval       int64
	BLERefScanInterval           int64
	BLERefRSSI                   int
	BLERefFilter                 string
	BLEAdvertisingType           string
	PressUplink                  string
	PingSlotPeriod               string
	Timeout                      int
	BLERefMode                   string
	ClassSelect                  string
	ConfirmedUplinks             string
	Hopping                      string
}

func parseSleepState(state string) string {
	switch state {
	case "0":
		return "Asleep"
	case "1":
		return "Awake"
	default:
		return "Unknown"
	}
}

func parseDebugOption(option string) string {
	switch option {
	case "0":
		return "Tones Disabled"
	case "1":
		return "Tones Enabled"
	default:
		return "Unknown"
	}
}

func parseMACOption(option string) string {
	switch option {
	case "0":
		return "LoRa Module"
	case "1":
		return "LoRa DevEUI"
	default:
		return "Unknown"
	}
}

func parseSpreadingFactor(sf string) string {
	sfInt, _ := strconv.ParseInt(sf, 16, 0)
	if sfInt == 255 {
		return "ADR"
	}
	return fmt.Sprintf("%d", sfInt)
}

func parseLoRaEnable(enable string) string {
	enableInt, _ := strconv.Atoi(enable)
	if enableInt == 1 {
		return "Enabled"
	}
	return "Disabled"
}

func parseLoRaRegion(region string) string {
	regionInt, _ := strconv.Atoi(region)
	regions := map[int]string{
		0: "AS 923MHz_GRP1", 1: "AU 915MHz", 5: "EU 868MHz",
		6: "SK 930MHz", 7: "IN 865MHz", 8: "US 915MHz",
		10: "AS923_GRP2", 11: "AS923_GRP3",
	}
	if r, ok := regions[regionInt]; ok {
		return r
	}
	return "Not Selected"
}

func parseBLEGain(gain string) string {
	gains := map[string]string{
		"D8": "-40dBm", "EC": "-20dBm", "F0": "-16dBm",
		"F4": "-12dBm", "F8": "-8dBm", "FC": "-4dBm",
		"00": "0dBm", "03": "3dBm", "04": "4dBm",
	}
	if g, ok := gains[gain]; ok {
		return g
	}
	return "Unknown"
}

func parseRefFilter(filter string) string {
	decoded := ""
	for i := 0; i < len(filter); i += 2 {
		b, _ := strconv.ParseInt(filter[i:i+2], 16, 0)
		decoded += string(rune(b))
	}
	return decoded
}

func parseBLEAdvertisingType(adType string) string {
	if adType == "01" {
		return "sBeacon"
	}
	return "Default"
}

func parsePressUplink(uplink string) string {
	if uplink == "01" {
		return "Disabled"
	}
	return "Enabled"
}

func parsePingSlotPeriod(period string) string {
	periodInt, _ := strconv.Atoi(period)
	periods := map[int]string{
		0: "1S", 1: "2S", 2: "4S", 3: "8S",
		4: "16S", 5: "32S", 6: "64S", 7: "128S",
	}
	if p, ok := periods[periodInt]; ok {
		return p
	}
	return "Unknown"
}

func parseBLERefMode(flags uint64) string {
	switch flags & 0b11 {
	case 0b10:
		return "Reference Tags"
	case 0b01:
		return "BluFi"
	default:
		return "Disabled"
	}
}

func parseClassSelect(flags uint64) string {
	switch (flags >> 4) & 0b11 {
	case 0b10:
		return "Class B"
	case 0b01:
		return "Class C"
	default:
		return "Class A"
	}
}

func parseConfirmedUplinks(flags uint64) string {
	if flags&0b1 == 0b1 {
		return "Enabled"
	}
	return "Disabled"
}

func parseHopping(flags uint64) string {
	if (flags>>4)&0b1 == 0b1 {
		return "Enabled"
	}
	return "Disabled"
}

func complementToDec(hex string) int {
	i, _ := strconv.ParseInt(hex, 16, 0)
	if i > 127 {
		return int(i - 256)
	}
	return int(i)
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
	printStructuredOutput(m24lr)

	settings, err := m24lr.ReadDittoSettings()
	if err != nil {
		fmt.Printf("Failed to read Ditto settings: %v\n", err)
	} else {
		PrintMappedDittoSettings(settings)
	}
}

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
	})

	printSection("Beacon Information", func() {
		beaconInfo, _ := m24lr.ReadSKU()
		printField("Type", beaconInfo.BeaconType)
		printField("Name", beaconInfo.Name)
		printField("Image", beaconInfo.Image)
	})

	printSection("Ditto MACs", func() {
		macs, _ := m24lr.ReadDittoMacs()
		printField("LoRa MAC", strings.ToUpper(macs.LoRaMAC))
		printField("BLE MAC", strings.ToUpper(macs.BleMac))
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

func PrintMappedDittoSettings(settings *DittoSettings) {
	fmt.Println()
	fmt.Println(colorGreen + "=== Asset+ Tag Settings ===" + colorReset)

	printMappedField("Hardware Version", settings.HardwareVersion)
	printMappedField("Firmware Version", settings.FirmwareVersion)
	printMappedField("Beacon Type", strconv.Itoa(settings.BeaconType))
	printMappedField("Debug Tones", settings.DebugOption)
	printMappedField("BLE TX PWR", settings.BLEGain)
	printMappedField("LoRa Region", settings.LoRaRegion)
	printMappedField("Stationary -> Moved Threshold", fmt.Sprintf("%d", settings.MotionMoved))
	printMappedField("Moved ->Stationary Threshold", fmt.Sprintf("%d", settings.MotionStationary))
	printMappedField("Activity Window", fmt.Sprintf("%d", settings.MotionAccelActivity))
	printMappedField("Activity Threshold", fmt.Sprintf("%d", settings.MotionAccelActivityThreshold))
	printMappedField("Motion Threshold", fmt.Sprintf("%d", settings.Accelerometer))
	printMappedField("HBR in Hours", fmt.Sprintf("%d", settings.DownlinkBitRate)) // Assuming HBR is stored in DownlinkBitRate
	printMappedField("ABR in minutes", fmt.Sprintf("%d", settings.ABR2))
	printMappedField("Tag Status", settings.SleepState)
	printMappedField("GNSS Max Lock Time in Minutes", fmt.Sprintf("%d", settings.GNSSMax))
	printMappedField("DOP Threshold", fmt.Sprintf("%.1f", settings.DOP))
	printMappedField("Post Movement/DR", "0") // Not clear where this is stored, using a default value
	printMappedField("LoRa Enable", settings.LoRaEnable)
	printMappedField("Button Press Uplink", settings.PressUplink)
	printMappedField("BLE Advertising Type", settings.BLEAdvertisingType)
	printMappedField("BLE Advertising rate in mS", fmt.Sprintf("%d", settings.BLEAdvertisingInterval))
	printMappedField("Position Engine BLE Scan", settings.BLERefMode)
	printMappedField("Position Engine BLE Scan duration in mS", fmt.Sprintf("%d", settings.BLERefScanInterval))
	printMappedField("BLE Reference Tag Filter ID", settings.BLERefFilter)
	printMappedField("BLE Scan RSSI Threshold", fmt.Sprintf("%d", settings.BLERefRSSI))
	printMappedField("LoRaWAN Class B Ping Slot", settings.PingSlotPeriod)
	printMappedField("LoRaWAN Class B Timeout", fmt.Sprintf("%d", settings.Timeout))
	printMappedField("LoRaWAN Class select", settings.ClassSelect)
	printMappedField("LoRaWAN Confirmed Uplinks", settings.ConfirmedUplinks)
	printMappedField("LoRaWAN Sub-band Hopping", settings.Hopping)
}

func printMappedField(label, value string) {
	fmt.Printf("%s%s:%s %s\n", colorYellow, label, colorReset, value)
}
