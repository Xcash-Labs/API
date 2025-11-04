package main

import (
	"context"
	"os"
	"time"
	"crypto/tls"
	"crypto/x509"
	"log"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client

func main() {
	user := os.Getenv("MONGODB_READ_USERNAME")
	pass := os.Getenv("MONGODB_READ_PASSWORD")
	if user == "" || pass == "" {
		log.Fatal("Missing MongoDB_READ_USERNAME or MongoDB_READ_PASSWORD")
	}

	const (
		host    = "seeds.xcashseeds.us"                     // primary read node
		caPath  = "/etc/ssl/mongodb/mongodb.crt"            // shared self-signed cert
		appName = "xcash-api-read"
	)

	// --- TLS config using the same cert you already deploy on all nodes ---
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		log.Fatalf("read CA: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		log.Fatal("bad CA PEM")
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
		ServerName: host, // must match DNS SAN in your cert
	}

	// --- Direct connection to a single node (read-only) ---
	uri := "mongodb://" + host + ":27017/?" +
		"tls=true&directConnection=true&authSource=admin&appName=" + appName

	clientOpts := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsCfg).
		SetAuth(options.Credential{
			AuthSource: "admin",
			Username:   user,
			Password:   pass,
			AuthMechanism:  "SCRAM-SHA-256",
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
	app.Get("/v1/xcash/blockchain/unauthorized/stats/", v1_xcash_blockchain_unauthorized_stats)
	app.Get("/v1/xcash/blockchain/unauthorized/blocks/", v1_xcash_blockchain_unauthorized_blocks_blockHeight)
	app.Get("/v1/xcash/blockchain/unauthorized/blocks/:blockHeight/", v1_xcash_blockchain_unauthorized_blocks_blockHeight)
	app.Get("/v1/xcash/blockchain/unauthorized/tx/:txHash/", v1_xcash_blockchain_unauthorized_tx_txHash)
	app.Post("/v1/xcash/blockchain/unauthorized/tx/prove/", v1_xcash_blockchain_unauthorized_tx_prove)
	app.Post("/v1/xcash/blockchain/unauthorized/address/prove/", v1_xcash_blockchain_unauthorized_address_prove)
	app.Get("/v1/xcash/blockchain/unauthorized/address/history/:type/:address/", v1_xcash_blockchain_unauthorized_address_history)
	app.Get("/v1/xcash/blockchain/unauthorized/address/validate/:address/", v1_xcash_blockchain_unauthorized_address_validate)
	app.Post("/v1/xcash/blockchain/unauthorized/address/createIntegrated/", v1_xcash_blockchain_unauthorized_address_create_integrated)

	// setup xcash dpops routes
	app.Get("/v2/xcash/dpops/unauthorized/stats/", v2_xcash_dpops_unauthorized_stats)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/registered/", v1_xcash_dpops_unauthorized_delegates_registered)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/online/", v1_xcash_dpops_unauthorized_delegates_online)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/active/", v1_xcash_dpops_unauthorized_delegates_active)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/:delegateName/", v1_xcash_dpops_unauthorized_delegates)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/rounds/:delegateName", v1_xcash_dpops_unauthorized_delegates_rounds)
	app.Get("/v1/xcash/dpops/unauthorized/delegates/votes/:delegateName/:start/:limit", v1_xcash_dpops_unauthorized_delegates_votes)
	app.Get("/v1/xcash/dpops/unauthorized/votes/:address", v1_xcash_dpops_unauthorized_votes)
	app.Get("/v1/xcash/dpops/unauthorized/rounds/:blockHeight", v1_xcash_dpops_unauthorized_rounds)
	app.Get("/v1/xcash/dpops/unauthorized/lastBlockProducer", v1_xcash_dpops_unauthorized_last_block_producer)

	// setup xpayment twitter routes
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/stats/", v1_xpayment_twitter_unauthorized_stats)
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/statsPerDay/:start/:limit", v1_xpayment_twitter_unauthorized_statsperday)
	app.Get("/v1/xpayment-twitter/twitter/unauthorized/topStats/:amount", v1_xpayment_twitter_unauthorized_topstats)
	app.Post("/v1/xpayment-twitter/twitter/unauthorized/recentTips/:amount", v1_xpayment_twitter_unauthorized_recent_tips)

	// setup global routes
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendString("Invalid API Request")
	})

	// start the timers
	go timers()
	go timers_build_data()

	// start the server
	if err := app.Listen(":9000"); err != nil {
    	log.Fatalf("fiber listen: %v", err)
	}
}