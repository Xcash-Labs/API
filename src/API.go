package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	// setup the mongodb connection

	fmt.Println("Hello, world!")

	user := os.Getenv("MONGODB_READ_USERNAME")
	pass := os.Getenv("MONGODB_READ_PASSWORD")
	const caPath   = "/etc/ssl/mongodb/mongodb.crt" // CA cert to verify server
	const certPath = "/etc/ssl/mongodb/mongodb.pem" // client cert (PEM, contains cert)
	const keyPath  = "/etc/ssl/mongodb/mongodb.pem" // client key (PEM, same file if combined)
	const mongoTLSHost = "seeds.xcashseeds.us"

	if user == "" || pass == "" {
		log.Fatal("Mongo env missing: set MONGODB_READ_USERNAME and MONGODB_READ_PASSWORD")
	}

	// --- Build tls.Config with CA + client cert ---
	caPEM, err := os.ReadFile(caPath)
	if err != nil { log.Fatalf("read CA: %v", err) }
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		log.Fatal("bad CA PEM")
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil { log.Fatalf("load client keypair: %v", err) }

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		ServerName:   mongoTLSHost,
	}

	uri := "mongodb://localhost:27017/?" + "directConnection=true&tls=true&retryReads=true&appName=xcash-api"

	clientOpts := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsCfg)

	if user != "" && pass != "" {
		clientOpts.SetAuth(options.Credential{
			AuthSource: "admin",
			Username:   user,
			Password:   pass,
			Mechanism: "SCRAM-SHA-256",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cl, err := mongo.Connect(ctx, clientOpts)
	if err != nil { log.Fatalf("mongo connect: %v", err) }
	if err := cl.Ping(ctx, nil); err != nil { log.Fatalf("mongo ping: %v", err) }

	log.Println("MongoDB connected (hard-coded TLS, direct/local)")

	if mongoClienterror != nil {
		os.Exit(0)
	}

	defer func() {
		if mongoClienterror = mongoClient.Disconnect(context.TODO()); mongoClienterror != nil {
			os.Exit(0)
		}
	}()

	// setup fiber
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
	app.Listen(":9000")
}
