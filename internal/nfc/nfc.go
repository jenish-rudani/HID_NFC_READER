package nfc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"bitbucket.org/bluvision-cloud/kit/log"
	"bitbucket.org/bluvision/pcsc/pcsc"
)

// NfcCard represents a M24LR series RFID tag
type NfcCard struct {
	uid    string
	Reader pcsc.Card
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
func (m *NfcCard) ReadLoRaSettings(tagType int) (*LoRaSettings, error) {
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
	log.Infof("Beacon Type: %d\n", settings.BeaconType)
	log.Infof("Hardware Version: %s\n", settings.HardwareVersion)
	log.Infof("Firmware Version: %s\n", settings.FirmwareVersion)
	log.Infof("Sleep State: %s\n", settings.SleepState)
	log.Infof("Min/Max Threshold: %s\n", settings.MinMaxThreshold)
	log.Infof("Range Type: %s\n", settings.RangeType)
	log.Infof("Spreading Factor: %s\n", settings.SpreadingFactor)
	log.Infof("Downlink Bit Rate: %d\n", settings.DownlinkBitRate)
	log.Infof("Uplink Bit Rate: %d\n", settings.UplinkBitRate)
	log.Infof("High Temperature: %d\n", settings.HighTemperature)
	log.Infof("Low Temperature: %d\n", settings.LowTemperature)
	log.Infof("Accelerometer: %d\n", settings.Accelerometer)
	log.Infof("GNSS Min: %d\n", settings.GNSSMin)
	log.Infof("GNSS Max: %d\n", settings.GNSSMax)
	log.Infof("DOP: %.1f\n", settings.DOP)
	log.Infof("Range Threshold: %d\n", settings.RangeThreshold)
	log.Infof("Sensor Period: %d\n", settings.SensorPeriod)
	log.Infof("Range Offset: %d\n", settings.RangeOffset)
	log.Infof("Maximum Range: %d\n", settings.MaximumRange)
}

// BeaconInfo holds information about the beacon type
type BeaconInfo struct {
	BeaconType string
	Name       string
	Image      string // In Go, we'll just store the image name/path
}

// APDUInfo holds the parsed information from an APDU response
type APDUInfo struct {
	Tag         byte
	Value       string
	ValueRaw    []byte
	StatusWords [2]byte
}

func ParseAPDU(apdu []uint8) (APDUInfo, error) {
	info := APDUInfo{}

	if len(apdu) < 5 {
		return info, fmt.Errorf("APDU too short")
	}

	// Extract status words (last two bytes)
	info.StatusWords = [2]byte{apdu[len(apdu)-2], apdu[len(apdu)-1]}

	// Check for successful processing (SW1SW2 = 9000)
	if info.StatusWords != [2]byte{0x90, 0x00} {
		return info, fmt.Errorf("unsuccessful processing: SW1SW2 = %02X%02X", info.StatusWords[0], info.StatusWords[1])
	}

	// Check for the response tag (BD or 9D)
	if apdu[0] != 0xBD && apdu[0] != 0x9D {
		return info, fmt.Errorf("unexpected response tag: %02X", apdu[0])
	}

	// Extract the inner TLV
	innerLength := int(apdu[1])
	if len(apdu) < innerLength+2 {
		return info, fmt.Errorf("APDU length mismatch")
	}

	info.Tag = apdu[2]
	valueLength := int(apdu[3])
	valueEnd := 4 + valueLength

	if valueEnd > len(apdu)-2 {
		return info, fmt.Errorf("invalid value length")
	}

	// Extract the value based on the tag
	switch info.Tag {
	case 0x02, 0x92: // Product name or Serial number
		info.Value = string(apdu[4:valueEnd])
		info.ValueRaw = apdu[4:valueEnd]
		// Remove null terminator if present
		info.Value = strings.TrimRight(info.Value, "\x00")
	default:
		// For other tags, just store the hex representation
		info.Value = fmt.Sprintf("%X", apdu[4:valueEnd])
		info.ValueRaw = apdu[4:valueEnd]
	}

	return info, nil
}

// ReadSKU reads the SKU (beacon type) from the tag
func (m *NfcCard) ReadSKU() (*BeaconInfo, error) {
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
	log.Infof("Type: %s\n", info.BeaconType)
	log.Infof("Name: %s\n", info.Name)
	log.Infof("Image: %s\n", info.Image)
}

// LoraInfo Structure to hold LoRa information
type LoraInfo struct {
	Timestamp string
	DevEUI    string
	JoinEUI   string
	JoinKey   string
	CRCStatus string
}

func (m *NfcCard) ReadLoraInfo() (*LoraInfo, error) {
	info := &LoraInfo{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Read DevEUI
	devEui, err := m.ReadLoraDevEui()
	if err != nil {
		return nil, fmt.Errorf("failed to read DevEUI: %v", err)
	}
	info.DevEUI = strings.ToUpper(devEui)

	// Read JoinEUI
	joinEui, err := m.ReadLoraJoinEui()
	if err != nil {
		return nil, fmt.Errorf("failed to read JoinEUI: %v", err)
	}
	info.JoinEUI = strings.ToUpper(joinEui)

	// Read JoinKey
	joinKey, err := m.ReadLoraJoinKey()
	if err != nil {
		return nil, fmt.Errorf("failed to read JoinKey: %v", err)
	}
	info.JoinKey = strings.ToUpper(joinKey)

	// Validate CRC
	err = m.ValidateCRC()
	if err != nil {
		info.CRCStatus = "INVALID"
	} else {
		info.CRCStatus = "VALID"
	}

	return info, nil
}

func (m *NfcCard) EraseTag() error {
	log.Info("Starting NFC tag erasure by writing 0xFFFFFFFF to all blocks...")

	// Write 0xFFFFFFFF to blocks 0-48
	zeroBlock := "ffffffff" // 4 bytes of Fs in hex

	for block := 0; block <= 48; block++ {
		log.Infof("Erasing block %d...", block)
		_, err := m.WriteBlock(block, zeroBlock)
		if err != nil {
			return fmt.Errorf("failed to erase block %d: %v", block, err)
		}
	}

	// Calculate and write CRC for the zeroed configuration
	err := m.CalculateAndWriteCRC()
	if err != nil {
		return fmt.Errorf("failed to write CRC after erasure: %v", err)
	}

	log.Info("NFC tag erasure completed successfully")
	return nil
}

// ReadUUID reads the UUID and related information based on the beacon type
func (m *NfcCard) ReadUUID(beaconType uint64) (*UUIDInfo, error) {
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

// NewCardReader creates a new NfcCard instance
func NewCardReader(reader pcsc.Reader) (*NfcCard, error) {

	card, err := reader.ConnectCardPCSC()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to card: %v", err)
	}

	m24lr := &NfcCard{
		Reader: card,
	}

	err = m24lr.getUID()
	if err != nil {
		card.DisconnectCard()
		return nil, fmt.Errorf("failed to get UID: %v", err)
	}

	return m24lr, nil
}

// UID returns the UID of the tag
func (m *NfcCard) UID() string {
	return m.uid
}

// ReadBlock reads a block from the tag
func (m *NfcCard) ReadBlock(blockNumber int) (string, error) {
	cmd := fmt.Sprintf("FFB0%04X04", blockNumber)
	return m.transmit(cmd, 0x9000)
}

// WriteBlock writes a block to the tag
func (m *NfcCard) WriteBlock(blockNumber int, block string) (string, error) {
	cmd := fmt.Sprintf("FFD6%04X04%s", blockNumber, block)
	return m.transmit(cmd, 0x9000)
}

// AFI returns the Application Family Identifier
func (m *NfcCard) AFI() (string, error) {
	return m.transmit("FF30020001", 0x9000)
}

// DSFID returns the Data Storage Format Identifier
func (m *NfcCard) DSFID() (string, error) {
	return m.transmit("FF30030001", 0x9000)
}

// MemorySize returns the memory size of the tag
func (m *NfcCard) MemorySize() (uint16, error) {
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
func (m *NfcCard) ICReference() error {
	return fmt.Errorf("ICReference: Instruction not supported")
}

func (m *NfcCard) transmit(cmdHex string, expectedSW uint16) (string, error) {
	cmd, err := hex.DecodeString(cmdHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode command: %v", err)
	}

	resp, err := m.Reader.Apdu(cmd)
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

// ReadBleMac reads the LoRa and BLE MAC addresses from the tag
func (m *NfcCard) ReadBleMac() (string, error) {

	// Read BLE MAC
	bleDword1, err := m.ReadBlock(18)
	if err != nil {
		return "", fmt.Errorf("failed to read BLE block 18: %v", err)
	}
	bleDword2, err := m.ReadBlock(19)
	if err != nil {
		return "", fmt.Errorf("failed to read BLE block 19: %v", err)
	}

	BLEMacParts := []string{
		bleDword2[2:4], bleDword2[0:2],
		bleDword1[6:8], bleDword1[4:6], bleDword1[2:4], bleDword1[0:2],
	}
	BleMac := strings.Join(BLEMacParts, ":")

	return BleMac, nil
}

// ReadMACAddress reads the MAC address from blocks 11 and 12
func (m *NfcCard) ReadMACAddress() (string, error) {
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
func (m *NfcCard) ReadLocalName() (string, error) {
	var readLocalValue strings.Builder

	for block := 3; block <= 6; block++ {
		blockData, err := m.ReadBlock(block)
		if err != nil {
			return "", fmt.Errorf("failed to read block %d: %v", block, err)
		}
		log.Infof("blockData: %s\n", blockData)
		readLocalValue.WriteString(blockData)
	}

	log.Infof("readLocalValue: %s\n", readLocalValue.String())
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
func (m *NfcCard) Close() error {
	return m.Reader.DisconnectUnpowerCard()
}

func (m *NfcCard) ReadBLELocalName() (string, error) {
	log.Info("Reading BLE local name: ")
	block22, err := m.ReadBlock(22)
	if err != nil {
		return "", err
	}
	block23, err := m.ReadBlock(23)
	if err != nil {
		return "", err
	}
	localBleName := fmt.Sprintf("%s%s", block22, block23)
	ascii, err := decodeHexToASCII(localBleName)
	if err != nil {
		return "", err
	}
	log.Infof("RawBlockData: %s, ASCII: %s\n", localBleName, ascii)
	return ascii, nil
}

// WriteLoraJoinEui writes the LoRa Join EUI to blocks 0 and 1, this is the unique 64 bits EUI from the network (eg. Senet)
func (m *NfcCard) WriteLoraJoinEui(joinEui string) error {

	//Split the key into 2 blocks
	block1 := joinEui[:8]
	block2 := joinEui[8:]

	_, err := m.WriteBlock(0, block1)
	if err != nil {
		return err
	}
	_, err = m.WriteBlock(1, block2)
	if err != nil {
		return err
	}
	return m.CalculateAndWriteCRC()
}

func (m *NfcCard) ReadLoraJoinEui() (string, error) {
	joinEuiWord1, err := m.ReadBlock(0)
	if err != nil {
		return "", err
	}
	joinEuiWord2, err := m.ReadBlock(1)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", joinEuiWord1, joinEuiWord2), nil
}

// WriteLoraJoinKey writes the LoRa App Key to blocks 3, 4, 5, and 6, this is the random 128 bits key
func (m *NfcCard) WriteLoraJoinKey(loraAppKey string) error {
	if len(loraAppKey) > 32 {
		return fmt.Errorf("invalid LoRa App Key length, should be 32 characters in hex")
	}
	var block [4]string

	//Split the key into 4 blocks
	block[0] = loraAppKey[:8]
	block[1] = loraAppKey[8:16]
	block[2] = loraAppKey[16:24]
	block[3] = loraAppKey[24:]

	for i := 3; i <= 6; i++ {
		_, err := m.WriteBlock(i, block[i-3])
		if err != nil {
			return fmt.Errorf("failed to read block %d: %v", i, err)
		}
	}
	return m.CalculateAndWriteCRC()
}

// ReadLoraJoinKey reads the LoRa network key from blocks 7, 8, 9, and 10, this is the random 128 bits key
func (m *NfcCard) ReadLoraJoinKey() (string, error) {
	blocks := make(map[int]string)
	for i := 3; i <= 6; i++ {
		block, err := m.ReadBlock(i)
		if err != nil {
			return "", fmt.Errorf("failed to read block %d: %v", i, err)
		}
		blocks[i] = block
	}
	joinKey := fmt.Sprintf("%s%s%s%s", blocks[3], blocks[4], blocks[5], blocks[6])
	return joinKey, nil
}

// decodeHexToASCII decodes a hex string to ASCII
func decodeHexToASCII(hexString string) (string, error) {
	// Remove any spaces from the hex string
	hexString = strings.ReplaceAll(hexString, " ", "")

	// Decode hex to bytes
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return "", fmt.Errorf("error decoding hex string: %v", err)
	}

	// Convert bytes to string (ASCII)
	asciiString := string(bytes)

	// Remove null characters from the end of the string
	asciiString = strings.TrimRight(asciiString, "\x00")

	return asciiString, nil
}

func (m *NfcCard) WriteTagSleepBit(bitValue bool) error {
	log.Infof("Writing Tag Sleep Bit: %t", bitValue)
	block13, err := m.ReadBlock(13)
	if err != nil {
		log.Errorf("Failed to read block 13: %v", err)
		return err
	}
	finalBlock13 := ""
	log.Infof("Read Block 9: %s", block13)
	if bitValue {
		finalBlock13 = fmt.Sprintf("%s%s%s", block13[0:2], "0", block13[3:])
	} else {
		finalBlock13 = fmt.Sprintf("%s%s%s", block13[0:2], "1", block13[3:])
	}
	log.Infof("Final Block 13: %s", finalBlock13)
	_, err = m.WriteBlock(13, finalBlock13)
	if err != nil {
		log.Errorf("Failed to write block 13: %v", err)
		return err
	}
	return m.CalculateAndWriteCRC()
}

func (m *NfcCard) WriteBLELocalName(name string) error {
	// Convert ASCII to asciiToHex
	asciiToHex := encodeASCIIToHex(name)
	log.Infof("bleLocalName asciiToHex: %s\n", asciiToHex)
	// Ensure the asciiToHex string is exactly 8 bytes (16 characters)
	if len(asciiToHex) > 16 {
		return fmt.Errorf("name too long, maximum 8 bytes allowed")
	}
	asciiToHex = fmt.Sprintf("%-16s", asciiToHex)         // Pad with spaces at the end if shorter
	asciiToHex = strings.ReplaceAll(asciiToHex, " ", "0") // Replace spaces with zeroes, :(

	// Split the asciiToHex string into two 8-byte blocks
	block22 := asciiToHex[:8]
	block23 := asciiToHex[8:]
	fmt.Printf("WriteBLELocalName blockData: %s | %s\n", block22, block23)
	// Write to block 22
	_, err := m.WriteBlock(22, block22)
	if err != nil {
		return fmt.Errorf("failed to write block 22: %w", err)
	}

	// Write to block 23
	_, err = m.WriteBlock(23, block23)
	if err != nil {
		return fmt.Errorf("failed to write block 23: %w", err)
	}

	return m.CalculateAndWriteCRC()
}

// Helper function to encode ASCII to hex
func encodeASCIIToHex(s string) string {
	var hexString string
	for _, c := range s {
		hexString += fmt.Sprintf("%02x", c)
	}
	return hexString
}

func (m *NfcCard) ReadLoraDevEui() (string, error) {
	// Read LoRa MAC
	loraDword1, err := m.ReadBlock(11)
	if err != nil {
		return "", fmt.Errorf("failed to read LoRa block 11: %v", err)
	}
	loraDword2, err := m.ReadBlock(12)
	if err != nil {
		return "", fmt.Errorf("failed to read LoRa block 12: %v", err)
	}

	devEuiParts := []string{
		loraDword1[0:2], loraDword1[2:4], loraDword1[4:6], loraDword1[6:8],
		loraDword2[0:2], loraDword2[2:4], loraDword2[4:6], loraDword2[6:8],
	}

	devEui := strings.Join(devEuiParts, ":")
	devEuiUpperCase := strings.ToUpper(devEui)

	return devEuiUpperCase, nil
}

func (m *NfcCard) WriteTagPostBit(bitValue bool) error {
	log.Infof("Writing Tag Post Bit, %v", bitValue)
	block9, err := m.ReadBlock(0x09)
	if err != nil {
		log.Errorf("Failed to read block 9: %v", err)
		return err
	}
	log.Infof("Read Block 9: %s", block9)

	finalBlock9 := ""
	if bitValue {
		finalBlock9 = "01" + block9[2:]
	} else {
		finalBlock9 = "00" + block9[2:]
	}
	log.Infof("Final Block 9: %s", finalBlock9[:8])
	_, err = m.WriteBlock(0x09, finalBlock9[:8])
	if err != nil {
		log.Errorf("Failed to write block 9: %v", err)
		return err
	}
	return m.CalculateAndWriteCRC()
}

func (m *NfcCard) getUID() error {
	uid, err := m.transmit("FFCA000000", 0x9000)
	if err != nil {
		return err
	}
	m.uid = uid
	return nil
}

// ReadDittoSettings reads all settings from an Asset+ tag
func (m *NfcCard) ReadDittoSettings() (*DittoSettings, error) {
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

	// Parse Block 14: 000F3200
	// Parse Block 14: 000F3200
	settings.GNSSMin, _ = strconv.ParseInt(blocks[14][:2], 16, 64)
	settings.GNSSMax, _ = strconv.ParseInt(blocks[14][2:4], 16, 64)
	dop, _ := strconv.ParseInt(blocks[14][4:6], 16, 64)
	settings.DOP = float64(dop / 10)
	settings.OperationalMode, _ = strconv.ParseInt(blocks[14][6:], 16, 64)

	// Parse Block 7: 01080000
	fmt.Println(blocks[7])
	settings.LoRaEnable = parseLoRaEnable(blocks[7][:2])
	settings.LoRaRegion = parseLoRaRegion(blocks[7][2:4])

	// Parse Block 19: A6CE00F4
	settings.ABR2, _ = strconv.Atoi(blocks[19][4:6])
	settings.BLEGain = parseBLEGain(blocks[19][6:])

	// Parse Block 20: 00000200
	motionMovedHex := fmt.Sprintf("%04s", blocks[20][4:])            // This is equivalent to Substring(4, 4).PadLeft(4, "0")
	motionMovedRearranged := motionMovedHex[2:] + motionMovedHex[:2] // Rearrange the string
	motionMoved, _ := strconv.ParseInt(motionMovedRearranged, 16, 64)
	settings.MotionMoved = int(motionMoved)

	// Parse Block 21: 05000A04
	motionStationaryHex := fmt.Sprintf("%04s", blocks[21][:4]) // This pads the string to 4 characters if needed
	motionStationaryRearranged := motionStationaryHex[2:] + motionStationaryHex[:2]
	motionStationary, _ := strconv.ParseInt(motionStationaryRearranged, 16, 64)
	settings.MotionStationary = int(motionStationary)
	settings.MotionAccelActivity, _ = strconv.ParseInt(blocks[21][4:6], 16, 64)
	settings.MotionAccelActivityThreshold, _ = strconv.ParseInt(blocks[21][6:], 16, 64)

	// Parse Blocks 24-29: C4091027, A64F6D6E, 692D4944, 00000000, 00000000, 00000007
	BleAdvRateHex := fmt.Sprintf("%04s", blocks[24][:4])
	BleAdvRateHexRearranged := BleAdvRateHex[2:] + BleAdvRateHex[:2]
	settings.BLEAdvertisingInterval, _ = strconv.ParseInt(BleAdvRateHexRearranged, 16, 64)

	BleRfScanIntervalHex := fmt.Sprintf("%04s", blocks[24][4:])
	BleRfScanInterValRearranged := BleRfScanIntervalHex[2:] + BleRfScanIntervalHex[:2]
	settings.BLERefScanInterval, _ = strconv.ParseInt(BleRfScanInterValRearranged, 16, 64)
	settings.BLERefRSSI = complementToDec(blocks[25][:2])
	settings.BLERefFilter = parseRefFilter(blocks[25][2:] + blocks[26] + blocks[27] + blocks[28] + blocks[29][:2])
	settings.BLEAdvertisingType = parseBLEAdvertisingType(blocks[29][2:4])
	settings.PressUplink = parsePressUplink(blocks[29][4:6])
	settings.PingSlotPeriod = parsePingSlotPeriod(blocks[29][6:])

	// Parse Block 30: 3C000002
	settings.Timeout, _ = strconv.ParseInt(blocks[30][:2], 16, 8)

	// Parse Flags from Block 30 and 31
	flags1, _ := strconv.ParseUint(blocks[30][6:], 16, 8)
	flags2, _ := strconv.ParseUint(blocks[31][:2], 16, 8)
	flags1Bin := fmt.Sprintf("%08b", flags1) // Convert to 8-bit binary string
	flags2Bin := fmt.Sprintf("%08b", flags2) // Convert to 8-bit binary string

	// Reverse the bits
	var reversedFlags1Bin string
	var reversedFlags2Bin string
	for i := len(flags1Bin) - 1; i >= 0; i-- {
		reversedFlags1Bin += string(flags1Bin[i])
	}
	for i := len(flags2Bin) - 1; i >= 0; i-- {
		reversedFlags2Bin += string(flags2Bin[i])
	}

	settings.BLERefMode = parseBLERefMode(reversedFlags1Bin[:2])

	settings.ClassSelect = parseClassSelect(reversedFlags1Bin[4:6])
	settings.ConfirmedUplinks = parseConfirmedUplinks(reversedFlags2Bin[:1])
	settings.Hopping = parseHopping(reversedFlags2Bin[4:5])

	return settings, nil
}

// ReadConfigurationForCRC reads blocks 0-47 and prepares data for CRC calculation
func (m *NfcCard) ReadConfigurationForCRC() ([]byte, error) {
	nfcData := make([]byte, 0, 192) // 48 blocks * 4 bytes per block = 192 bytes

	// Read blocks 0 to 47
	for block := 0; block <= 47; block++ {
		blockData, err := m.ReadBlock(block)
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %v", block, err)
		}

		// Convert hex string to bytes
		// Each block contains 8 hex chars representing 4 bytes
		bytes, err := hex.DecodeString(blockData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode block %d data: %v", block, err)
		}

		// Append the 4 bytes to nfcData
		nfcData = append(nfcData, bytes...)
	}

	return nfcData, nil
}

// calculateCRC implements the CRC-16 CCITT algorithm
func calculateCRC(data []byte) uint16 {
	crc := uint16(0xFFFF)        // Initial value
	polynomial := uint16(0x1021) // CRC-16 CCITT polynomial
	for i := 0; i < len(data); i++ {
		crc ^= uint16(uint16(data[i]) << 8)
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = uint16((uint16(crc) << 1) ^ polynomial)
			} else {
				crc <<= 1
			}
		}
	}
	return uint16(crc & 0xFFFF)
}

// CalculateAndWriteCRC calculates CRC for all configuration blocks and writes it
func (m *NfcCard) CalculateAndWriteCRC() error {
	// Read all configuration data
	nfcData, err := m.ReadConfigurationForCRC()
	if err != nil {
		return fmt.Errorf("failed to read configuration: %v", err)
	}

	// Calculate CRC
	crc := calculateCRC(nfcData)

	// Reverse the CRC bytes and format for writing
	reversedCRCHex := reverseCRC(crc)
	log.Infof("Calculated CRC: 0x%04X, Reversed for storage: 0x%s", crc, reversedCRCHex)

	// Write reversed CRC to designated block
	_, err = m.WriteBlock(48, reversedCRCHex)
	if err != nil {
		return fmt.Errorf("failed to write CRC: %v", err)
	}

	return nil
}

// ValidateCRC reads the configuration and validates against stored CRC
func (m *NfcCard) ValidateCRC() error {
	log.Info("Validating CRC...")
	// Read configuration data
	nfcData, err := m.ReadConfigurationForCRC()
	if err != nil {
		return fmt.Errorf("failed to read configuration: %v", err)
	}

	// Calculate CRC
	calculatedCRC := calculateCRC(nfcData)

	// Read stored CRC from block 48
	storedCRCBlock, err := m.ReadBlock(48)
	if err != nil {
		return fmt.Errorf("failed to read CRC block: %v", err)
	}

	// Extract and reverse the stored CRC bytes
	lsb, _ := strconv.ParseUint(storedCRCBlock[0:2], 16, 8)
	msb, _ := strconv.ParseUint(storedCRCBlock[2:4], 16, 8)
	storedCRC := uint16(msb)<<8 | uint16(lsb)

	log.Infof("Calculated CRC: 0x%04X, Stored CRC: 0x%04X", calculatedCRC, storedCRC)

	// Compare CRCs
	if calculatedCRC != storedCRC {
		return fmt.Errorf("CRC validation failed: calculated=0x%04X, stored=0x%04X",
			calculatedCRC, storedCRC)
	}

	return nil
}

// WriteLoraDevEui Modified write methods to update CRC
func (m *NfcCard) WriteLoraDevEui(loraDevEui string) error {
	// Write DevEUI
	if len(loraDevEui) != 16 {
		fmt.Printf("entered lora dev eui length is not 16, for '%s' it is == %d\n", loraDevEui, len(loraDevEui))
		return errors.New("invalid lora deveui, len must be 16")
	}

	_, err := m.WriteBlock(11, loraDevEui[0:8])
	if err != nil {
		return err
	}
	_, err = m.WriteBlock(12, loraDevEui[8:])
	if err != nil {
		return err
	}

	// Calculate and write new CRC
	return m.CalculateAndWriteCRC()
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
	GNSSMin                      int64
	GNSSMax                      int64
	DOP                          float64
	OperationalMode              int64
	LoRaEnable                   string
	LoRaRegion                   string
	ABR2                         int
	BLEGain                      string
	MotionMoved                  int
	MotionStationary             int
	MotionAccelActivity          int64
	MotionAccelActivityThreshold int64
	BLEAdvertisingInterval       int64
	BLERefScanInterval           int64
	BLERefRSSI                   int
	BLERefFilter                 string
	BLEAdvertisingType           string
	PressUplink                  string
	PingSlotPeriod               string
	Timeout                      int64
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
		0: "1 s", 1: "2 s", 2: "4 s", 3: "8 s",
		4: "16 s", 5: "32 s", 6: "64 s", 7: "128 s",
	}
	if p, ok := periods[periodInt]; ok {
		return p
	}
	return "Unknown"
}

func parseBLERefMode(flags string) string {
	switch flags {
	case "10":
		return "Reference Tags"
	case "01":
		return "BluFi"
	case "00":
		return "Disabled"
	default:
		return "Unknown"
	}
}

func parseClassSelect(flags string) string {
	switch flags {
	case "00":
		return "Class A"
	case "10":
		return "Class B"
	case "01":
		return "Class C"
	default:
		return "Unknown"
	}
}

func parseConfirmedUplinks(flags string) string {
	switch flags {
	case "0":
		return "Disabled"
	case "1":
		return "Enabled"
	default:
		return "Unknown"
	}
}

func parseHopping(flags string) string {
	switch flags {
	case "0":
		return "Disabled"
	case "1":
		return "Enabled"
	default:
		return "Unknown"
	}
}

func complementToDec(hex string) int {
	i, _ := strconv.ParseInt(hex, 16, 0)
	if i > 127 {
		return int(i - 256)
	}
	return int(i)
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

func printStructuredOutput(m24lr *NfcCard) {
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
		BleMac, _ := m24lr.ReadBleMac()
		LoRaMAC, _ := m24lr.ReadLoraDevEui()
		printField("LoRa MAC", strings.ToUpper(LoRaMAC))
		printField("BLE MAC", strings.ToUpper(BleMac))
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
	log.Infof("\n%s=== %s ===%s\n", colorCyan, title, colorReset)
	content()
}

func printField(label, value string) {
	log.Infof("%s%s:%s %s\n", colorYellow, label, colorReset, value)
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
	log.Infof("%s%s:%s %s\n", colorYellow, label, colorReset, value)
}

// Helper function to reverse CRC bytes
func reverseCRC(crc uint16) string {
	// Convert CRC to bytes
	msb := byte(crc >> 8)   // Most significant byte
	lsb := byte(crc & 0xFF) // Least significant byte

	// Format as hex string with reversed bytes and padded zeros
	return fmt.Sprintf("%02X%02X0000", lsb, msb)
}
