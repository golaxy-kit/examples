package main

import (
	"context"
	"fmt"
	"kit.golaxy.org/plugins/gate/gtp_client"
	"kit.golaxy.org/plugins/transport"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		panic("missing endpoint")
	}

	cli, err := gtp_client.Connect(context.Background(), os.Args[1],
		gtp_client.Option{}.RecvDataHandlers(func(client *gtp_client.Client, data []byte) error {
			fmt.Println(string(data))
			return nil
		}),
		gtp_client.Option{}.EncCipherSuite(transport.CipherSuite{
			SecretKeyExchange:   transport.SecretKeyExchange_ECDHE,
			SymmetricEncryption: transport.SymmetricEncryption_XChaCha20,
			BlockCipherMode:     transport.BlockCipherMode_None,
			PaddingMode:         transport.PaddingMode_None,
			MACHash:             transport.Hash_Fnv1a64,
		}),
		gtp_client.Option{}.CompressedSize(128),
		gtp_client.Option{}.IOTimeout(10*time.Minute),
		gtp_client.Option{}.IORetryTimes(1000),
	)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	fmt.Println("this console is", cli.GetSessionId())

	for {
		var text string
		fmt.Scanln(&text)
		if err := cli.SendData([]byte(text)); err != nil {
			if err := gtp_client.Reonnect(cli); err != nil {
				fmt.Println("reconnect err:", err)
			}
		}
	}
}