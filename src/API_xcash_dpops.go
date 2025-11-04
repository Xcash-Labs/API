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




// Routes



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