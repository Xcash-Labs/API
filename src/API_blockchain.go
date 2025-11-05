package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Handlers

// File: src/API_blockchain.go
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ---- RPC types ----

type rpcRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcGetBlockHeaderByHeightResp struct {
	Result struct {
		BlockHeader struct {
			Hash      string `json:"hash"`
			Height    uint64 `json:"height"`
			Reward    uint64 `json:"reward"`
			Timestamp uint64 `json:"timestamp"`
		} `json:"block_header"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type rpcGetBlockResp struct {
	Result struct {
		// "json" is a JSON string representing the block
		JSON string `json:"json"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Shape of block JSON from get_block: we only grab what's needed
type blockJSON struct {
	MinerTx struct {
		Extra []string `json:"extra"` // array of hex chunks
	} `json:"miner_tx"`
}

// ---- Response shape you return ----

type apiBlockResp struct {
	Height       uint64   `json:"height"`
	Hash         string   `json:"hash"`
	Reward       uint64   `json:"reward"`
	Time         uint64   `json:"time"`
	XcashDPOPS   bool     `json:"xcashDPOPS"`
	DelegateName string   `json:"delegateName"`
	Tx           []string `json:"tx"`
}

// ---- Helper: daemon call ----

func callDaemonRPC(method string, params interface{}, out interface{}) error {
	reqBody := rpcRequest{
		JsonRPC: "2.0",
		ID:      "0",
		Method:  method,
		Params:  params,
	}
	b, _ := json.Marshal(reqBody)

	httpClient := &http.Client{Timeout: 6 * time.Second}
	resp, err := httpClient.Post("http://127.0.0.1:18281/json_rpc", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	// Check error envelope if present
	switch v := out.(type) {
	case *rpcGetBlockHeaderByHeightResp:
		if v.Error != nil {
			return fmt.Errorf("daemon RPC error %d: %s", v.Error.Code, v.Error.Message)
		}
	case *rpcGetBlockResp:
		if v.Error != nil {
			return fmt.Errorf("daemon RPC error %d: %s", v.Error.Code, v.Error.Message)
		}
	}
	return nil
}

// ---- Helper: join miner_tx.extra into one contiguous hex string ----

func joinExtraHex(extra []string) string {
	// Sometimes it's an array of "hex bytes"; sometimes already concatenated. Normalize.
	if len(extra) == 1 {
		return strings.ToLower(strings.TrimPrefix(extra[0], "0x"))
	}
	var sb strings.Builder
	for _, e := range extra {
		sb.WriteString(strings.ToLower(strings.TrimPrefix(e, "0x")))
	}
	return sb.String()
}

// ---- Helper: find custom tags and try to extract fields ----
// We assume a simple TLV: <tag:1><len:1><value:len>... for 0x07 and 0xFA,
// which is how most Monero tx_extra items are structured.
// Adjust parsing if your actual layout differs.

type parsedExtra struct {
	HasVRF07       bool
	HasPublicFA    bool
	VRFPubKeyHex   string // from 0x07, if present
	DelegateAddr   string // from 0xFA, if present (public address)
}

func parseTxExtraForXcash(extraHex string) parsedExtra {
	out := parsedExtra{}
	b, err := hex.DecodeString(extraHex)
	if err != nil || len(b) < 2 {
		return out
	}

	i := 0
	for i+2 <= len(b) {
		tag := b[i]
		i++
		l := int(b[i])
		i++
		if i+l > len(b) || l < 0 {
			break
		}
		val := b[i : i+l]

		switch tag {
		case 0x07:
			// Your VRF tag packed fields; we only need the pubkey if it’s included.
			// Heuristic: try to read a 32 or 33 byte pubkey inside.
			out.HasVRF07 = true
			if len(val) >= 32 {
				out.VRFPubKeyHex = strings.ToLower(hex.EncodeToString(val[len(val)-32:]))
			}
		case 0xFA:
			// Your public tx tag; ideally it includes the delegate’s address in ASCII or as a TLV subfield.
			out.HasPublicFA = true
			// Try to extract an ASCII address if present
			asASCII := string(val)
			// Very rough check for X-Cash style address (starts with 'X' and long)
			if strings.HasPrefix(asASCII, "X") && len(asASCII) > 70 {
				out.DelegateAddr = strings.TrimSpace(asASCII)
			}
		default:
			// ignore others
		}
		i += l
	}
	return out
}

// ---- Helper: resolve delegate name from Mongo ----

func resolveDelegateName(ctx context.Context, mc *mongo.Client, p parsedExtra) (string, error) {
	if mc == nil {
		return "", errors.New("no mongo client")
	}
	db := mc.Database(XCASH_DPOPS_DATABASE)

	// 1) If we have a direct delegate address from 0xFA, use it
	if p.DelegateAddr != "" {
		var doc struct {
			Name string `bson:"name"`
		}
		err := db.Collection("delegates").
			FindOne(ctx, bson.M{"public_address": p.DelegateAddr}, nil).
			Decode(&doc)
		if err == nil && doc.Name != "" {
			return doc.Name, nil
		}
	}

	// 2) Try winner from a rounds collection by vrf pk (if present)
	if p.VRFPubKeyHex != "" {
		// try delegates by vrf_public_key
		var doc2 struct {
			Name string `bson:"name"`
		}
		err := db.Collection("delegates").
			FindOne(ctx, bson.M{"vrf_public_key": p.VRFPubKeyHex}, nil).
			Decode(&doc2)
		if err == nil && doc2.Name != "" {
			return doc2.Name, nil
		}
	}

	// 3) Optional: a direct rounds entry that captured the winner
	var round struct {
		Winner struct {
			Name    string `bson:"name"`
			Address string `bson:"address"`
			VRFPK   string `bson:"vrf_pk"`
		} `bson:"winner"`
	}
	// We don’t know the height here, so caller can attempt a separate rounds lookup by height if desired.
	// Leaving this path as a soft attempt without a filter.
	_ = db.Collection("consensus_rounds").FindOne(ctx, bson.M{}, nil).Decode(&round) // best-effort
	if round.Winner.Name != "" {
		return round.Winner.Name, nil
	}

	return "", errors.New("delegate not found")
}

// Handaler

func v2_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {
	// Path param: /v2/xcash/blockchain/unauthorized/blocks/:height?
	heightStr := c.Params("height")
	if heightStr == "" {
		heightStr = c.Query("height") // support ?height= as well
	}
	if heightStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing height"})
	}
	h, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid height"})
	}

	// 1) Header
	var hdr rpcGetBlockHeaderByHeightResp
	if err := callDaemonRPC("get_block_header_by_height", map[string]uint64{"height": h}, &hdr); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "daemon error: " + err.Error()})
	}
	bh := hdr.Result.BlockHeader

	// 2) Full block
	var blk rpcGetBlockResp
	if err := callDaemonRPC("get_block", map[string]uint64{"height": h}, &blk); err != nil {
		// We can still answer with header data
		resp := apiBlockResp{
			Height:       bh.Height,
			Hash:         bh.Hash,
			Reward:       bh.Reward,
			Time:         bh.Timestamp,
			XcashDPOPS:   false,
			DelegateName: "",
			Tx:           []string{},
		}
		return c.JSON(resp)
	}

	// Parse block JSON to get miner_tx.extra
	var bj blockJSON
	if err := json.Unmarshal([]byte(blk.Result.JSON), &bj); err != nil {
		// Fallback to header-only response
		resp := apiBlockResp{
			Height:       bh.Height,
			Hash:         bh.Hash,
			Reward:       bh.Reward,
			Time:         bh.Timestamp,
			XcashDPOPS:   false,
			DelegateName: "",
			Tx:           []string{},
		}
		return c.JSON(resp)
	}

	extraHex := joinExtraHex(bj.MinerTx.Extra)
	parsed := parseTxExtraForXcash(extraHex)

	// Determine xcashDPOPS (true if we detect either custom tag)
	xcashDPOPS := parsed.HasVRF07 || parsed.HasPublicFA

	// Try to resolve delegate name if Mongo is available
	delegateName := ""
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if name, er := resolveDelegateName(ctx, mongoClient, parsed); er == nil {
			delegateName = name
		}
	}

	resp := apiBlockResp{
		Height:       bh.Height,
		Hash:         bh.Hash,
		Reward:       bh.Reward,
		Time:         bh.Timestamp,
		XcashDPOPS:   xcashDPOPS,
		DelegateName: delegateName,
		Tx:           []string{},
	}
	return c.JSON(resp)
}