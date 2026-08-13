package driver

import (
	"context"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
)

type GraphEngine interface {
	Name() string
	Connect(ctx context.Context) error
	Close(ctx context.Context) error

	CreateIndex(ctx context.Context) error
	BulkInsertNodes(ctx context.Context, nodes []dataset.User, batchSize int) (int64, error)
	BulkInsertRelationships(ctx context.Context, rels []dataset.Relationship, batchSize int) (int64, error)
	VerifyCounts(ctx context.Context) (int64, int64, error)

	OneHop(ctx context.Context, startNodeID int64) ([]int64, error)
	TwoHop(ctx context.Context, startNodeID int64) ([]int64, error)
	ThreeHop(ctx context.Context, startNodeID int64, limit int) ([]int64, error)
	PointLookup(ctx context.Context, nodeID int64) (*dataset.User, error)
	Aggregation(ctx context.Context) (int64, error)
	WriteRelationship(ctx context.Context, fromID, toID int64) error
	FilteredLookup(ctx context.Context, age int) ([]int64, error)
}