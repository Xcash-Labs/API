package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func blockchain_size() (int64, error) {
	// Variables
	var size int64

	err := filepath.Walk(BLOCKCHAIN_DIRECTORY, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func RandStringBytes(n int) string {
	// Constants
	const letterBytes = "0123456789abcdef"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func get_current_block_height() int {
	// Variables
	var data_read CurrentBlockHeight
	var data_send string
	var error error

	data_send, error = send_http_data("http://127.0.0.1:18281/json_rpc", `{"jsonrpc":"2.0","id":"0","method":"get_block_count"}`)
	if !strings.Contains(data_send, "\"result\"") || error != nil {
		return 0
	}
	if err := json.Unmarshal([]byte(data_send), &data_read); err != nil {
		return 0
	}
	return data_read.Result.Count
}

func get_block_delegate(requestBlockHeight int) string {
	// Variables
	var database_data XcashDpopsReserveBytesCollection

	// get the collection
	block_height_data := strconv.Itoa(int(((requestBlockHeight - XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT) / BLOCKS_PER_DAY_FIVE_MINUTE_BLOCK_TIME)) + 1)
	collection_number := "reserve_bytes_" + block_height_data
	collection := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection(collection_number)

	// get the reserve bytes
	filter := bson.D{{"block_height", strconv.Itoa(requestBlockHeight)}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, filter).Decode(&database_data)
	if err == mongo.ErrNoDocuments {
		return ""
	} else if err != nil {
		return ""
	}

	// get the delegate name from the reserve bytes
	delegate_name := database_data.ReserveBytes[strings.Index(database_data.ReserveBytes, BLOCKCHAIN_RESERVED_BYTES_START)+len(BLOCKCHAIN_RESERVED_BYTES_START) : strings.Index(database_data.ReserveBytes, BLOCKCHAIN_DATA_SEGMENT_STRING)]
	delegate_name_data, err := hex.DecodeString(delegate_name)
	if err != nil {
		return ""
	}

	return string(delegate_name_data)
}

func v2_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {
	// Short-circuit if DB is down; still okay to serve pure RPC data, but we need DB for DPOPS bits
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	var (
		output                v2XcashBlockchainUnauthorizedBlocksBlockHeight
		dataSend              string
		err                   error
		reqHeightStr          = strings.TrimSpace(c.Params("blockHeight"))
		info                  BlockchainStats
		block                 BlockchainBlock
		blockJSON             BlockchainBlockJson
	)

	// 1) Determine height
	if reqHeightStr == "" {
		dataSend, err = send_http_data("http://127.0.0.1:18281/json_rpc", `{"jsonrpc":"2.0","id":"0","method":"get_info"}`)
		if err != nil || !strings.Contains(dataSend, `"result"`) {
			return c.JSON(ErrorResults{"Could not get the block data"})
		}
		if jsonErr := json.Unmarshal([]byte(dataSend), &info); jsonErr != nil {
			return c.JSON(ErrorResults{"Could not get the block data"})
		}
		// latest-1 (match old behavior)
		reqHeightStr = strconv.FormatInt(int64(info.Result.Height-1), 10)
	}

	// 2) Fetch block by height
	dataSend, err = send_http_data("http://127.0.0.1:18281/json_rpc",
		`{"jsonrpc":"2.0","id":"0","method":"get_block","params":{"height":`+reqHeightStr+`}}`)
	if err != nil || !strings.Contains(dataSend, `"result"`) {
		return c.JSON(ErrorResults{"Could not get the block data"})
	}
	if jsonErr := json.Unmarshal([]byte(dataSend), &block); jsonErr != nil {
		return c.JSON(ErrorResults{"Could not get the block data"})
	}

	// 3) Parse block JSON (tx list)
	s := string(block.Result.JSON)
	s = strings.ReplaceAll(s, `\n`, "")
	s = strings.ReplaceAll(s, `\`, "")
	if jsonErr := json.Unmarshal([]byte(s), &blockJSON); jsonErr != nil {
		return c.JSON(ErrorResults{"Could not get the block data"})
	}

	// ---- New DPOPS lookup (consensus_rounds) ----
	// If a round doc exists for this block_height, it's a DPOPS block.
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	psDB := mongoClient.Database(XCASH_DPOPS_DATABASE)
	colRounds := psDB.Collection("consensus_rounds")

	bh := int64(block.Result.BlockHeader.Height)
	var roundDoc bson.M
	findRound := options.FindOne().SetProjection(bson.D{
		{Key: "_id", Value: 0},
		{Key: "winner", Value: 1}, // winner.public_address, winner.vrf_public_key (binary)
	})
	roundErr := colRounds.FindOne(ctx, bson.D{{Key: "block_height", Value: bh}}, findRound).Decode(&roundDoc)

	xcashDPOPS := (roundErr == nil)
	delegateName := ""

	if xcashDPOPS {
		// Resolve winner.public_address -> delegates.delegate_name
		winnerAddr := ""
		if w, ok := roundDoc["winner"].(bson.M); ok {
			winnerAddr = asString(w["public_address"])
		}
		if winnerAddr != "" {
			colDelegates := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")
			var ddoc bson.M
			if err := colDelegates.FindOne(
				ctx,
				bson.D{{Key: "public_address", Value: winnerAddr}},
				options.FindOne().SetProjection(bson.D{
					{Key: "_id", Value: 0},
					{Key: "delegate_name", Value: 1},
				}),
			).Decode(&ddoc); err == nil {
				delegateName = asString(ddoc["delegate_name"])
			}
		}
	}

	// 4) Build response
	output.Height = block.Result.BlockHeader.Height
	output.Hash = block.Result.BlockHeader.Hash
	output.Reward = block.Result.BlockHeader.Reward
	output.Time = block.Result.BlockHeader.Timestamp
	output.XcashDPOPS = xcashDPOPS
	output.DelegateName = delegateName
	output.Tx = blockJSON.TxHashes

	return c.JSON(output)
}