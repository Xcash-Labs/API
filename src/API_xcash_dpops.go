package main

import (
	"strings"
	"strconv"
	"sort"
	"context"
    "errors" 
	"time"
    "fmt"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Returns (publicAddress, publicKey, error). publicKey is what you use to read statistics (_id == publicKey).
func getDelegateKeysFromName(ctx context.Context, delegateName string) (string, string, error) {
	if mongoClient == nil {
		return "", "", fmt.Errorf("database unavailable")
	}
	col := mongoClient.Database(XCASH_DPOPS_DATABASE).Collection("delegates")

	// Only fetch the fields we need
	proj := bson.D{
		{Key: "_id", Value: 0},
		{Key: "public_address", Value: 1},
		{Key: "public_key", Value: 1},
	}
	opts := options.FindOne().SetProjection(proj)

	var doc bson.M
	if err := col.FindOne(ctx, bson.D{{Key: "delegate_name", Value: delegateName}}, opts).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", "", fmt.Errorf("delegate not found")
		}
		return "", "", err
	}

	addr := asString(doc["public_address"])
	key  := asString(doc["public_key"])
	if key == "" {
		return "", "", fmt.Errorf("delegate missing public_key")
	}
	return addr, key, nil
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

func v2_xcash_dpops_unauthorized_delegates(c *fiber.Ctx) error {
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	delegateName := c.Params("delegateName")
	if strings.TrimSpace(delegateName) == "" {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := mongoClient.Database(XCASH_DPOPS_DATABASE)
	colDelegates := db.Collection("delegates")
	colProofs := db.Collection("reserve_proofs")
	colStats := db.Collection("statistics")

	// 1) Load the requested delegate (need public_address + public_key + profile fields)
	var d bson.M
	delegateProj := options.FindOne().SetProjection(bson.D{
		{Key: "_id", Value: 0},
		{Key: "delegate_name", Value: 1},
		{Key: "delegate_type", Value: 1},
		{Key: "public_address", Value: 1},
		{Key: "public_key", Value: 1},
		{Key: "IP_address", Value: 1},
		{Key: "online_status", Value: 1},
		{Key: "delegate_fee", Value: 1},
		{Key: "about", Value: 1},
		{Key: "website", Value: 1},
		{Key: "team", Value: 1},
		{Key: "specifications", Value: 1},
		{Key: "total_vote_count", Value: 1},
	})
	if err := colDelegates.FindOne(
		ctx,
		bson.D{{Key: "delegate_name", Value: delegateName}},
		delegateProj,
	).Decode(&d); err != nil {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	pubAddr := asString(d["public_address"]) // for proofs + response
	pubKey  := asString(d["public_key"])     // for statistics (_id == public_key)
	if pubAddr == "" || pubKey == "" {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	// 2) STRICT: stats must exist (_id == public_key)
	var s bson.M
	statsProj := options.FindOne().SetProjection(bson.D{
		{Key: "_id", Value: 1},
		{Key: "block_verifier_total_rounds", Value: 1},
		{Key: "block_verifier_online_total_rounds", Value: 1},
		{Key: "block_producer_total_rounds", Value: 1},
		{Key: "last_counted_block", Value: 1},
	})
	if err := colStats.FindOne(ctx, bson.D{{Key: "_id", Value: pubKey}}, statsProj).Decode(&s); err != nil {
		return c.JSON(ErrorResults{fmt.Sprintf("Statistics data not found for delegate: %s", delegateName)})
	}

	// 3) Aggregate voters and summed votes for this delegate (by public_address)
	totalVoters := 0
	totalVotes := int64(0)
	{
		p := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"public_address_voted_for": pubAddr}}},
			{{
				Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$public_address_voted_for"},
					{Key: "voters", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "sumVotes", Value: bson.D{{Key: "$sum", Value: "$total_vote"}}},
				},
			}},
			{{Key: "$limit", Value: 1}},
		}
		if cur, err := colProofs.Aggregate(ctx, p); err == nil {
			var rows []bson.M
			_ = cur.All(ctx, &rows)
			_ = cur.Close(ctx)
			if len(rows) == 1 {
				totalVoters = int(toInt64(rows[0]["voters"]))
				totalVotes  = toInt64(rows[0]["sumVotes"])
			}
		}
		// fallback to stored delegate total if no agg rows
		if totalVotes == 0 {
			totalVotes = toInt64(d["total_vote_count"])
		}
	}

	// 4) Rank by votes across all delegates (desc)
	type row struct{ Name string; Votes int64 }
	var all []row
	{
		proj := bson.D{
			{Key: "_id", Value: 0},
			{Key: "delegate_name", Value: 1},
			{Key: "total_vote_count", Value: 1},
		}
		cur, err := colDelegates.Find(ctx, bson.D{}, options.Find().SetProjection(proj))
		if err != nil {
			return c.JSON(ErrorResults{"Could not get the delegates data"})
		}
		var docs []bson.M
		if err := cur.All(ctx, &docs); err != nil {
			return c.JSON(ErrorResults{"Could not get the delegates data"})
		}
		for _, it := range docs {
			all = append(all, row{
				Name:  asString(it["delegate_name"]),
				Votes: toInt64(it["total_vote_count"]),
			})
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].Votes > all[j].Votes })
	}

	rank := 0
	for i := range all {
		if all[i].Name == delegateName {
			rank = i + 1
			break
		}
	}

	// 5) Build v2 output
	var out v2XcashDpopsUnauthorizedDelegatesData

	// online
	switch v := d["online_status"].(type) {
	case bool:
		out.Online = v
	case string:
		out.Online = strings.EqualFold(v, "true")
	}

	// fill fields
	out.Votes         = totalVotes
	out.Voters        = totalVoters
	out.IPAdress      = asString(d["IP_address"])
	out.DelegateName  = asString(d["delegate_name"])
	out.PublicAddress = pubAddr
	out.About         = asString(d["about"])
	out.Website       = asString(d["website"])
	out.Team          = asString(d["team"])
	out.Specifications = asString(d["specifications"]) // or "server_specs" if that’s your field
	out.DelegateType  = asString(d["delegate_type"])
	out.Fee           = int(toInt64(d["delegate_fee"]))

	verifierTotal  := toInt64(s["block_verifier_total_rounds"])
	verifierOnline := toInt64(s["block_verifier_online_total_rounds"])
	producerTotal  := toInt64(s["block_producer_total_rounds"])

	out.TotalRounds              = int(verifierTotal)
	out.TotalBlockProducerRounds = int(producerTotal)
	if verifierTotal > 0 {
		out.OnlinePercentage = int((verifierOnline * 100) / verifierTotal)
	}
	out.Rank = rank

	return c.JSON(out)
}

func v2_xcash_dpops_unauthorized_delegates_votes(c *fiber.Ctx) error {
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	delegateName := c.Params("delegateName")
	if strings.TrimSpace(delegateName) == "" {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := mongoClient.Database(XCASH_DPOPS_DATABASE)
	colProofs := db.Collection("reserve_proofs")

	// Use your new helper to get the delegate's wallet address (and key if you want)
	// addr = public_address to match public_address_voted_for
	addr, _, err := getDelegateKeysFromName(ctx, delegateName)
	if err != nil || addr == "" {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	// One vote per voter → no $group needed.
	// IMPORTANT: voter address may be in `_id` (preferred) or sometimes `public_address`.
	// Use $ifNull to fall back: publicAddress = public_address ?? _id
	type aggRow struct {
		PublicAddress string      `bson:"publicAddress"`
		Amount        interface{} `bson:"amount"`
	}
	var rows []aggRow

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"public_address_voted_for": addr}}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "publicAddress", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$public_address", "$_id"}}}},
			{Key: "amount",        Value: "$total_vote"},
		}}},
		// drop any odd empties just in case
		{{Key: "$match", Value: bson.M{"publicAddress": bson.M{"$ne": ""}}}},
		{{Key: "$sort",  Value: bson.D{{Key: "amount", Value: -1}}}},
	}

	if cur, err := colProofs.Aggregate(ctx, pipeline); err == nil {
		_ = cur.All(ctx, &rows)
		_ = cur.Close(ctx)
	} else {
		return c.JSON(ErrorResults{"Could not get the delegates data"})
	}

	out := make([]v2XcashDpopsUnauthorizedDelegatesVotes, 0, len(rows))
	for _, r := range rows {
		out = append(out, v2XcashDpopsUnauthorizedDelegatesVotes{
			PublicAddress: r.PublicAddress,
			Amount:        toInt64(r.Amount),
		})
	}

	return c.JSON(out)
}

func v2_xcash_dpops_unauthorized_votes(c *fiber.Ctx) error {
	if mongoClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database unavailable"})
	}

	addr := strings.TrimSpace(c.Params("address"))
	if addr == "" {
		return c.JSON(ErrorResults{"Could not get the vote details"})
	}
	// Optional: keep your old format checks if you want
	// if !strings.HasPrefix(addr, XCASH_WALLET_PREFIX) || len(addr) != XCASH_WALLET_LENGTH {
	// 	return c.JSON(ErrorResults{"Could not get the vote details"})
	// }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := mongoClient.Database(XCASH_DPOPS_DATABASE)
	colProofs := db.Collection("reserve_proofs")
	colDelegates := db.Collection("delegates")

	// --- 1) Find this voter's proof ---
	// In the new system, the voter may be in `_id` (preferred) or `public_address`.
	// We match either to be safe.
	var proof bson.M
	if err := colProofs.FindOne(
		ctx,
		bson.M{
			"$or": bson.A{
				bson.M{"_id": addr},
				bson.M{"public_address": addr},
			},
		},
		options.FindOne().SetProjection(bson.D{
			{Key: "_id", Value: 0},
			{Key: "public_address_voted_for", Value: 1},
			{Key: "total_vote", Value: 1},
		}),
	).Decode(&proof); err != nil {
		// Not found = this address hasn't voted
		return c.JSON(ErrorResults{"This address has not voted"})
	}

	delegateAddr := asString(proof["public_address_voted_for"])
	if delegateAddr == "" {
		return c.JSON(ErrorResults{"This address has not voted"})
	}
	amount := toInt64(proof["total_vote"])

	// --- 2) Resolve delegate name from delegates.public_address ---
	var d bson.M
	if err := colDelegates.FindOne(
		ctx,
		bson.D{{Key: "public_address", Value: delegateAddr}},
		options.FindOne().SetProjection(bson.D{
			{Key: "_id", Value: 0},
			{Key: "delegate_name", Value: 1},
		}),
	).Decode(&d); err != nil {
		// If the delegate doc is missing (shouldn't happen), keep a generic error
		return c.JSON(ErrorResults{"Could not get the vote details"})
	}
	delegateName := asString(d["delegate_name"])
	if delegateName == "" {
		return c.JSON(ErrorResults{"Could not get the vote details"})
	}

	// --- 3) Shape response (new type) ---
	out := v2XcashDpopsUnauthorizedVotes{
		DelegateName: delegateName,
		Amount:       amount,
	}
	return c.JSON(out)
}