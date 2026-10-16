package pipeline

import (
	"context"
	"os"

	"hermes-decompiler/pkg/database/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetHashesFromDatabaseFast returns the same groups as GetHashesFromDatabaseWithRNVersion but via a
// covered index scan (cover_rn_pkg_3hash) + grouping in Go, instead of $group/$lookup over full
// documents that carry the raw IR strings.
func GetHashesFromDatabaseFast(rnVersion string) ([]*HashGroup, error) {
	ctx := context.Background()
	dbName := os.Getenv("HEDIS_PACKAGES_DB")
	if dbName == "" {
		dbName = "clean"
	}
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_CONNECTION_STRING")))
	if err != nil {
		return nil, err
	}
	defer cl.Disconnect(ctx)
	db := cl.Database(dbName)
	var maxOID primitive.ObjectID
	if o, err := primitive.ObjectIDFromHex(os.Getenv("HEDIS_MAX_OID")); err == nil { // M35: DB pin, see loadNormKeys
		maxOID = o
	}
	rns := resolveRNs(ctx, db, rnVersion, maxOID) // M37 RN fallback
	if len(rns) == 0 {
		return nil, nil
	}
	hq := bson.M{"react_native_version": bson.M{"$in": rns}}
	if !maxOID.IsZero() {
		hq["_id"] = bson.M{"$lte": maxOID}
	}
	if os.Getenv("HEDIS_NO_SUB") != "" { // ablation: root-entry rows only
		hq["sub"] = bson.M{"$ne": true}
	}
	cur, err := db.Collection("hashes").Find(ctx, hq,
		options.Find().SetProjection(bson.M{"_id": 0, "package_id": 1, "react_native_version": 1, "structural.hash": 1, "ar1.hash": 1, "ar2.hash": 1}).
			SetHint("cover_rn_pkg_3hash").SetBatchSize(50000))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	groups := map[primitive.ObjectID]*HashGroup{}
	var order []primitive.ObjectID
	for cur.Next(ctx) {
		h := &models.FunctionObjectHashModel{}
		if err := cur.Decode(h); err != nil {
			return nil, err
		}
		g, ok := groups[h.PackageID]
		if !ok {
			g = &HashGroup{PackageID: h.PackageID}
			groups[h.PackageID] = g
			order = append(order, h.PackageID)
		}
		g.Hashes = append(g.Hashes, h)
	}
	if len(order) == 0 {
		return nil, nil
	}
	pc, err := db.Collection("packages").Find(ctx, bson.M{"_id": bson.M{"$in": order}})
	if err != nil {
		return nil, err
	}
	var ps []models.PackageWithVulnerabilitiesModel
	if err := pc.All(ctx, &ps); err != nil {
		return nil, err
	}
	out := make([]*HashGroup, 0, len(ps))
	for i := range ps {
		if g, ok := groups[ps[i].ID]; ok {
			g.Result = &ps[i]
			out = append(out, g)
		}
	}
	return out, nil
}
