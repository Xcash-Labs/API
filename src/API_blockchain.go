package main

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// Handlers

func v2_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {
	// DB required for DPOPS bits
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	var (
		out          v2XcashBlockchainUnauthorizedBlocksBlockHeight
		reqHeightStr = strings.TrimSpace(c.Params("blockHeight"))
	)

	// 1) Determine height (default: latest-1)
	if reqHeightStr == "" {
		dataSend, err := send_http_data("http://127.0.0.1:18281/json_rpc", `{"jsonrpc":"2.0","id":"0","method":"get_info"}`)
		if err != nil || !strings.Contains(dataSend, `"result"`) {
			return c.JSON(ErrorResults{"Could not get the block data"})
		}
		var info BlockchainStats
		if err := json.Unmarshal([]byte(dataSend), &info); err != nil || info.Result.Height == 0 {
			return c.JSON(ErrorResults{"Could not get the block data"})
		}
		h := int64(info.Result.Height - 1)
		if h < 0 {
			h = 0
		}
		reqHeightStr = strconv.FormatInt(h, 10)
	}

	// 2) Fetch block by height
	dataSend, err := send_http_data(
		"http://127.0.0.1:18281/json_rpc",
		`{"jsonrpc":"2.0","id":"0","method":"get_block","params":{"height":`+reqHeightStr+`}}`,
	)
	if err != nil || !strings.Contains(dataSend, `"result"`) {
		return c.JSON(ErrorResults{"Could not get the block data"})
	}
	var block BlockchainBlock
	if err := json.Unmarshal([]byte(dataSend), &block); err != nil {
		return c.JSON(ErrorResults{"Could not get the block data"})
	}

	// 3) Parse embedded JSON (tx list)
	raw := strings.ReplaceAll(string(block.Result.JSON), `\n`, "")
	raw = strings.ReplaceAll(raw, `\`, "")
	var blockJSON BlockchainBlockJson
	if err := json.Unmarshal([]byte(raw), &blockJSON); err != nil {
		// Not fatal for basic block data; keep going with empty tx list
		blockJSON.TxHashes = nil
	}

	// ---- DPOPS lookup (consensus_rounds) ----
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	psDB := mongoClient.Database(XCASH_DPOPS_DATABASE) // "XCASH_PROOF_OF_STAKE"
	colRounds := psDB.Collection("consensus_rounds")

	bh := int64(block.Result.BlockHeader.Height)
	var roundDoc bson.M
	findRound := options.FindOne().SetProjection(bson.D{
		{Key: "_id", Value: 0},
		{Key: "winner", Value: 1}, // winner.public_address (string), winner.vrf_public_key (bin)
	})

	roundErr := colRounds.FindOne(ctx, bson.D{{Key: "block_height", Value: bh}}, findRound).Decode(&roundDoc)
	xcashDPOPS := (roundErr == nil)
	delegateName := ""

	if xcashDPOPS {
		// Resolve winner.public_address -> delegates.delegate_name
		if w, ok := roundDoc["winner"].(bson.M); ok {
			if winnerAddr := asString(w["public_address"]); winnerAddr != "" {
				var ddoc bson.M
				if err := psDB.Collection("delegates").FindOne(
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
	}

	// 4) Build response
	out.Height = block.Result.BlockHeader.Height
	out.Hash = block.Result.BlockHeader.Hash
	out.Reward = block.Result.BlockHeader.Reward
	out.Time = block.Result.BlockHeader.Timestamp
	out.XcashDPOPS = xcashDPOPS
	out.DelegateName = delegateName
	out.Tx = blockJSON.TxHashes

	return c.JSON(out)
}