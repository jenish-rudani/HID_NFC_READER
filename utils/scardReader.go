package main

import (
	"fmt"
	"github.com/ebfe/scard"
)

func main() {
	// Establish a PC/SC context
	context, err := scard.EstablishContext()
	if err != nil {
		fmt.Println("Error EstablishContext:", err)
		return
	}

	// Release the PC/SC context (when needed)
	defer context.Release()

	// List available readers
	readers, err := context.ListReaders()
	if err != nil {
		fmt.Println("Error ListReaders:", err)
		return
	}

	// Use the first reader
	reader := readers[0]
	fmt.Println("Using reader:", reader)

	// Connect to the card
	card, err := context.Connect(reader, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		fmt.Println("Error Connect:", err)
		return
	}

	// Disconnect (when needed)
	defer card.Disconnect(scard.LeaveCard)

	// Send select APDU
	var cmd_select = []byte{0x00, 0xa4, 0x04, 0x00, 0x0A, 0xA0,
		0x00, 0x00, 0x00, 0x62, 0x03, 0x01, 0x0C, 0x06, 0x01}
	rsp, err := card.Transmit(cmd_select)
	if err != nil {
		fmt.Println("Error Transmit:", err)
		return
	}
	fmt.Println(rsp)

	// Send command APDU
	var cmd_command = []byte{0x00, 0x00, 0x00, 0x00}
	rsp, err = card.Transmit(cmd_command)
	if err != nil {
		fmt.Println("Error Transmit:", err)
		return
	}
	fmt.Println(rsp)
	for i := 0; i < len(rsp)-2; i++ {
		fmt.Printf("%c", rsp[i])
	}
	fmt.Println()
}
