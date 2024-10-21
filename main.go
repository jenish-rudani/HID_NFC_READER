package main

import (
	"bitbucket.org/bluvision/pcsc/pcsc"
	"fmt"
	"github.com/jenish-rudani/HID_NFC_READER/internal/utils/log"
	"strings"
)

// APDUInfo holds the parsed information from an APDU response
type APDUInfo struct {
	Tag         byte
	Value       string
	ValueRaw    []byte
	StatusWords [2]byte
}

func parseAPDU(apdu []uint8) (APDUInfo, error) {
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

func main() {
	log.Info("Starting HID NFC Reader")
	test()
	TestNfc()
	fmt.Println("NFC test completed successfully")
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

func test() {
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
		sam, err := samReader.ConnectDirect()
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

		productInfo, err := parseAPDU(apdu)
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
		serialInfo, err := parseAPDU(apdu)
		if err != nil {
			fmt.Printf("Error parsing serial number: %v\n", err)
		} else {
			fmt.Printf("Serial Number: %s\n", serialInfo.Value)
		}

	}

}

func TestNfc() {
	// Initialize PCSC
	ctx, err := pcsc.NewContext()
	if err != nil {
		fmt.Printf("Failed to create PCSC context: %v\n", err)
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
	//var cardReaders []*nfc.CardReader
	//create reader list
	//for _, reader := range rdrlst {
	//	reader := pcsc.NewReader(ctx, reader)
	//	//log.Infof("NFC Reader: %# v", pretty.Formatter(reader))
	//	//// Create CardReader instance
	//	//cardReader, err := nfc.NewCardReader(reader)
	//	//if err != nil {
	//	//	fmt.Printf("Failed to create CardReader: %v\n", err)
	//	//	return
	//	//}
	//	//cardReaders = append(cardReaders, cardReader)
	//}
	//log.Infof("All card readers: %# v", pretty.Formatter(cardReaders))
	//TODO: need to handle defer

	//joinEui, err := m24lr.ReadLoraJoinEui()
	//if err != nil {
	//	log.Errorf("Failed to read LoRa join EUI: %v\n", err)
	//	return
	//}
	//log.Infof("LORA JOIN EUI: 0x%s\n", joinEui)
	//
	//loraJoinKey, err := m24lr.ReadLoraJoinKey()
	//if err != nil {
	//	log.Errorf("Failed to read LoRa Join key: %v\n", err)
	//	return
	//}
	//log.Infof("LORA JOIN KEY (128Bits): 0x%s\n", loraJoinKey)
	////err = m24lr.ReadBLELocalName()
	////if err != nil {
	////	fmt.Printf("Failed to read BLE local name: %v\n", err)
	////	return
	////}
	////
	////err = m24lr.WriteBLELocalName("Asset++")
	////if err != nil {
	////	fmt.Printf("Failed to write BLE local name: %v\n", err)
	////	return
	////}
	////
	////err = m24lr.ReadBLELocalName()
	////if err != nil {
	////	fmt.Printf("Failed to read BLE local name: %v\n", err)
	////	return
	////}
	//
	//devEui, err := m24lr.ReadLoraDevEui()
	//if err != nil {
	//	log.Errorf("Failed to read BLE local name: %v\n", err)
	//	return
	//}
	//log.Infof("DevEUI (Hex): %s\n", devEui)
	//
	//err = m24lr.WriteLoraDevEui("0000000000000000")
	//if err != nil {
	//	log.Errorf("Failed to write DevEUI: %v\n", err)
	//	return
	//}
	//
	//devEui, err = m24lr.ReadLoraDevEui()
	//if err != nil {
	//	log.Errorf("Failed to read BLE local name: %v\n", err)
	//	return
	//}
	//log.Infof("DevEUI (Hex): %s\n", devEui)
}
