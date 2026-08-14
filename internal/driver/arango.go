package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
)

type ArangoEngine struct {
	name     string
	uri      string
	user     string
	password string
	database string
	client   *http.Client
}

func NewArangoEngine(name, uri, user, password, database string) *ArangoEngine {
	if database == "" {
		database = "_system"
	}
	return &ArangoEngine{
		name:     name,
		uri:      uri,
		user:     user,
		password: password,
		database: database,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (a *ArangoEngine) Name() string {
	return a.name
}

func (a *ArangoEngine) Connect(ctx context.Context) error {
	url := fmt.Sprintf("%s/_api/version", a.uri)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.user, a.password)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("[%s] HTTP connection failed: %w", a.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[%s] auth failed with HTTP status: %d", a.name, resp.StatusCode)
	}

	fmt.Printf("[%s] Connected successfully to %s\n", a.name, a.uri)
	return nil
}

func (a *ArangoEngine) Close(ctx context.Context) error {
	return nil
}

func (a *ArangoEngine) CreateIndex(ctx context.Context) error {
	if err := a.createCollection(ctx, "User", 2); err != nil {
		return fmt.Errorf("create User collection: %w", err)
	}
	if err := a.createCollection(ctx, "FRIEND", 3); err != nil {
		return fmt.Errorf("create FRIEND collection: %w", err)
	}

	url := fmt.Sprintf("%s/_db/%s/_api/index?collection=User", a.uri, a.database)
	payload := map[string]any{
		"type":   "persistent",
		"fields": []string{"id"},
		"unique": true,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.user, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create index failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

func (a *ArangoEngine) createCollection(ctx context.Context, name string, colType int) error {
	url := fmt.Sprintf("%s/_db/%s/_api/collection", a.uri, a.database)
	payload := map[string]any{
		"name": name,
		"type": colType,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.user, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("collection creation for %s failed with status %d: %s", name, resp.StatusCode, string(respBytes))
	}
	return nil
}

func (a *ArangoEngine) BulkInsertNodes(ctx context.Context, nodes []dataset.User, batchSize int) (int64, error) {
	var totalInserted int64
	url := fmt.Sprintf("%s/_db/%s/_api/document/User", a.uri, a.database)

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := make([]map[string]any, 0, end-i)
		for _, u := range nodes[i:end] {
			batch = append(batch, map[string]any{
				"_key":   fmt.Sprintf("%d", u.ID),
				"id":     u.ID,
				"age":    u.Age,
				"gender": u.Gender,
			})
		}

		body, err := json.Marshal(batch)
		if err != nil {
			return totalInserted, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return totalInserted, err
		}
		req.SetBasicAuth(a.user, a.password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			return totalInserted, fmt.Errorf("[%s] node batch failed at %d: %w", a.name, i, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
			respBytes, _ := io.ReadAll(resp.Body)
			return totalInserted, fmt.Errorf("[%s] node batch status %d: %s", a.name, resp.StatusCode, string(respBytes))
		}

		totalInserted += int64(len(batch))
	}
	return totalInserted, nil
}

func (a *ArangoEngine) BulkInsertRelationships(ctx context.Context, rels []dataset.Relationship, batchSize int) (int64, error) {
	var totalInserted int64
	url := fmt.Sprintf("%s/_db/%s/_api/document/FRIEND", a.uri, a.database)

	for i := 0; i < len(rels); i += batchSize {
		end := i + batchSize
		if end > len(rels) {
			end = len(rels)
		}

		batch := make([]map[string]any, 0, end-i)
		for _, r := range rels[i:end] {
			batch = append(batch, map[string]any{
				"_from": fmt.Sprintf("User/%d", r.FromUser),
				"_to":   fmt.Sprintf("User/%d", r.ToUser),
			})
		}

		body, err := json.Marshal(batch)
		if err != nil {
			return totalInserted, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			return totalInserted, err
		}
		req.SetBasicAuth(a.user, a.password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			return totalInserted, fmt.Errorf("[%s] rel batch failed at %d: %w", a.name, i, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
			respBytes, _ := io.ReadAll(resp.Body)
			return totalInserted, fmt.Errorf("[%s] rel batch status %d: %s", a.name, resp.StatusCode, string(respBytes))
		}

		totalInserted += int64(len(batch))
	}
	return totalInserted, nil
}

func (a *ArangoEngine) VerifyCounts(ctx context.Context) (int64, int64, error) {
	nodeCount, err := a.runCountQuery(ctx, "FOR u IN User COLLECT WITH COUNT INTO length RETURN length")
	if err != nil {
		return 0, 0, err
	}
	relCount, err := a.runCountQuery(ctx, "FOR r IN FRIEND COLLECT WITH COUNT INTO length RETURN length")
	if err != nil {
		return nodeCount, 0, err
	}
	return nodeCount, relCount, nil
}

func (a *ArangoEngine) runCountQuery(ctx context.Context, aql string) (int64, error) {
	res, err := a.runAQL(ctx, aql, nil)
	if err != nil {
		return 0, err
	}
	if len(res) > 0 {
		if countFloat, ok := res[0].(float64); ok {
			return int64(countFloat), nil
		}
	}
	return 0, nil
}

// runAQL executes query and handles cursor pagination & error responses
func (a *ArangoEngine) runAQL(ctx context.Context, query string, bindVars map[string]any) ([]any, error) {
	url := fmt.Sprintf("%s/_db/%s/_api/cursor", a.uri, a.database)
	payload := map[string]any{
		"query":     query,
		"batchSize": 10000,
	}
	if bindVars != nil {
		payload["bindVars"] = bindVars
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.user, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AQL failed with HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var cursorResp struct {
		Result   []any  `json:"result"`
		HasMore  bool   `json:"hasMore"`
		ID       string `json:"id"`
		Error    bool   `json:"error"`
		ErrorNum int    `json:"errorNum"`
		ErrMsg   string `json:"errorMessage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&cursorResp); err != nil {
		return nil, fmt.Errorf("decode cursor error: %w", err)
	}

	if cursorResp.Error {
		return nil, fmt.Errorf("AQL error (%d): %s", cursorResp.ErrorNum, cursorResp.ErrMsg)
	}

	allResults := cursorResp.Result

	// Follow pagination if hasMore is true
	for cursorResp.HasMore && cursorResp.ID != "" {
		nextURL := fmt.Sprintf("%s/_db/%s/_api/cursor/%s", a.uri, a.database, cursorResp.ID)
		nextReq, err := http.NewRequestWithContext(ctx, "PUT", nextURL, nil)
		if err != nil {
			break
		}
		nextReq.SetBasicAuth(a.user, a.password)

		nextResp, err := a.client.Do(nextReq)
		if err != nil {
			break
		}

		if err := json.NewDecoder(nextResp.Body).Decode(&cursorResp); err != nil {
			nextResp.Body.Close()
			break
		}
		nextResp.Body.Close()

		allResults = append(allResults, cursorResp.Result...)
	}

	return allResults, nil
}

func (a *ArangoEngine) OneHop(ctx context.Context, startNodeID int64) ([]int64, error) {
	query := `FOR v IN 1..1 OUTBOUND CONCAT('User/', @id) FRIEND RETURN v.id`
	res, err := a.runAQL(ctx, query, map[string]any{"id": fmt.Sprintf("%d", startNodeID)})
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, item := range res {
		if f, ok := item.(float64); ok {
			ids = append(ids, int64(f))
		}
	}
	return ids, nil
}

func (a *ArangoEngine) TwoHop(ctx context.Context, startNodeID int64) ([]int64, error) {
	query := `FOR v IN 2..2 OUTBOUND CONCAT('User/', @id) FRIEND RETURN DISTINCT v.id`
	res, err := a.runAQL(ctx, query, map[string]any{"id": fmt.Sprintf("%d", startNodeID)})
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, item := range res {
		if f, ok := item.(float64); ok {
			ids = append(ids, int64(f))
		}
	}
	return ids, nil
}

func (a *ArangoEngine) ThreeHop(ctx context.Context, startNodeID int64, limit int) ([]int64, error) {
	query := `FOR v IN 3..3 OUTBOUND CONCAT('User/', @id) FRIEND LIMIT @limit RETURN DISTINCT v.id`
	res, err := a.runAQL(ctx, query, map[string]any{"id": fmt.Sprintf("%d", startNodeID), "limit": limit})
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, item := range res {
		if f, ok := item.(float64); ok {
			ids = append(ids, int64(f))
		}
	}
	return ids, nil
}

func (a *ArangoEngine) PointLookup(ctx context.Context, nodeID int64) (*dataset.User, error) {
	query := `FOR u IN User FILTER u.id == @id RETURN {id: u.id, age: u.age, gender: u.gender}`
	res, err := a.runAQL(ctx, query, map[string]any{"id": nodeID})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("user not found: %d", nodeID)
	}

	m, ok := res[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid record format")
	}

	var id, age int64
	if v, ok := m["id"].(float64); ok {
		id = int64(v)
	}
	if v, ok := m["age"].(float64); ok {
		age = int64(v)
	}
	gender, _ := m["gender"].(string)

	return &dataset.User{
		ID:     id,
		Age:    int(age),
		Gender: gender,
	}, nil
}

func (a *ArangoEngine) Aggregation(ctx context.Context) (int64, error) {
	query := `
		FOR r IN FRIEND
		COLLECT user_id = r._from WITH COUNT INTO friend_count
		SORT friend_count DESC
		LIMIT 10
		RETURN { user_id: user_id, friend_count: friend_count }
	`
	res, err := a.runAQL(ctx, query, nil)
	if err != nil {
		return 0, err
	}
	return int64(len(res)), nil
}

func (a *ArangoEngine) WriteRelationship(ctx context.Context, fromID, toID int64) error {
	edge := map[string]any{
		"_from":     fmt.Sprintf("User/%d", fromID),
		"_to":       fmt.Sprintf("User/%d", toID),
		"timestamp": time.Now().UnixNano(),
	}
	body, err := json.Marshal(edge)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/_db/%s/_api/document/FRIEND", a.uri, a.database)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(a.user, a.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("edge write failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (a *ArangoEngine) FilteredLookup(ctx context.Context, age int) ([]int64, error) {
	query := `FOR u IN User FILTER u.age == @age LIMIT 100 RETURN u.id`
	res, err := a.runAQL(ctx, query, map[string]any{"age": age})
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, val := range res {
		switch v := val.(type) {
		case float64:
			ids = append(ids, int64(v))
		case string:
			id, _ := strconv.ParseInt(v, 10, 64)
			ids = append(ids, id)
		}
	}
	return ids, nil
}
