package driver

import "github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/config"

// BuildEngines returns the list of active GraphEngine adapters based on target filter
func BuildEngines(cfg *config.Config, target string) []GraphEngine {
	var list []GraphEngine

	if (target == "all" || target == "cognodb") && cfg.CognoDB.URI != "" {
		list = append(list, NewBoltEngine("CognoDB Cloud", cfg.CognoDB.URI, cfg.CognoDB.User, cfg.CognoDB.Password))
	}
	if (target == "all" || target == "neo4j") && cfg.Neo4j.URI != "" {
		list = append(list, NewBoltEngine("Neo4j AuraDB", cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password))
	}
	if (target == "all" || target == "memgraph") && cfg.Memgraph.URI != "" {
		list = append(list, NewBoltEngine("Memgraph Cloud", cfg.Memgraph.URI, cfg.Memgraph.User, cfg.Memgraph.Password))
	}
	if (target == "all" || target == "falkordb") && cfg.FalkorDB.URI != "" {
		list = append(list, NewBoltEngine("FalkorDB Cloud", cfg.FalkorDB.URI, cfg.FalkorDB.User, cfg.FalkorDB.Password))
	}
	if (target == "all" || target == "arango") && cfg.ArangoDB.URI != "" {
		list = append(list, NewArangoEngine("ArangoDB Remote VM", cfg.ArangoDB.URI, cfg.ArangoDB.User, cfg.ArangoDB.Password, cfg.ArangoDB.Database))
	}
	return list
}