package main

import (
	"strings"
	"strconv"
	"sort"
	"context"
//	"encoding/hex"
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
	colStats := db.Collection("statistics")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1) Load delegates (include public_key for joining to statistics._id)
	proj := bson.D{
		{Key: "_id", Value: 0},
		{Key: "public_address", Value: 1},
		{Key: "public_key", Value: 1}, // <-- needed to join to statistics._id
		{Key: "delegate_name", Value: 1},
		{Key: "delegate_type", Value: 1},
		{Key: "IP_address", Value: 1},
		{Key: "online_status", Value: 1},
		{Key: "delegate_fee", Value: 1},

		// Fallbacks (only used if statistics doc missing—though we now hard-fail)
		{Key: "block_verifier_total_rounds", Value: 1},
		{Key: "block_producer_total_rounds", Value: 1},
		{Key: "block_verifier_online_percentage", Value: 1},

		{Key: "total_vote_count", Value: 1}, // prefer this if present
	}
	cur, err := colDelegates.Find(ctx, bson.D{}, options.Find().SetProjection(proj))
	if err != nil {
		return c.JSON(ErrorResults{"Could not get the delegates registered"})
	}
	var delegates []bson.M
	if err := cur.All(ctx, &delegates); err != nil {
		return c.JSON(ErrorResults{"Could not get the delegates registered"})
	}

	// Build IN lists for proofs & stats
	pubAddrs := make([]string, 0, len(delegates))
	pubKeys := make([]string, 0, len(delegates))
	for _, d := range delegates {
		if pa := asString(d["public_address"]); pa != "" {
			pubAddrs = append(pubAddrs, pa)
		}
		if pk := asString(d["public_key"]); pk != "" {
			pubKeys = append(pubKeys, pk)
		}
	}

	// 2) Aggregate voters/sumVotes from reserve_proofs
	type aggRow struct {
		ID       string      `bson:"_id"`
		Voters   int32       `bson:"voters"`
		SumVotes interface{} `bson:"sumVotes"`
	}
	voterMap := make(map[string]struct {
		voters   int
		sumVotes int64
	}, len(delegates))

	if len(pubAddrs) > 0 {
		pipe := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"public_address_voted_for": bson.M{"$in": pubAddrs}}}},
			{{
				Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$public_address_voted_for"},
					{Key: "voters", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "sumVotes", Value: bson.D{{Key: "$sum", Value: "$total_vote"}}},
				},
			}},
		}
		if aCur, err := colProofs.Aggregate(ctx, pipe); err == nil {
			var agg []aggRow
			_ = aCur.All(ctx, &agg)
			_ = aCur.Close(ctx)
			for _, r := range agg {
				voterMap[r.ID] = struct {
					voters   int
					sumVotes int64
				}{
					voters:   int(r.Voters),
					sumVotes: toInt64(r.SumVotes),
				}
			}
		}
	}

	// 3) Load statistics by _id (which equals delegates.public_key)
	// Expected stats fields:
	//   _id: <public_key>
	//   block_verifier_total_rounds
	//   block_verifier_online_total_rounds
	//   block_producer_total_rounds
	//   last_counted_block
	statsMap := make(map[string]bson.M, len(delegates))
	if len(pubKeys) > 0 {
		statsCur, err := colStats.Find(ctx, bson.M{"_id": bson.M{"$in": pubKeys}},
			options.Find().SetProjection(bson.D{
				{Key: "_id", Value: 1}, // keep for lookup
				{Key: "block_verifier_total_rounds", Value: 1},
				{Key: "block_verifier_online_total_rounds", Value: 1},
				{Key: "block_producer_total_rounds", Value: 1},
				{Key: "last_counted_block", Value: 1},
			}))
		if err == nil {
			var statsDocs []bson.M
			if err := statsCur.All(ctx, &statsDocs); err == nil {
				for _, s := range statsDocs {
					if id := asString(s["_id"]); id != "" {
						statsMap[id] = s
					}
				}
			}
		}
	}

	// 4) Build output (hard-fail if any delegate lacks statistics)
	out := make([]*v2XcashDpopsUnauthorizedDelegatesBasicData, 0, len(delegates))
	typesForSort := make([]string, 0, len(delegates))

	for _, it := range delegates {
		pubAddr := asString(it["public_address"])
		pubKey := asString(it["public_key"])

		// Require statistics for this delegate
		s, ok := statsMap[pubKey]
		if !ok {
			return c.JSON(ErrorResults{"Statistics data not found for delegate"})
		}

		row := new(v2XcashDpopsUnauthorizedDelegatesBasicData)
		row.DelegateName = asString(it["delegate_name"])
		row.IPAdress = asString(it["IP_address"])
		row.DelegateType = asString(it["delegate_type"])

		switch v := it["online_status"].(type) {
		case bool:
			row.Online = v
		case string:
			row.Online = strings.EqualFold(v, "true")
		default:
			row.Online = false
		}

		row.Fee = int(toInt64(it["delegate_fee"]))

		// Votes
		votes := toInt64(it["total_vote_count"])
		if votes == 0 {
			if v, ok := voterMap[pubAddr]; ok {
				votes = v.sumVotes
			}
		}
		row.Votes = votes

		// Voters
		if v, ok := voterMap[pubAddr]; ok {
			row.Voters = v.voters
		} else {
			row.Voters = 0
		}

		// Rounds & Online %
		verifierTotal := toInt64(s["block_verifier_total_rounds"])
		verifierOnline := toInt64(s["block_verifier_online_total_rounds"])
		producerTotal := toInt64(s["block_producer_total_rounds"])

		row.TotalRounds = int(verifierTotal)
		row.TotalBlockProducerRounds = int(producerTotal)
		if verifierTotal > 0 {
			row.OnlinePercentage = int((verifierOnline * 100) / verifierTotal)
		} else {
			row.OnlinePercentage = 0
		}

		out = append(out, row)
		typesForSort = append(typesForSort, row.DelegateType)
	}

	// 5) Sort: delegate_type asc, Online true first, Votes desc
	sort.SliceStable(out, func(i, j int) bool {
		if typesForSort[i] != typesForSort[j] {
			return typesForSort[i] < typesForSort[j]
		}
		if out[i].Online != out[j].Online {
			return out[i].Online // true first
		}
		return out[i].Votes > out[j].Votes
	})

	return c.JSON(out)
}