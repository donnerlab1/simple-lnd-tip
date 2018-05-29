package main

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-martini/martini"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

func tip(l *log.Logger, client lnrpc.LightningClient) string {
	ctx := context.Background()
	getInfoResp, err := client.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	pk := ""
	if err != nil {
		fmt.Println("Cannot get info from node:", err)
		pk = err.Error()
	} else {
		pk = getInfoResp.IdentityPubkey
	}
	spew.Dump(getInfoResp)
	ret := "Get invoice at http://donnerlab.com/get_invoice/\n"
	ret += "pubkey: " + pk + "\n"
	ret += "Proudly powered by https://github.com/donnerlab1/simple-lnd-tip"
	return ret
}

func get_invoice(l *log.Logger, client lnrpc.LightningClient) string {
	ctx := context.Background()
	addInvoiceResp, err := client.AddInvoice(ctx, &lnrpc.Invoice{})

	if err != nil {
		fmt.Println("Cannot get tip from node:", err)
		return err.Error()
	}
	spew.Dump(addInvoiceResp)
	return addInvoiceResp.PaymentRequest
}

func pay_invoice(l *log.Logger, client lnrpc.LightningClient, payment_request string) string {
	ctx := context.Background()
	fmt.Println(payment_request)
	sendRequestResp, err := client.SendPaymentSync(ctx, &lnrpc.SendRequest{PaymentRequest: payment_request})

	if err != nil {
		fmt.Println("Cannot send payment from node:", err)
		return err.Error()
	}
	spew.Dump(sendRequestResp)
	return sendRequestResp.String()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: %s <testnet/mainnet> <listen_port> <lnd_port>", os.Args[0])
		return
	}
    network := os.Args[1]
	listen_port := os.Args[2]
	lnd_port := os.Args[3]

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get current user:", err)
		return
	}
	tlsCertPath := path.Join(usr.HomeDir, ".lnd/tls.cert")
	macaroonPath := path.Join(usr.HomeDir, ".lnd/admin.macaroon")

	tlsCreds, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
	if err != nil {
		fmt.Println("Cannot get node tls credentials", err)
		return
	}

	macaroonBytes, err := ioutil.ReadFile(macaroonPath)
	if err != nil {
		fmt.Println("Cannot read macaroon file", err)
		return
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macaroonBytes); err != nil {
		fmt.Println("Cannot unmarshal macaroon", err)
		return
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(mac)),
	}

	fmt.Print("Trying to connect to lnd...")
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", lnd_port), opts...)
	if err != nil {
		fmt.Println("cannot dial to lnd", err)
		return
	}
	client := lnrpc.NewLightningClient(conn)
	fmt.Println("ok")

	m := martini.Classic()
	m.Get("/tip", func(log *log.Logger) string {
		return tip(log, client)
	})
	m.Get("/get_invoice", func(log *log.Logger) string {
		return get_invoice(log, client)
	})
    if network == "testnet" {
        m.Get("/pay_invoice/**", func(log *log.Logger, params martini.Params) string {
            return pay_invoice(log, client, params["_1"])
        })
    }
	m.RunOnAddr(fmt.Sprintf(":%s", listen_port))

}
