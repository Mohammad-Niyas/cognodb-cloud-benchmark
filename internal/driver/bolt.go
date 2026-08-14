package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type BoltEngine struct {
	name     string
	uri      string
	user     string
	password string
	driver   neo4j.DriverWithContext
}

func NewBoltEngine(name, uri, user, password string) *BoltEngine {
	return &BoltEngine{
		name:     name,
		uri:      uri,
		user:     user,
		password: password,
	}
}

func (b *BoltEngine) Name() string {
	return b.name
}

func (b *BoltEngine) Connect(ctx context.Context) error {
	auth := neo4j.BasicAuth(b.user, b.password, "")
	if b.user == "" && b.password == "" {
		auth = neo4j.NoAuth()
	}

	driver, err := neo4j.NewDriverWithContext(b.uri, auth)
	if err != nil {
		return fmt.Errorf("[%s] driver creation failed: %w", b.name, err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("[%s] connection verification failed: %w", b.name, err)
	}

	b.driver = driver
	fmt.Printf("[%s] Connected successfully to %s\n", b.name, b.uri)
	return nil
}

func (b *BoltEngine) Close(ctx context.Context) error {
	if b.driver != nil {
		return b.driver.Close(ctx)
	}
	return nil
}

func (b *BoltEngine) CreateIndex(ctx context.Context) error {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	engineLower := strings.ToLower(b.name)
	var query string

	switch {
	case strings.Contains(engineLower, "memgraph"):
		query = "CREATE INDEX ON :User(id);"
	case strings.Contains(engineLower, "falkordb"):
		query = "CREATE INDEX FOR (u:User) ON (u.id);"
	default:
		query = "CREATE INDEX user_id_idx IF NOT EXISTS FOR (u:User) ON (u.id)"
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, nil)
		return nil, err
	})

	if err != nil && (strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "IndexAlreadyExists")) {
		return nil
	}
	return err
}

func (b *BoltEngine) BulkInsertNodes(ctx context.Context, nodes []dataset.User, batchSize int) (int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	var totalInserted int64

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := make([]map[string]any, 0, end-i)
		for _, u := range nodes[i:end] {
			batch = append(batch, map[string]any{
				"id":     u.ID,
				"age":    u.Age,
				"gender": u.Gender,
			})
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				UNWIND $batch AS row
				CREATE (u:User {id: row.id, age: row.age, gender: row.gender})
			`
			return tx.Run(ctx, query, map[string]any{"batch": batch})
		})
		if err != nil {
			return totalInserted, fmt.Errorf("[%s] node batch failed at %d: %w", b.name, i, err)
		}

		totalInserted += int64(len(batch))
	}

	return totalInserted, nil
}

func (b *BoltEngine) BulkInsertRelationships(ctx context.Context, rels []dataset.Relationship, batchSize int) (int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	var totalInserted int64

	for i := 0; i < len(rels); i += batchSize {
		end := i + batchSize
		if end > len(rels) {
			end = len(rels)
		}

		batch := make([]map[string]any, 0, end-i)
		for _, r := range rels[i:end] {
			batch = append(batch, map[string]any{
				"from": r.FromUser,
				"to":   r.ToUser,
			})
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				UNWIND $batch AS row
				MATCH (a:User {id: row.from})
				MATCH (b:User {id: row.to})
				CREATE (a)-[:FRIEND]->(b)
			`
			return tx.Run(ctx, query, map[string]any{"batch": batch})
		})
		if err != nil {
			return totalInserted, fmt.Errorf("[%s] rel batch failed at %d: %w", b.name, i, err)
		}

		totalInserted += int64(len(batch))
	}

	return totalInserted, nil
}

func (b *BoltEngine) VerifyCounts(ctx context.Context) (int64, int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)
	var nodeCount, relCount int64
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res1, err := tx.Run(ctx, "MATCH (u:User) RETURN count(u) AS count", nil)
		if err != nil {
			return nil, err
		}
		if res1.Next(ctx) {
			if val, ok := res1.Record().Get("count"); ok {
				if c, ok := val.(int64); ok {
					nodeCount = c
				}
			}
		}
		res2, err := tx.Run(ctx, "MATCH ()-[r:FRIEND]->() RETURN count(r) AS count", nil)
		if err != nil {
			return nil, err
		}
		if res2.Next(ctx) {
			if val, ok := res2.Record().Get("count"); ok {
				if c, ok := val.(int64); ok {
					relCount = c
				}
			}
		}
		return nil, nil
	})
	return nodeCount, relCount, err
}

func (b *BoltEngine) OneHop(ctx context.Context, startNodeID int64) ([]int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "MATCH (u:User {id: $id})-[:FRIEND]->(f:User) RETURN f.id AS friend_id"
		result, err := tx.Run(ctx, query, map[string]any{"id": startNodeID})
		if err != nil {
			return nil, err
		}

		var friendIDs []int64
		for result.Next(ctx) {
			if id, ok := result.Record().Get("friend_id"); ok {
				if idInt, ok := id.(int64); ok {
					friendIDs = append(friendIDs, idInt)
				}
			}
		}
		return friendIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]int64), nil
}

func (b *BoltEngine) TwoHop(ctx context.Context, startNodeID int64) ([]int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "MATCH (u:User {id: $id})-[:FRIEND]->()-[:FRIEND]->(fof:User) RETURN DISTINCT fof.id AS fof_id"
		result, err := tx.Run(ctx, query, map[string]any{"id": startNodeID})
		if err != nil {
			return nil, err
		}

		var fofIDs []int64
		for result.Next(ctx) {
			if id, ok := result.Record().Get("fof_id"); ok {
				if idInt, ok := id.(int64); ok {
					fofIDs = append(fofIDs, idInt)
				}
			}
		}
		return fofIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]int64), nil
}

func (b *BoltEngine) ThreeHop(ctx context.Context, startNodeID int64, limit int) ([]int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "MATCH (u:User {id: $id})-[:FRIEND*3]->(f3:User) RETURN DISTINCT f3.id AS f3_id LIMIT $limit"
		result, err := tx.Run(ctx, query, map[string]any{"id": startNodeID, "limit": limit})
		if err != nil {
			return nil, err
		}

		var f3IDs []int64
		for result.Next(ctx) {
			if id, ok := result.Record().Get("f3_id"); ok {
				if idInt, ok := id.(int64); ok {
					f3IDs = append(f3IDs, idInt)
				}
			}
		}
		return f3IDs, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]int64), nil
}

func (b *BoltEngine) PointLookup(ctx context.Context, nodeID int64) (*dataset.User, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "MATCH (u:User {id: $id}) RETURN u.id AS id, u.age AS age, u.gender AS gender"
		result, err := tx.Run(ctx, query, map[string]any{"id": nodeID})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			rec := result.Record()
			id, _ := rec.Get("id")
			age, _ := rec.Get("age")
			gender, _ := rec.Get("gender")

			user := &dataset.User{
				ID:     id.(int64),
				Age:    int(age.(int64)),
				Gender: gender.(string),
			}
			return user, nil
		}
		return nil, fmt.Errorf("user not found: %d", nodeID)
	})
	if err != nil {
		return nil, err
	}
	return res.(*dataset.User), nil
}

func (b *BoltEngine) Aggregation(ctx context.Context) (int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (u:User)-[r:FRIEND]->()
			RETURN u.id AS user_id, count(r) AS friend_count
			ORDER BY friend_count DESC
			LIMIT 10
		`
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return int64(0), err
		}

		var count int64
		for result.Next(ctx) {
			count++
		}
		return count, nil
	})
	if err != nil {
		return 0, err
	}
	return res.(int64), nil
}

// WriteRelationship creates/updates interaction edge without creating duplicate FRIEND topologies
func (b *BoltEngine) WriteRelationship(ctx context.Context, fromID, toID int64) error {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (a:User {id: $from}), (b:User {id: $to})
			MERGE (a)-[r:INTERACTION]->(b)
			SET r.last_updated = $ts
		`
		return tx.Run(ctx, query, map[string]any{"from": fromID, "to": toID, "ts": time.Now().UnixNano()})
	})
	return err
}

func (b *BoltEngine) FilteredLookup(ctx context.Context, age int) ([]int64, error) {
	session := b.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := "MATCH (u:User) WHERE u.age = $age RETURN u.id AS id LIMIT 100"
		result, err := tx.Run(ctx, query, map[string]any{"age": age})
		if err != nil {
			return nil, err
		}

		var ids []int64
		for result.Next(ctx) {
			if id, ok := result.Record().Get("id"); ok {
				if idInt, ok := id.(int64); ok {
					ids = append(ids, idInt)
				}
			}
		}
		return ids, nil
	})
	if err != nil {
		return nil, err
	}
	return res.([]int64), nil
}
