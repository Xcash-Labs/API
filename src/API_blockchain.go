package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"github.com/gofiber/fiber/v2"
)

// Handlers

func v2_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {
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

	// 3) Parse embedded block JSON to get tx list (best-effort)
	var txHashes []string
	if block.Result.JSON != "" {
		raw := block.Result.JSON

		// If the daemon returned JSON as a quoted string with escapes, unquote it safely.
		if uq, err := strconv.Unquote(raw); err == nil {
			raw = uq
		} else {
			// fallback (older approach)
			raw = strings.ReplaceAll(raw, `\n`, "")
			raw = strings.ReplaceAll(raw, `\`, "")
		}

		var bj BlockchainBlockJson
		if err := json.Unmarshal([]byte(raw), &bj); err == nil {
			txHashes = bj.TxHashes
		}
	}

	// 4) Compute DPoPS flag: first DPoPS block is height 3
	h := block.Result.BlockHeader.Height // (int)
	xcashDPOPS := h >= 3

	// 5) Build response using your existing struct types directly
	out.Height = h
	out.Hash = block.Result.BlockHeader.Hash
	out.Reward = block.Result.BlockHeader.Reward // (int64)
	out.Time = block.Result.BlockHeader.Timestamp // (int)
	out.XcashDPOPS = xcashDPOPS
	out.TxHashes = txHashes

	return c.JSON(out)
}

func v2_xcash_blockchain_unauthorized_height(c *fiber.Ctx) error {
	dataSend, err := send_http_get("http://127.0.0.1:18281/get_height")
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "daemon unreachable"})
	}

	var gh struct {
		Height int    `json:"height"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(dataSend), &gh); err != nil || gh.Status != "OK" || gh.Height <= 0 {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "invalid daemon response"})
	}

	return c.JSON(fiber.Map{"height": gh.Height})
}