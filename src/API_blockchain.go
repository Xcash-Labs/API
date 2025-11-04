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

func v1_xcash_blockchain_unauthorized_blocks_blockHeight(c *fiber.Ctx) error {

	// Variables
	var data_send string
	var data_read_1 BlockchainStats
	var data_read_2 BlockchainBlock
	var data_read_3 BlockchainBlockJson
	var output v1XcashBlockchainUnauthorizedBlocksBlockHeight
	var requestBlockHeight string
	var xcash_dpops_status bool
	var xcash_dpops_delegate string
	var error error

	// get info
	data_send, error = send_http_data("http://127.0.0.1:18281/json_rpc", `{"jsonrpc":"2.0","id":"0","method":"get_info"}`)

	if !strings.Contains(data_send, "\"result\"") || error != nil {
		error := ErrorResults{"Could not get the block data"}
		return c.JSON(error)
	}

	if err := json.Unmarshal([]byte(data_send), &data_read_1); err != nil {
		error := ErrorResults{"Could not get the block data"}
		return c.JSON(error)
	}

	// get the resource
	requestBlockHeight = c.Params("blockHeight")
	if requestBlockHeight == "" {
		requestBlockHeight = strconv.FormatInt(int64(data_read_1.Result.Height-1), 10)
	}

	// get block
	data_send, error = send_http_data("http://127.0.0.1:18281/json_rpc", `{"jsonrpc":"2.0","id":"0","method":"get_block","params":{"height":`+requestBlockHeight+`}}`)
	if !strings.Contains(data_send, "\"result\"") || error != nil {
		error := ErrorResults{"Could not get the block data"}
		return c.JSON(error)
	}

	if err := json.Unmarshal([]byte(data_send), &data_read_2); err != nil {
		error := ErrorResults{"Could not get the block data"}
		return c.JSON(error)
	}

	// get the tx
	s := string(data_read_2.Result.JSON)
	s = strings.Replace(s, "\\n", "", -1)
	s = strings.Replace(s, "\\", "", -1)
	if err := json.Unmarshal([]byte(s), &data_read_3); err != nil {
		error := ErrorResults{"Could not get the block data"}
		return c.JSON(error)
	}

	// get the dpops block status
	if data_read_2.Result.BlockHeader.Height >= XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT {
		xcash_dpops_status = true
		xcash_dpops_delegate = get_block_delegate(data_read_2.Result.BlockHeader.Height)
	} else {
		xcash_dpops_status = false
		xcash_dpops_delegate = ""
	}

	// fill in the data
	output.Height = data_read_2.Result.BlockHeader.Height
	output.Hash = data_read_2.Result.BlockHeader.Hash
	output.Reward = data_read_2.Result.BlockHeader.Reward
	output.Time = data_read_2.Result.BlockHeader.Timestamp
	output.XcashDPOPS = xcash_dpops_status
	output.DelegateName = xcash_dpops_delegate
	output.Tx = data_read_3.TxHashes

	return c.JSON(output)
}