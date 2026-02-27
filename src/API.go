package main

import (
	"context"
	"os"
	"time"
	"crypto/tls"
	"crypto/x509"
	"log"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client

func main() {
	host   := "127.0.0.1"
	uri    := "mongodb://" + host + ":27017/?" +
			"directConnection=true&serverSelectionTimeoutMS=2000&authSource=admin&tls=true&appName=xcash-api-read"

	caPath   := "/etc/ssl/mongodb/mongodb.crt"
	pemPath  := "/etc/ssl/mongodb/mongodb.pem"

	caPEM, err := os.ReadFile(caPath)
	if err != nil { log.Fatalf("read CA: %v", err) }
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) { log.Fatal("bad CA PEM") }

	clientCert, err := tls.LoadX509KeyPair(pemPath, pemPath)
	if err != nil { log.Fatalf("load client keypair: %v", err) }

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCert}, 
		ServerName:   host,
	}

	clientOpts := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsCfg).
		SetAuth(options.Credential{
			AuthSource:    "XCASH_PROOF_OF_STAKE",
			Username:      os.Getenv("MONGODB_READ_USERNAME"),
			Password:      os.Getenv("MONGODB_READ_PASSWORD"),
			AuthMechanism: "SCRAM-SHA-256",
		})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	log.Println("MongoDB connected (direct TLS read-only)")
	mongoClient = client

	defer func() {
		ctxDisc, cancelDisc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDisc()
		if err := client.Disconnect(ctxDisc); err != nil {
			log.Printf("disconnect error: %v", err)
		}
	}()

	app := fiber.New(fiber.Config{
		Prefork:               false,
		DisableStartupMessage: true,
	})

	// setup blockchain routes
	app.Get("/v2/xcash/blockchain/unauthorized/blocks/", v2_xcash_blockchain_unauthorized_blocks_blockHeight)
	app.Get("/v2/xcash/blockchain/unauthorized/blocks/:blockHeight", v2_xcash_blockchain_unauthorized_blocks_blockHeight)
	app.Get("/v2/xcash/blockchain/unauthorized/height/", v2_xcash_blockchain_unauthorized_height)

	// setup xcash dpops routes

	fmt.Println("URL:", c.OriginalURL())
	fmt.Println("Params:", c.AllParams())

	app.Get("/v2/xcash/dpops/unauthorized/delegates/registered/", v2_xcash_dpops_unauthorized_delegates_registered)
	app.Get("/v2/xcash/dpops/unauthorized/delegates/:delegateName", v2_xcash_dpops_unauthorized_delegates)
	app.Get("/v2/xcash/dpops/unauthorized/delegates/votes/:delegateName", v2_xcash_dpops_unauthorized_delegates_votes)
	app.Get("/v2/xcash/dpops/unauthorized/delegates/voters/:delegateName", v2_xcash_dpops_unauthorized_delegate_voters)
	app.Get("/v2/xcash/dpops/unauthorized/votes/:address", v2_xcash_dpops_unauthorized_votes)
	app.Get("/v2/xcash/dpops/unauthorized/rounds/:blockHeight", v2_xcash_dpops_unauthorized_rounds)

	// setup xpayment twitter routes
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/stats/", v1_xpayment_twitter_unauthorized_stats)
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/statsPerDay/:start/:limit", v1_xpayment_twitter_unauthorized_statsperday)
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/topStats/:amount", v1_xpayment_twitter_unauthorized_topstats)
	app.Post("/v1/xpayment-twitter/twitter/unauthorized/recentTips/:amount", v1_xpayment_twitter_unauthorized_recent_tips)

	// setup global routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendString("Invalid API Request")
	})

	// start the server
	if err := app.Listen(":9000"); err != nil {
    	log.Fatalf("fiber listen: %v", err)
	}
}