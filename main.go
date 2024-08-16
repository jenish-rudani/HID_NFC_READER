package hidNfc

import (
	"flag"
	"fmt"
	_ "fmt"
	pcsc "github.com/dumacp/smartcard/pcsc"
	"log"
)

func main() {
	fmt.Println("Start Logs")
	flag.Parse()
	ctx, err := pcsc.NewContext()
	if err != nil {
		fmt.Println("error new context: ", err)
	}
	defer ctx.Release()

	readers, err := pcsc.ListReaders(ctx)
	for i, el := range readers {
		fmt.Printf("reader %v: %s\n", i, el)
	}
	piccReaders := make([]pcsc.Reader, 0)
	for _, el := range readers {
		log.Printf("reader: %s\n", el)

		piccReaders = append(piccReaders, pcsc.NewReader(ctx, el))

	}

	for _, piccReader := range piccReaders {
		log.Printf("picc reader: %s\n", piccReader)
		picc, err := piccReader.ConnectCardPCSC()
		if err != nil {
			log.Printf("%s\n", err)
		}

		resp1, err := picc.UID()
		log.Printf("picc UID: % X\n", resp1)
		if err != nil {
			log.Fatalf("%s\n", err)
		}

		resp2, err := picc.ATR()
		log.Printf("picc ATR: % X\n", resp2)
		if err != nil {
			log.Fatalf("%s\n", err)
		}

		resp3, err := picc.ATS()
		log.Printf("picc ATS: % X\n", resp3)
		if err != nil {
			log.Fatalf("%s\n", err)
		}
		picc.DisconnectCard()
	}
}
