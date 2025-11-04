package main

import (
	"strings"
	"strconv"
	"sort"
	"context"
	"encoding/hex"
//	"encoding/json"
	"time"
    "fmt"

//	"math"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


func varint_decode(s string) int64 {
	// Variables
	var varint int64
	var length int = 0
	var count int = 0
	var counter int = 0
	var bytecount int = 0
	var number int64 = 1
	var start int = 0
	const BITS_IN_BYTE = 8

	// convert the string to decimal
	varint, _ = strconv.ParseInt(s, 16, 64)

	// get the length
	if varint <= 0xFF {
		return varint
	} else if varint > 0xFF && varint < 0xFFFF {
		length = 2
	} else if varint >= 0xFFFF && varint < 0xFFFFFF {
		length = 3
	} else if varint >= 0xFFFFFF && varint < 0xFFFFFFFF {
		length = 4
	} else if varint >= 0xFFFFFFFF && varint < 0xFFFFFFFFFF {
		length = 5
	} else if varint >= 0xFFFFFFFFFF && varint < 0xFFFFFFFFFFFF {
		length = 6
	} else if varint >= 0xFFFFFFFFFFFF && varint < 0xFFFFFFFFFFFFFF {
		length = 7
	} else {
		length = 8
	}

	// create a byte array for the varint
	bytes := make([]int8, length)

	for count = 0; count < length; count++ {
		// convert each byte to binary and read the bytes in reverse order
		bytes[count] = int8(((varint >> (BITS_IN_BYTE * uint(count))) & 0xFF))
	}

	counter = (BITS_IN_BYTE - 1)
	bytecount = 0
	start = 0

	for count = 0; count < length*BITS_IN_BYTE; count++ {
		// loop through each bit until you find the first 1. for every bit after this:
		// if 0 then number = number * 2;
		// if 1 then number = (number * 2) + 1;
		// dont use the bit if its the first bit
		if counter != (BITS_IN_BYTE - 1) {
			if (bytes[bytecount] & (1 << uint(counter))) != 0 {
				if start == 1 {
					number = (number * 2) + 1
				}
				start = 1
			} else {
				if start == 1 {
					number = number * 2
				}
			}
		}

		if counter == 0 {
			counter = (BITS_IN_BYTE - 1)
			bytecount++
		} else {
			counter--
		}
	}
	return number
}

func get_reserve_bytes(block_height int) string {
	var database_data XcashDpopsReserveBytesCollection

	// get the collection
	block_height_data := strconv.Itoa(int(((block_height - XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT) / BLOCKS_PER_DAY_FIVE_MINUTE_BLOCK_TIME)) + 1)
	collection_number := "reserve_bytes_" + block_height_data
	collection := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection(collection_number)

	// get the reserve bytes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, bson.D{{"block_height", strconv.Itoa(block_height)}}).Decode(&database_data)
	if err == mongo.ErrNoDocuments {
		return ""
	} else if err != nil {
		return ""
	}
	return database_data.ReserveBytes
}

func get_delegate_address_from_name(delegate string) string {
	var database_data XcashDpopsDelegatesCollection

	// set the collection
	collection := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")

	// get the delegates data
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, bson.D{{"delegate_name", delegate}}).Decode(&database_data)
	if err == mongo.ErrNoDocuments {
		return ""
	} else if err != nil {
		return ""
	}
	return database_data.PublicAddress
}

func get_delegate_name_from_address(address string) string {
	var database_data XcashDpopsDelegatesCollection

	// set the collection
	collection := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")

	// get the delegates data
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, bson.D{{"public_address", address}}).Decode(&database_data)
	if err == mongo.ErrNoDocuments {
		return ""
	} else if err != nil {
		return ""
	}
	return database_data.DelegateName
}






func v2_xcash_dpops_unauthorized_delegates_registered(c *fiber.Ctx) error {
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	db := mongoClient.Database(XCASH_DPOPS_DATABASE)
	colDelegates := db.Collection("delegates")
	colProofs := db.Collection("reserve_proofs")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) Load delegates (project only what we use)
	proj := bson.D{
		{Key: "_id", Value: 0},
		{Key: "public_address", Value: 1},
		{Key: "delegate_name", Value: 1},
		{Key: "delegate_type", Value: 1},
		{Key: "IP_address", Value: 1},
		{Key: "shared_delegate_status", Value: 1},
		{Key: "online_status", Value: 1},
		{Key: "delegate_fee", Value: 1},
		{Key: "block_verifier_total_rounds", Value: 1},
		{Key: "block_producer_total_rounds", Value: 1},
		{Key: "block_verifier_online_percentage", Value: 1},
		{Key: "total_vote_count", Value: 1}, // prefer this if present
	}
	cur, err := colDelegates.Find(ctx, bson.D{}, options.Find().SetProjection(proj))
	if err != nil {
		return c.JSON(ErrorResults{"Could not get the delegates registered"})
	}
	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return c.JSON(ErrorResults{"Could not get the delegates registered"})
	}

	// 2) Pre-aggregate voters (and vote sums as fallback) from reserve_proofs
	//    Group by public_address_voted_for -> { voters: count, sumVotes: sum(total_vote) }
	type aggRow struct {
		ID        string      `bson:"_id"`
		Voters    int32       `bson:"voters"`
		SumVotes  interface{} `bson:"sumVotes"` // NumberLong/Decimal/etc.
	}
	var agg []aggRow
	pipe := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$public_address_voted_for"},
			{Key: "voters", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "sumVotes", Value: bson.D{{Key: "$sum", Value: "$total_vote"}}},
		}}},
	}
	if aCur, err := colProofs.Aggregate(ctx, pipe); err == nil {
		_ = aCur.All(ctx, &agg)
		_ = aCur.Close(ctx)
	}
	// Build quick lookup map
	voterMap := make(map[string]struct {
		voters   int
		sumVotes int64
	}, len(agg))
	for _, r := range agg {
		voterMap[r.ID] = struct {
			voters   int
			sumVotes int64
		}{
			voters:   int(r.Voters),
			sumVotes: toInt64(r.SumVotes),
		}
	}

	// 3) Build output rows
	out := make([]*v2XcashDpopsUnauthorizedDelegatesBasicData, 0, len(docs))
	for _, it := range docs {
		row := new(v2XcashDpopsUnauthorizedDelegatesBasicData)

		pubAddr := asString(it["public_address"])
		row.DelegateName = asString(it["delegate_name"])
		row.IPAdress = asString(it["IP_address"])

		// shared_delegate_status can be "solo"/"shared" or bool
		switch v := it["shared_delegate_status"].(type) {
		case string:
			row.SharedDelegate = (v != "solo")
		case bool:
			row.SharedDelegate = v
		default:
			row.SharedDelegate = false
		}

		// online_status can be bool or string
		switch v := it["online_status"].(type) {
		case bool:
			row.Online = v
		case string:
			row.Online = strings.EqualFold(v, "true")
		default:
			row.Online = false
		}

		// numeric fields (handle NumberLong, string, etc.)
		row.Fee = int(toInt64(it["delegate_fee"]))
		row.TotalRounds = int(toInt64(it["block_verifier_total_rounds"]))
		row.TotalBlockProducerRounds = int(toInt64(it["block_producer_total_rounds"]))
		row.OnlinePercentage = int(toInt64(it["block_verifier_online_percentage"]))

		// Votes: prefer delegate.total_vote_count; fallback to reserve_proofs sum
		votes := toInt64(it["total_vote_count"])
		if votes == 0 {
			if v, ok := voterMap[pubAddr]; ok {
				votes = v.sumVotes
			}
		}
		row.Votes = votes

		// Voters: from reserve_proofs aggregation
		if v, ok := voterMap[pubAddr]; ok {
			row.Voters = v.voters
		} else {
			row.Voters = 0
		}

		// Keep struct shape: no SeedNode now (remove logic; default false)
		row.SeedNode = false

		// Stash delegate_type for sorting (not in output struct; use local var)
		itType := asString(it["delegate_type"])
		// Attach as JSON field if your struct includes it; otherwise keep only for sort.
		// If you want it in the response, add a field to v2XcashDpopsUnauthorizedDelegatesBasicData.

		// Attach type temporarily via a map to keep sort simple
		cType := itType
		// Abuse the address field? No. Let's carry a parallel slice for types.
		// We'll sort using a keyed comparator below; see sort closure.

		// To keep it simple, we’ll append and rely on comparator closure capturing cType
		rowCopy := *row
		_ = cType // captured in comparator via index map built below
		out = append(out, &rowCopy)
	}

	// Build a parallel slice of delegate_type for sorting (aligned by index)
	types := make([]string, len(out))
	for i, it := range docs {
		types[i] = asString(it["delegate_type"])
	}

	// 4) Sort: delegate_type asc, Online true first, Votes desc
	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := types[i], types[j]
		if ti != tj {
			return ti < tj
		}
		if out[i].Online != out[j].Online {
			return out[i].Online // true first
		}
		return out[i].Votes > out[j].Votes
	})

	return c.JSON(out)
}








func toInt64(v any) int64 {
    switch t := v.(type) {
    case int64:
        return t
    case int32:
        return int64(t)
    case int:
        return int64(t)
    case float64:
        return int64(t)
    case string:
        if n, err := strconv.ParseInt(t, 10, 64); err == nil {
            return n
        }
        return 0
    case primitive.Decimal128:
        bi, _, errDec := t.BigInt() // v1.17.6: (*big.Int, scale int, error)
        if errDec != nil || bi == nil {
            return 0
        }
        return bi.Int64()
    default:
        return 0
    }
}


func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}














func v1_xcash_dpops_unauthorized_delegates_online(c *fiber.Ctx) error {

	// Variables
	output := []*v2XcashDpopsUnauthorizedDelegatesBasicData{}
	var mongo_sort *mongo.Cursor
	var mongo_results []bson.M
	var err error

	// setup database
	collection_delegates := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get total delegates votes
	mongo_sort, err = collection_delegates.Find(ctx, bson.D{{}})
	if err != nil {
		error := ErrorResults{"Could not get the delegates online"}
		return c.JSON(error)
	}

	if err = mongo_sort.All(ctx, &mongo_results); err != nil {
		error := ErrorResults{"Could not get the delegates online"}
		return c.JSON(error)
	}

	for _, item := range mongo_results {
		if strings.Contains(item["online_status"].(string), "false") {
			continue
		}

		// fill in the data
		data := new(v2XcashDpopsUnauthorizedDelegatesBasicData)
		data.Votes, _ = strconv.ParseInt(item["total_vote_count"].(string), 10, 64)

		// get the total voters for the delegates
		total_voters := 0
		for count4 := 1; count4 < TOTAL_RESERVE_PROOFS_DATABASES; count4++ {
			count2, _ := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("reserve_proofs_"+string(count4)).CountDocuments(ctx, bson.D{{"public_address_voted_for", item["public_address"].(string)}})
			total_voters += int(count2)
		}
		data.Voters = total_voters

		data.IPAdress = item["IP_address"].(string)
		data.DelegateName = item["delegate_name"].(string)
		if item["shared_delegate_status"].(string) == "solo" {
			data.SharedDelegate = false
		} else {
			data.SharedDelegate = true
		}

		if strings.Contains(item["IP_address"].(string), ".xcash.tech") {
			data.SeedNode = true
		} else {
			data.SeedNode = false
		}

		data.Online = true

		data.Fee, _ = strconv.Atoi(item["delegate_fee"].(string))
		data.TotalRounds, _ = strconv.Atoi(item["block_verifier_total_rounds"].(string))
		data.TotalBlockProducerRounds, _ = strconv.Atoi(item["block_producer_total_rounds"].(string))
		data.OnlinePercentage, _ = strconv.Atoi(item["block_verifier_online_percentage"].(string))

		output = append(output, data)
	}

	// sort the arrray by how xcash dpops sorts the delegates
	sort.Slice(output[:], func(i, j int) bool {
		var count1 int
		var count2 int

		// check if the delegate is a network data node
		if output[i].IPAdress == "seed1.xcash.tech" {
			count1 = 3
		} else if output[i].IPAdress == "seed2.xcash.tech" {
			count1 = 2
		} else if output[i].IPAdress == "seed3.xcash.tech" {
			count1 = 1
		} else {
			count1 = 0
		}

		if output[j].IPAdress == "seed1.xcash.tech" {
			count2 = 3
		} else if output[j].IPAdress == "seed2.xcash.tech" {
			count2 = 2
		} else if output[j].IPAdress == "seed3.xcash.tech" {
			count2 = 1
		} else {
			count2 = 0
		}

		if count1 != count2 {
			if count2-count1 < 0 {
				return true
			} else {
				return false
			}
		}

		return output[i].Votes > output[j].Votes
	})

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_delegates_active(c *fiber.Ctx) error {

	// Variables
	output := []*v2XcashDpopsUnauthorizedDelegatesBasicData{}
	var mongo_sort *mongo.Cursor
	var mongo_results []bson.M
	var err error

	// setup database
	collection_delegates := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get total delegates votes
	mongo_sort, err = collection_delegates.Find(ctx, bson.D{{}})
	if err != nil {
		error := ErrorResults{"Could not get the delegates active"}
		return c.JSON(error)
	}

	if err = mongo_sort.All(ctx, &mongo_results); err != nil {
		error := ErrorResults{"Could not get the delegates active"}
		return c.JSON(error)
	}

	for _, item := range mongo_results {
		// fill in the data
		data := new(v2XcashDpopsUnauthorizedDelegatesBasicData)
		data.Votes, _ = strconv.ParseInt(item["total_vote_count"].(string), 10, 64)

		// get the total voters for the delegates
		total_voters := 0
		for count4 := 1; count4 < TOTAL_RESERVE_PROOFS_DATABASES; count4++ {
			count2, _ := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("reserve_proofs_"+string(count4)).CountDocuments(ctx, bson.D{{"public_address_voted_for", item["public_address"].(string)}})
			total_voters += int(count2)
		}
		data.Voters = total_voters

		data.IPAdress = item["IP_address"].(string)
		data.DelegateName = item["delegate_name"].(string)
		if item["shared_delegate_status"].(string) == "solo" {
			data.SharedDelegate = false
		} else {
			data.SharedDelegate = true
		}

		if strings.Contains(item["IP_address"].(string), ".xcash.tech") {
			data.SeedNode = true
		} else {
			data.SeedNode = false
		}

		if strings.Contains(item["online_status"].(string), "true") {
			data.Online = true
		} else {
			data.Online = false
		}

		data.Fee, _ = strconv.Atoi(item["delegate_fee"].(string))
		data.TotalRounds, _ = strconv.Atoi(item["block_verifier_total_rounds"].(string))
		data.TotalBlockProducerRounds, _ = strconv.Atoi(item["block_producer_total_rounds"].(string))
		data.OnlinePercentage, _ = strconv.Atoi(item["block_verifier_online_percentage"].(string))

		output = append(output, data)
	}

	// sort the arrray by how xcash dpops sorts the delegates
	sort.Slice(output[:], func(i, j int) bool {
		var count1 int
		var count2 int

		// check if the delegate is a network data node
		if output[i].IPAdress == "seed1.xcash.tech" {
			count1 = 3
		} else if output[i].IPAdress == "seed2.xcash.tech" {
			count1 = 2
		} else if output[i].IPAdress == "seed3.xcash.tech" {
			count1 = 1
		} else {
			count1 = 0
		}

		if output[j].IPAdress == "seed1.xcash.tech" {
			count2 = 3
		} else if output[j].IPAdress == "seed2.xcash.tech" {
			count2 = 2
		} else if output[j].IPAdress == "seed3.xcash.tech" {
			count2 = 1
		} else {
			count2 = 0
		}

		if count1 != count2 {
			if count2-count1 < 0 {
				return true
			} else {
				return false
			}
		}

		// check if the delegate is online
		if output[i].Online != output[j].Online {
			if output[i].Online == true {
				return true
			} else {
				return false
			}
		}
		return output[i].Votes > output[j].Votes
	})

	// only return the top 50
	if len(output) > BLOCK_VERIFIERS_AMOUNT {
		output = output[0:BLOCK_VERIFIERS_AMOUNT]
	}

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_delegates(c *fiber.Ctx) error {

	// Variables
	output_data := []*v2XcashDpopsUnauthorizedDelegatesBasicData{}
	var mongo_sort *mongo.Cursor
	var mongo_results []bson.M
	var delegate string
	var database_data_delegates XcashDpopsDelegatesCollection
	var output v1XcashDpopsUnauthorizedDelegatesData
	var total_voters int

	delegate = c.Params("delegateName")
	if delegate == "" {
		error := ErrorResults{"Could not get the delegates data"}
		return c.JSON(error)
	}

	// setup database
	collection_delegates := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get the delegate
	err := collection_delegates.FindOne(ctx, bson.D{{"delegate_name", delegate}}).Decode(&database_data_delegates)
	if err == mongo.ErrNoDocuments {
		error := ErrorResults{"Could not get the delegates data"}
		return c.JSON(error)
	} else if err != nil {
		error := ErrorResults{"Could not get the delegates data"}
		return c.JSON(error)
	}

	if database_data_delegates.OnlineStatus == "true" {
		output.Online = true
	} else {
		output.Online = false
	}

	// get total voters
	for count4 := 1; count4 < TOTAL_RESERVE_PROOFS_DATABASES; count4++ {
		count2, _ := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("reserve_proofs_"+string(count4)).CountDocuments(ctx, bson.D{{"public_address_voted_for", database_data_delegates.PublicAddress}})
		total_voters += int(count2)
	}

	mongo_sort, err = collection_delegates.Find(ctx, bson.D{{}})
	if err != nil {
		error := ErrorResults{"Could not get the delegates data"}
		return c.JSON(error)
	}

	if err = mongo_sort.All(ctx, &mongo_results); err != nil {
		error := ErrorResults{"Could not get the delegates data"}
		return c.JSON(error)
	}

	for _, item := range mongo_results {
		// fill in the data
		data := new(v2XcashDpopsUnauthorizedDelegatesBasicData)
		data.DelegateName = item["delegate_name"].(string)
		data.Votes, _ = strconv.ParseInt(item["total_vote_count"].(string), 10, 64)
		data.IPAdress = item["IP_address"].(string)
		if strings.Contains(item["online_status"].(string), "true") {
			data.Online = true
		} else {
			data.Online = false
		}
		output_data = append(output_data, data)
	}

	// sort the arrray by how xcash dpops sorts the delegates
	sort.Slice(output_data[:], func(i, j int) bool {
		var count1 int
		var count2 int

		// check if the delegate is a network data node
		if output_data[i].IPAdress == "seed1.xcash.tech" {
			count1 = 3
		} else if output_data[i].IPAdress == "seed2.xcash.tech" {
			count1 = 2
		} else if output_data[i].IPAdress == "seed3.xcash.tech" {
			count1 = 1
		} else {
			count1 = 0
		}

		if output_data[j].IPAdress == "seed1.xcash.tech" {
			count2 = 3
		} else if output_data[j].IPAdress == "seed2.xcash.tech" {
			count2 = 2
		} else if output_data[j].IPAdress == "seed3.xcash.tech" {
			count2 = 1
		} else {
			count2 = 0
		}

		if count1 != count2 {
			if count2-count1 < 0 {
				return true
			} else {
				return false
			}
		}

		// check if the delegate is online
		if output_data[i].Online != output_data[j].Online {
			if output_data[i].Online == true {
				return true
			} else {
				return false
			}
		}
		return output_data[i].Votes > output_data[j].Votes
	})

	// get the Rank
	for rank, item := range output_data {
		if item.DelegateName == database_data_delegates.DelegateName {
			output.Rank = rank + 1
			break
		}
	}

	if database_data_delegates.SharedDelegateStatus == "solo" {
		output.SharedDelegate = false
	} else {
		output.SharedDelegate = true
	}

	if strings.Contains(database_data_delegates.IPAddress, ".xcash.tech") {
		output.SeedNode = true
	} else {
		output.SeedNode = false
	}

	// fill in the data
	output.Votes, _ = strconv.ParseInt(database_data_delegates.TotalVoteCount, 10, 64)
	output.Voters = int(total_voters)
	output.IPAdress = database_data_delegates.IPAddress
	output.DelegateName = database_data_delegates.DelegateName
	output.PublicAddress = database_data_delegates.PublicAddress
	output.About = database_data_delegates.About
	output.Website = database_data_delegates.Website
	output.Team = database_data_delegates.Team
	output.Specifications = database_data_delegates.ServerSpecs
	output.Fee, _ = strconv.Atoi(database_data_delegates.DelegateFee)
	output.TotalRounds, _ = strconv.Atoi(database_data_delegates.BlockVerifierTotalRounds)
	output.TotalBlockProducerRounds, _ = strconv.Atoi(database_data_delegates.BlockProducerTotalRounds)
	output.OnlinePercentage, _ = strconv.Atoi(database_data_delegates.BlockVerifierOnlinePercentage)

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_delegates_rounds(c *fiber.Ctx) error {

	// Variables
	var output v1XcashDpopsUnauthorizedDelegatesRounds
	var delegate string
	var current_block_height int
	var mongo_sort *mongo.Cursor
	var error error
	var totalBlocksProduced int = 0
	var totalBlockRewards int64 = 0

	// setup database
	collection := mongoClient.Database(XCASH_API_DATABASE).Collection("blocks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get the resource
	if delegate = c.Params("delegateName"); delegate == "" {
		error := ErrorResults{"Could not get the delegate round details"}
		return c.JSON(error)
	}

	// check if the delegate is in the database
	count, err := collection.CountDocuments(ctx, bson.D{{"delegate", delegate}})
	if err != nil || int(count) == 0 {
		error := ErrorResults{"Could not get the delegate round details"}
		return c.JSON(error)
	}

	// get the previous block Height
	if current_block_height = get_current_block_height(); current_block_height == 0 {
		error := ErrorResults{"Could not get the delegate round details"}
		return c.JSON(error)
	}
	current_block_height -= 1
	current_block_height -= XCASH_PROOF_OF_STAKE_BLOCK_HEIGHT

	mongo_sort, error = collection.Find(ctx, bson.D{{"delegate", delegate}})
	if error != nil {
		error := ErrorResults{"Could not get the delegate round details"}
		return c.JSON(error)
	}

	var mongo_results []bson.M
	if error = mongo_sort.All(ctx, &mongo_results); error != nil {
		error := ErrorResults{"Could not get the delegate round details"}
		return c.JSON(error)
	}

	for _, item := range mongo_results {
		height, _ := strconv.Atoi(item["height"].(string))
		reward, _ := strconv.ParseInt(item["reward"].(string), 10, 64)
		timestamp, _ := strconv.Atoi(item["time"].(string))
		output.BlocksProduced = append(output.BlocksProduced, BlocksProduced{height, reward, timestamp})

		totalBlocksProduced++
		totalBlockRewards += reward
	}

	// fill in the data
	output.TotalBlocksProduced = totalBlocksProduced
	output.TotalBlockRewards = totalBlockRewards
	output.AveragePercentage = int((float64(current_block_height)) / (float64(totalBlocksProduced * (BLOCK_VERIFIERS_AMOUNT - 5))) * 100)
	output.AverageTime = int(float64((current_block_height * XCASH_DPOPS_BLOCK_TIME)) / float64(totalBlocksProduced))

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_delegates_votes(c *fiber.Ctx) error {

	// Variables
	output := []*v1XcashDpopsUnauthorizedDelegatesVotes{}
	var mongo_sort *mongo.Cursor
	var delegate string
	var count4 int
	var start int
	var limit int
	var err error

	// get the resource
	if delegate = c.Params("delegateName"); delegate == "" {
		error := ErrorResults{"Could not get the delegate vote details"}
		return c.JSON(error)
	}

	if start, _ = strconv.Atoi(c.Params("start")); c.Params("start") == "" || start < 0 {
		error := ErrorResults{"Could not get the delegate vote details"}
		return c.JSON(error)
	}

	if limit, _ = strconv.Atoi(c.Params("limit")); c.Params("limit") == "" || limit > MAXIMUM_AMOUNT_OF_VOTERS_PER_DELEGATE {
		error := ErrorResults{"Could not get the delegate vote details"}
		return c.JSON(error)
	}

	// get the delegates PublicAddress
	address := get_delegate_address_from_name(delegate)

	// setup database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for count4 = 1; count4 < TOTAL_RESERVE_PROOFS_DATABASES; count4++ {
		mongo_sort, err = mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("reserve_proofs_"+strconv.Itoa(count4)).Find(ctx, bson.D{{Key: "public_address_voted_for", Value: address}})
		if err != nil {
			continue
		}

		var mongo_results []bson.M
		if err = mongo_sort.All(ctx, &mongo_results); err != nil {
			continue
		}

		for _, item := range mongo_results {
			// fill in the data
			data := new(v1XcashDpopsUnauthorizedDelegatesVotes)
			data.PublicAddress = item["public_address_created_reserve_proof"].(string)
			data.ReserveProof = item["reserve_proof"].(string)
			data.Amount, _ = strconv.ParseInt(item["total"].(string), 10, 64)
			output = append(output, data)
		}

	}

	// sort the arrray by vote total
	sort.Slice(output[:], func(i, j int) bool {
		return output[i].Amount > output[j].Amount
	})

	// only return the start and limit
	if limit > len(output) {
		limit = len(output)
	}
	if start > len(output) {
		start = len(output)
	}
	output = output[start:limit]

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_votes(c *fiber.Ctx) error {

	// Variables
	var output v1XcashDpopsUnauthorizedVotes
	var address string
	var database_data XcashDpopsReserveProofsCollection
	var count4 int

	// setup database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get the resource
	if address = c.Params("address"); address == "" || address[0:len(XCASH_WALLET_PREFIX)] != XCASH_WALLET_PREFIX || len(address) != XCASH_WALLET_LENGTH {
		error := ErrorResults{"Could not get the vote details"}
		return c.JSON(error)
	}

	// get the votes
	for count4 = 1; count4 < TOTAL_RESERVE_PROOFS_DATABASES; count4++ {
		err := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("reserve_proofs_"+string(count4)).FindOne(ctx, bson.D{{"public_address_created_reserve_proof", address}}).Decode(&database_data)
		if err == mongo.ErrNoDocuments {
			continue
		} else if err != nil {
			continue
		}
	}

	if database_data.PublicAddressVotedFor == "" {
		error := ErrorResults{"This address has not voted"}
		return c.JSON(error)
	}

	// fill in the data
	output.DelegateName = get_delegate_name_from_address(database_data.PublicAddressVotedFor)
	output.Amount, _ = strconv.ParseInt(database_data.Total, 10, 64)

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_rounds(c *fiber.Ctx) error {

	// Variables
	var output v1XcashDpopsUnauthorizedRounds
	var database_data XcashDpopsDelegatesCollection
	var block_height int
	var count int
	var data []string
	var str string

	// setup database
	collection := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// get the resource
	if block_height, _ = strconv.Atoi(c.Params("blockHeight")); c.Params("blockHeight") == "" {
		error := ErrorResults{"Could not get the round details"}
		return c.JSON(error)
	}

	// get all of the public keys in the block
	s := get_reserve_bytes(block_height)
	if s == "" {
		error := ErrorResults{"Could not get the round details"}
		return c.JSON(error)
	}
	s = s[strings.Index(s, BLOCKCHAIN_DATA_SEGMENT_PUBLIC_ADDRESS_STRING_DATA)+len(BLOCKCHAIN_DATA_SEGMENT_PUBLIC_ADDRESS_STRING_DATA) : len(s)]

	for count = 0; count < BLOCK_VERIFIERS_AMOUNT; count++ {
		str = s[0 : PUBLIC_KEY_LENGTH*2]
		data5, _ := hex.DecodeString(str)
		str = string(data5)
		data = append(data, str)
		s = s[(PUBLIC_KEY_LENGTH*2)+len(BLOCKCHAIN_DATA_SEGMENT_PUBLIC_ADDRESS_STRING_DATA) : len(s)]
	}

	// convert the public keys to public addresses
	for _, item := range data {
		// get the delegate name
		err := collection.FindOne(ctx, bson.D{{"public_key", item}}).Decode(&database_data)
		if err == mongo.ErrNoDocuments {
			output.Delegates = append(output.Delegates, "DELEGATE_REMOVED")
			continue
		} else if err != nil {
			continue
		}
		output.Delegates = append(output.Delegates, database_data.DelegateName)
	}

	return c.JSON(output)
}

func v1_xcash_dpops_unauthorized_last_block_producer(c *fiber.Ctx) error {

	// Variables
	var output v1XcashDpopsUnauthorizedLastBlockProducer
	var block_height int

	// get the previous block Height
	if block_height = get_current_block_height(); block_height == 0 {
		error := ErrorResults{"Could not get the last block producer"}
		return c.JSON(error)
	}

	// fill in the data
	output.LastBlockProducer = get_block_delegate(block_height - 1)

	return c.JSON(output)
}
