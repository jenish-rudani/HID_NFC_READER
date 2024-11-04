package main

import (
	"bitbucket.org/bluvision/pcsc/pcsc"
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/jenish-rudani/HID_NFC_READER/internal/nfc"
	"github.com/jenish-rudani/HID_NFC_READER/internal/utils/log"
	"os"
	"strconv"
	"strings"
)

var command string
var params string
var versionFlag bool

func initCommandLine() {
	flag.StringVar(&command, "cmd", "SerialNumberTest", "SerialNumberTest")
	flag.StringVar(&params, "param", "", "params")
	flag.BoolVar(&versionFlag, "version", false, "Print version information")
	flag.Parse()
}

func writeLoraInfoToCSV(filename string, info *nfc.LoraInfo, isNewFile bool) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if new file
	if isNewFile {
		header := []string{"Timestamp", "DevEUI", "JoinEUI", "JoinKey", "CRC Status"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %v", err)
		}
	}

	// Write data
	record := []string{
		info.Timestamp,
		info.DevEUI,
		info.JoinEUI,
		info.JoinKey,
		info.CRCStatus,
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write CSV record: %v", err)
	}

	return nil
}

func nfcRunCommands(command string, nfcCardInstance *nfc.NfcCard) error {
	var err error
	switch command {

	case "readloraloop":
		filename := "lora_info.csv"
		if params != "" {
			filename = params
		}

		// Check if file exists
		isNewFile := true
		if _, err := os.Stat(filename); err == nil {
			isNewFile = false
		}

		reader := bufio.NewReader(os.Stdin)
		tagCount := 0

		log.Info("Starting LoRa reading loop...")
		log.Infof("Results will be saved to: %s", filename)
		log.Info("Press 'x' to exit or any other key to read next tag...")

		for {
			fmt.Print("\nPress <Enter> to read next tag (or 'x' + <Enter> to exit): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input == "x" {
				log.Infof("Loop ended. Total tags read: %d", tagCount)
				break
			}

			log.Info("Reading tag...")
			info, err := nfcCardInstance.ReadLoraInfo()
			if err != nil {
				log.Errorf("Failed to read LoRa info: %v\n", err)
				continue
			}

			// Print info to console
			log.Info("Tag Read Successfully:")
			log.Infof("DevEUI: %s", info.DevEUI)
			log.Infof("JoinEUI: %s", info.JoinEUI)
			log.Infof("JoinKey: %s", info.JoinKey)
			log.Infof("CRC Status: %s", info.CRCStatus)

			// Write to CSV
			err = writeLoraInfoToCSV(filename, info, isNewFile)
			if err != nil {
				log.Errorf("Failed to write to CSV: %v\n", err)
				continue
			}

			tagCount++
			isNewFile = false
			log.Infof("Tag information saved to %s (Total tags: %d)", filename, tagCount)
		}

	case "erase":
		log.Warn("WARNING: This will erase all data from the NFC tag!")
		if params != "confirm" {
			log.Error("To erase the tag, use: -cmd erase -param confirm")
			break
		}
		err := nfcCardInstance.EraseTag()
		if err != nil {
			log.Errorf("Failed to erase tag: %v\n", err)
			break
		}
		log.Info("Tag erased successfully")

		// Validate the erasure
		err = nfcCardInstance.ValidateCRC()
		if err != nil {
			log.Errorf("Post-erase CRC validation failed: %v\n", err)
			break
		}
		log.Info("Post-erase CRC validation successful")

	case "SerialNumberTest":
		SerialNumberTest()
	case "validateCrc":
		err = nfcCardInstance.ValidateCRC()
		if err != nil {
			log.Errorf("Failed to validate CRC: %v\n", err)
			break
		}
	case "readblelocal":
		name, err := nfcCardInstance.ReadBLELocalName()
		if err != nil {
			log.Errorf("Failed to read BLE local name: %v\n", err)
			break
		}
		log.Infof("BLE Local Name: %s\n", name)
	case "writeblelocal":
		if params == "" {
			log.Errorf("Missing params (local name)\n")
			break
		}
		err = nfcCardInstance.WriteBLELocalName(params)
		if err != nil {
			log.Errorf("Failed to write BLE local name: %v\n", err)
			break
		}
		log.Info("BLE local name written successfully")
	case "readlorajoineui":
		joinEui, err := nfcCardInstance.ReadLoraJoinEui()
		if err != nil {
			log.Errorf("Failed to read LoRa JoinEUI: %v\n", err)
			break
		}
		log.Infof("LoRa JoinEUI: %s\n", strings.ToUpper(joinEui))
	case "writelorajoineui":
		if params == "" {
			log.Errorf("Missing params (JoinEUI)\n")
			break
		}
		err = nfcCardInstance.WriteLoraJoinEui(params)
		if err != nil {
			log.Errorf("Failed to write LoRa JoinEUI: %v\n", err)
			break
		}
		log.Info("LoRa JoinEUI written successfully")
	case "readlorajoinkey":
		joinKey, err := nfcCardInstance.ReadLoraJoinKey()
		if err != nil {
			log.Errorf("Failed to read LoRa Join Key: %v\n", err)
			break
		}
		log.Infof("LoRa Join Key: %s\n", strings.ToUpper(joinKey))
	case "writelorajoinkey":
		if params == "" {
			log.Errorf("Missing params (Join Key)\n")
			break
		}
		joinKey, err := nfcCardInstance.ReadLoraJoinKey()
		if err != nil {
			log.Errorf("Failed to read LoRa Join Key: %v\n", err)
			break
		}
		log.Infof("Previous LoRa Join Key: %s\n", strings.ToUpper(joinKey))

		err = nfcCardInstance.WriteLoraJoinKey(params)
		if err != nil {
			log.Errorf("Failed to write LoRa Join Key: %v\n", err)
			break
		}

		joinKey, err = nfcCardInstance.ReadLoraJoinKey()
		if err != nil {
			log.Errorf("Failed to read LoRa Join Key: %v\n", err)
			break
		}
		log.Infof("Current LoRa Join Key: %s\n", strings.ToUpper(joinKey))
		log.Info("LoRa Join Key written successfully")
	case "readloradeveui":
		devEui, err := nfcCardInstance.ReadLoraDevEui()
		if err != nil {
			log.Errorf("Failed to read LoRa DevEUI: %v\n", err)
			break
		}
		log.Infof("LoRa DevEUI: %s (%s)\n", strings.ToUpper(devEui), strings.ToUpper(strings.ReplaceAll(devEui, ":", "")))

	case "writeloradeveui":
		if params == "" {
			log.Errorf("Missing params (DevEUI)\n")
			break
		}
		err = nfcCardInstance.WriteLoraDevEui(params)
		if err != nil {
			log.Errorf("Failed to write LoRa DevEUI: %v\n", err)
			break
		}
		log.Info("LoRa DevEUI written successfully")
	case "readlora":
		log.Info("Reading all LoRa-related information...")

		// Read DevEUI
		devEui, err := nfcCardInstance.ReadLoraDevEui()
		if err != nil {
			log.Errorf("Failed to read LoRa DevEUI: %v\n", err)
			break
		}
		log.Infof("LoRa DevEUI: %s (%s)\n", strings.ToUpper(devEui), strings.ToUpper(strings.ReplaceAll(devEui, ":", "")))

		// Read Join EUI
		joinEui, err := nfcCardInstance.ReadLoraJoinEui()
		if err != nil {
			log.Errorf("Failed to read LoRa JoinEUI: %v\n", err)
			break
		}
		log.Infof("LoRa JoinEUI: %s\n", strings.ToUpper(joinEui))

		// Read Join Key
		joinKey, err := nfcCardInstance.ReadLoraJoinKey()
		if err != nil {
			log.Errorf("Failed to read LoRa Join Key: %v\n", err)
			break
		}
		log.Infof("LoRa Join Key: %s\n", strings.ToUpper(joinKey))

		// Print validation results
		if (strings.Compare(joinEui, "0000000000000000") == 0) ||
			(strings.Compare(joinEui, "FFFFFFFFFFFFFFFF") == 0) {
			log.Warn("JoinEUI has default value - needs to be programmed")
		}

		if (strings.Compare(joinKey, "00000000000000000000000000000000") == 0) ||
			(strings.Compare(joinKey, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") == 0) {
			log.Warn("Join Key has default value - needs to be programmed")
		}

		log.Info("Completed reading LoRa information")
	case "sleep":
		if params == "" {
			log.Errorf("Missing params\n")
			break
		}
		//Extract Params
		sleepState, err := strconv.ParseBool(params)
		if err != nil {
			log.Errorf("Failed to parse params: %v\n", err)
			break
		}
		err = nfcCardInstance.WriteTagSleepBit(sleepState)
		if err != nil {
			log.Errorf("Failed to set sleep state: %v\n", err)
		}
	case "tagpostbit":
		if params == "" {
			log.Errorf("Missing params\n")
			break
		}
		//Extract Params
		bitValue, err := strconv.ParseBool(params)
		if err != nil {
			log.Errorf("Failed to parse params: %v\n", err)
			break
		}
		err = nfcCardInstance.WriteTagPostBit(bitValue)
		if err != nil {
			log.Errorf("Failed to write tag post bit: %v\n", err)
			break
		}
		log.Info("Tag post bit written successfully")

	case "readmacs":
		loraMac, err := nfcCardInstance.ReadLoraDevEui()
		if err != nil {
			log.Errorf("Failed to read LoRa MAC: %v\n", err)
			break
		}
		bleMac, err := nfcCardInstance.ReadBleMac()
		if err != nil {
			log.Errorf("Failed to read MACs: %v\n", err)
			break
		}
		fmt.Printf("Lora MAC-> %s\n", strings.ToUpper(loraMac))
		fmt.Printf("BLE MAC-> 01:%s\n", strings.ToUpper(bleMac))
	case "cfgr":
		settings, err := nfcCardInstance.ReadDittoSettings()
		if err != nil {
			log.Errorf("Failed to read settings: %v\n", err)
			break
		}
		nfc.PrintMappedDittoSettings(settings)
	}

	return err
}

func initNfc(ctx *pcsc.Context, readerName string) (*nfc.NfcCard, error) {
	reader := pcsc.NewReader(ctx, readerName)
	cardInstance, err := nfc.NewCardReader(reader)
	if err != nil {
		log.Errorf("Failed to create nfcCard: %v\n", err)
		return nil, err
	}
	return cardInstance, nil
}

// convertToASCII converts a byte slice to an ASCII string and returns both ASCII and hex representations
func convertToASCII(data []byte) string {
	var result strings.Builder
	for _, b := range data {
		if b >= 32 && b <= 126 { // Printable ASCII range
			result.WriteByte(b)
		} else if b == 0 {
			break // Stop at null terminator
		} else {
			result.WriteString(fmt.Sprintf("\\x%02x", b))
		}
	}
	return result.String()
}

func SerialNumberTest() {
	getFwVersion := []byte{0xFF, 0x70, 0x07, 0x6B, 0x08, 0xA2, 0x06, 0xA0, 0x04, 0xA0, 0x02, 0x92, 0x00, 0x00}
	getProductName := []byte{0xFF, 0x70, 0x07, 0x6B, 0x08, 0xA2, 0x06, 0xA0, 0x04, 0xA0, 0x02, 0x82, 0x00, 0x00}
	ctx, err := pcsc.NewContext()
	if err != nil {
		log.Fatal("Not connection")
	}
	defer ctx.Release()
	readers, err := pcsc.ListReaders(ctx)
	for i, el := range readers {
		log.Infof("reader %v: %s\n", i, el)
	}
	samReaders := make([]pcsc.Reader, 0)
	for _, el := range readers {
		samReaders = append(samReaders, pcsc.NewReader(ctx, el))
	}
	for _, samReader := range samReaders {
		sam, err := samReader.ConnectDirectShared()
		if err != nil {
			log.Errorf("%s\n", err)
			continue
		}

		log.Infof("sam: %v", sam)
		apdu, err := sam.ControlApdu(0x42000000+2079, getProductName)
		if err != nil {
			log.Errorf("error transmitting read firmware control apdu")
			return
		}

		productInfo, err := nfc.ParseAPDU(apdu)
		if err != nil {
			log.Errorf("error parsing parse apudu")
		} else {
			fmt.Printf("Product Name: %s\n", convertToASCII([]byte(productInfo.ValueRaw)))
		}

		apdu, err = sam.ControlApdu(0x42000000+2079, getFwVersion)
		if err != nil {
			log.Errorf("error transmitting read firmware control apdu")
			return
		}
		serialInfo, err := nfc.ParseAPDU(apdu)
		if err != nil {
			fmt.Printf("Error parsing serial number: %v\n", err)
		} else {
			fmt.Printf("Serial Number: %s\n", serialInfo.Value)
		}
		err = sam.DisconnectUnpowerCard()
		if err != nil {
			log.Errorf("error %v", err)

		}
	}

}

func main() {
	initCommandLine()

	// Handle version flag
	if versionFlag {
		fmt.Printf("HID NFC Reader v%s\n", VERSION)
		fmt.Printf("Git commit: %s\n", GITCOMMIT)
		fmt.Printf("Built at: %s\n", BUILDTIME)
		return
	}

	// If parameters are provided, format them based on the command
	var formattedParam string
	if params != "" {
		switch params {
		case "writeloradeveui", "writelorajoineui":
			// For 8-byte keys (16 hex chars)
			formattedParam = formatKey(params)
			if len(strings.ReplaceAll(formattedParam, ":", "")) != 16 {
				log.Fatalf("Invalid key length for DevEUI/JoinEUI. Expected 16 hex characters, got %d",
					len(strings.ReplaceAll(formattedParam, ":", "")))
			}
		case "writelorajoinkey":
			// For 16-byte keys (32 hex chars)
			formattedParam = formatKey(params)
			if len(strings.ReplaceAll(formattedParam, ":", "")) != 32 {
				log.Fatalf("Invalid key length for Join Key. Expected 32 hex characters, got %d",
					len(strings.ReplaceAll(formattedParam, ":", "")))
			}
		default:
			formattedParam = params
		}
	}
	params = formattedParam

	// Initialize PCSC
	ctx, err := pcsc.NewContext()
	if err != nil {
		log.Infof("Failed to create PCSC context: %v\n", err)
		return
	}
	defer ctx.Release()

	// List readers
	rdrlst, err := pcsc.ListReaders(ctx)
	if err != nil {
		fmt.Printf("Failed to list readers: %v\n", err)
		return
	}

	if len(rdrlst) == 0 {
		fmt.Println("No readers found")
		return
	}

	nfcCardReader, err := initNfc(ctx, rdrlst[0])
	if err != nil {
		log.Errorf("Failed to initialize NFC card reader: %v\n", err)
		return
	}
	defer nfcCardReader.Close()

	commands := strings.Split(command, ",")
	for _, cmd := range commands {
		log.Infof("############## Running command: [%s] ############## ", cmd)
		err := nfcRunCommands(cmd, nfcCardReader)
		if err != nil {
			return
		}
	}
	err = nfcCardReader.Reader.DisconnectUnpowerCard()
	if err != nil {
		log.Errorf("Failed to disconnect card: %v\n", err)
		return
	}
	log.Info("SUCCESS")
}

func formatKey(key string) string {
	// Remove any existing colons or spaces
	key = strings.ReplaceAll(key, ":", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ToUpper(key)
	return key
}
