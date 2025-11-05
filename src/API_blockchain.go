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

// Handlers

func v2_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {
	var (
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

	// 3) Parse embedded JSON to get tx list (best-effort)
	var txHashes []string
	if block.Result.JSON != "" {
		// Unescape and decode the embedded block JSON
		raw := strings.ReplaceAll(block.Result.JSON, `\n`, "")
		raw = strings.ReplaceAll(raw, `\`, "")
		var bj BlockchainBlockJson
		if err := json.Unmarshal([]byte(raw), &bj); err == nil {
			txHashes = bj.TxHashes
		}
	}

	// 4) Compute DPoPS flag purely by height: first DPoPS block is height 3
	h := block.Result.BlockHeader.Height
	xcashDPOPS := h > XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT

	// 5) Build and return response (no delegateName)
	resp := struct {
		Height     uint64   `json:"height"`
		Hash       string   `json:"hash"`
		Reward     uint64   `json:"reward"`
		Time       uint64   `json:"time"`
		XcashDPOPS bool     `json:"xcashDPOPS"`
		Tx         []string `json:"tx"`
	}{
		Height:     h,
		Hash:       block.Result.BlockHeader.Hash,
		Reward:     block.Result.BlockHeader.Reward,
		Time:       block.Result.BlockHeader.Timestamp,
		XcashDPOPS: xcashDPOPS,
		Tx:         txHashes,
	}

	return c.JSON(resp)
}