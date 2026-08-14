package dataset

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type User struct {
	ID     int64  `json:"id"`
	Age    int    `json:"age"`
	Gender string `json:"gender"`
}

type Relationship struct {
	FromUser int64 `json:"from_user"`
	ToUser   int64 `json:"to_user"`
}

type SampledGraph struct {
	Users         []User
	Relationships []Relationship
}

// SampleBFSSubgraph extracts a connected subgraph of targetNodes using BFS
func SampleBFSSubgraph(relPath, profilePath string, targetNodes int, startNodeID int64) (*SampledGraph, error) {
	fmt.Printf("[Sampler] Phase 1: Reading relationships from %s...\n", relPath)

	file, err := os.Open(relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open relationships file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for relationships: %w", err)
	}
	defer gzReader.Close()

	// Build adjacency list
	adjList := make(map[int64][]int64)
	scanner := bufio.NewScanner(gzReader)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		if lineCount%5000000 == 0 {
			fmt.Printf("processed %d million edges into adjacency map\n", lineCount/1000000)
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		fromID, err1 := strconv.ParseInt(fields[0], 10, 64)
		toID, err2 := strconv.ParseInt(fields[1], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		adjList[fromID] = append(adjList[fromID], toID)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading relationships: %w", err)
	}

	fmt.Printf("[Sampler] Phase 2: Running BFS Snowball traversal starting at User %d...\n", startNodeID)

	visited := make(map[int64]bool)
	queue := []int64{startNodeID}
	visited[startNodeID] = true

	for len(queue) > 0 && len(visited) < targetNodes {
		current := queue[0]
		queue = queue[1:]

		for _, friend := range adjList[current] {
			if !visited[friend] && len(visited) < targetNodes {
				visited[friend] = true
				queue = append(queue, friend)
			}
		}
	}

	fmt.Printf("BFS completed. Selected %d densely connected users.\n", len(visited))

	fmt.Println("[Sampler] Phase 3: Extracting induced subgraph edges...")
	var sampledRels []Relationship
	for u := range visited {
		for _, v := range adjList[u] {
			if visited[v] {
				sampledRels = append(sampledRels, Relationship{FromUser: u, ToUser: v})
			}
		}
	}

	fmt.Printf("Extracted %d induced relationships.\n", len(sampledRels))

	// Parse user profile metadata
	fmt.Printf("[Sampler] Phase 4: Reading profile metadata for %d users from %s...\n", len(visited), profilePath)
	usersMap := make(map[int64]User)

	profileFile, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile file: %w", err)
	}
	defer profileFile.Close()

	profGzReader, err := gzip.NewReader(profileFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader for profiles: %w", err)
	}
	defer profGzReader.Close()

	profScanner := bufio.NewScanner(profGzReader)

	buf := make([]byte, 64*1024)
	profScanner.Buffer(buf, 1024*1024)

	for profScanner.Scan() {
		line := profScanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}

		id, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || !visited[id] {
			continue
		}

		gender := "unknown"
		if len(fields) > 3 && fields[3] != "null" && fields[3] != "" {
			if fields[3] == "1" {
				gender = "male"
			} else if fields[3] == "0" {
				gender = "female"
			}
		}

		age := 0
		if len(fields) > 4 && fields[4] != "null" && fields[4] != "" {
			if a, err := strconv.Atoi(fields[4]); err == nil {
				age = a
			}
		}

		usersMap[id] = User{
			ID:     id,
			Age:    age,
			Gender: gender,
		}
	}

	if err := profScanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading profiles: %w", err)
	}

	var sampledUsers []User
	for id := range visited {
		if u, ok := usersMap[id]; ok {
			sampledUsers = append(sampledUsers, u)
		} else {
			sampledUsers = append(sampledUsers, User{ID: id, Age: 25, Gender: "unknown"})
		}
	}

	return &SampledGraph{
		Users:         sampledUsers,
		Relationships: sampledRels,
	}, nil
}
