package dataset

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// CanonicalData holds the preloaded users and relationships from CSV
type CanonicalData struct {
	Users         []User
	UserIDs       []int64
	Relationships []Relationship
}

// LoadCanonical reads nodes.csv and relationships.csv into memory
func LoadCanonical(nodesPath, relsPath string) (*CanonicalData, error) {
	users, ids, err := loadNodesCSV(nodesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", nodesPath, err)
	}

	rels, err := loadRelationshipsCSV(relsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", relsPath, err)
	}

	return &CanonicalData{
		Users:         users,
		UserIDs:       ids,
		Relationships: rels,
	}, nil
}

func loadNodesCSV(path string) ([]User, []int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	var users []User
	var ids []int64

	for i, row := range records {
		if i == 0 || len(row) < 3 {
			continue
		}
		id, _ := strconv.ParseInt(row[0], 10, 64)
		age, _ := strconv.Atoi(row[1])
		users = append(users, User{
			ID:     id,
			Age:    age,
			Gender: row[2],
		})
		ids = append(ids, id)
	}
	return users, ids, nil
}

func loadRelationshipsCSV(path string) ([]Relationship, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var rels []Relationship
	for i, row := range records {
		if i == 0 || len(row) < 2 {
			continue
		}
		from, _ := strconv.ParseInt(row[0], 10, 64)
		to, _ := strconv.ParseInt(row[1], 10, 64)
		rels = append(rels, Relationship{
			FromUser: from,
			ToUser:   to,
		})
	}
	return rels, nil
}