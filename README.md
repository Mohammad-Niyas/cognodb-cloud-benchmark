# Cloud Graph Database Benchmark Suite

> **A Rigorous, Automated, and Reproducible Performance Comparison of CognoDB Cloud and Managed Graph Database Platforms**

---

## 1. Executive Summary

This repository contains an automated and reproducible benchmark suite comparing **CognoDB Cloud** with four managed graph database platforms:

- **Neo4j AuraDB**
- **Memgraph Cloud**
- **FalkorDB Cloud**
- **ArangoDB**

The benchmark evaluates all five platforms using the **same logical dataset, equivalent workload definitions, and a consistent benchmark environment**. Resource allocation, platform tiers, regional configuration, and other differences are documented explicitly to ensure a fair and transparent comparison.

The goal is not to identify a universal "winner", but to understand how each platform performs under the tested workloads and resource constraints.

### Key Benchmark Pillars

- **Fairness:** The same dataset, logical workloads, client environment, and measurement procedure are used across all platforms. Where exact resource parity is not possible, the differences are documented and considered in the analysis.

- **Workload Coverage:** The benchmark measures data-ingestion throughput, 1-hop, 2-hop, and 3-hop traversal latency, point and indexed lookups, aggregation queries, and concurrent mixed read/write workloads.

- **Deterministic Measurement:** Each workload is preceded by a defined warm-up phase and followed by at least **100 measurement iterations**, with latency reported using **p50 and p95 percentiles**.

- **Reproducibility:** Dataset preparation, database loading, workload execution, metric collection, and result generation are automated through the benchmark harness.

- **Transparent Reporting:** The benchmark reports performance results together with platform specifications, configuration details, failures, timeouts, resource limitations, and other relevant caveats.

### Databases Compared

| Platform | Role in Benchmark |
|---|---|
| **CognoDB Cloud** | Primary platform under evaluation |
| **Neo4j AuraDB** | Managed graph database comparison |
| **Memgraph Cloud** | Managed graph database comparison |
| **FalkorDB Cloud** | Managed graph database comparison |
| **ArangoDB** | Managed graph / multi-model database comparison |

> **Benchmark note:** Exact platform resources, dataset size, regions, and tier configurations are reported in the sections below and are not assumed to be identical where the providers expose different limits.
## 2. Benchmark Scope & Objectives

### 2.1 Primary Objectives

1. **Traversal Latency Comparison:** Measure 1-hop, 2-hop, and 3-hop traversal latency across all five platforms using equivalent logical workloads and the same dataset.

2. **Ingestion Efficiency:** Measure batch data-loading performance using nodes/second, relationships/second, and total wall-clock load time.

3. **Lookup and Aggregation Performance:** Evaluate point lookups, indexed or filtered lookups, and aggregation workloads using p50 and p95 latency.

4. **Concurrent Workload Behavior:** Measure sustained throughput, tail latency, and error rate under increasing concurrent client loads, including 1, 10, and 40 concurrent clients where supported by the selected configurations.

5. **Fairness and Transparency:** Document platform resources, configuration differences, network conditions, workload definitions, indexing choices, and other factors that may influence the measured results.

### 2.2 What This Benchmark Measures

* **Ingest Throughput:** Total load time, nodes/second, and relationships/second.
* **Traversal Latency:** p50 and p95 latency for 1-hop, 2-hop, and 3-hop traversals.
* **Lookup Latency:** p50 and p95 latency for point and indexed/filtered lookups.
* **Aggregation Latency:** p50 and p95 latency for count/group-by style workloads.
* **Concurrent Throughput:** Queries per second (QPS), p50/p95 latency, and error rate under concurrent read/write workloads.
* **Resource Footprint:** Stored data size, memory usage, instance specifications, and other resource metrics where observable.

### 2.3 What This Benchmark Does NOT Claim

* **Universal Performance Ranking:** The results do not establish a universally "best" graph database. Performance depends on workload, dataset, configuration, concurrency, and platform constraints.

* **Production-Scale Performance:** The benchmark focuses on entry-level or free/trial configurations and should not be interpreted as representative of large production clusters or enterprise-scale deployments.

* **Large-Graph Performance:** The benchmark uses a deliberately constrained public dataset that fits the selected benchmark environments. Results should not be extrapolated directly to billion-node or terabyte-scale graphs.

* **Perfect Hardware Equivalence:** Cloud providers expose different tiers and resource limits. Where exact CPU, memory, storage, or networking parity is not possible, those differences are documented and considered when interpreting the results.

* **Internal Engine Causality:** The benchmark observes externally measurable behavior. It does not by itself prove the internal architectural or execution-plan reasons behind every performance difference.
## 3. Platforms Compared

### 3.1 Why These Five Databases?

The benchmark compares five graph-capable database platforms representing different architectural and product approaches:

| Platform | Architectural / Product Model | Why Included |
| :--- | :--- | :--- |
| **CognoDB Cloud** | Managed graph database | **Primary target:** The platform being evaluated in this benchmark. |
| **Neo4j AuraDB** | Managed native property graph | **Established graph baseline:** A widely used property-graph platform with native Cypher support. |
| **Memgraph Cloud** | Managed graph database | **Graph-performance comparison:** A graph-focused in-memory platform with native Cypher support. |
| **FalkorDB Cloud** | Managed graph database | **Alternative graph architecture:** Provides a Redis-backed sparse matrix implementation for graph workloads. |
| **ArangoDB** | Multi-model database with graph capabilities | **Multi-model baseline:** Enables comparison against a platform supporting both document and graph workloads. |

---

### 3.2 Platform Resource Matrix

The table below records the actual deployment configuration used for the benchmark. Resource values are reported from provider documentation or observed directly in the deployed instance configuration.

| Platform | Deployment Model | Tier / Size | Engine Version | Compute (vCPU) | Memory (RAM) | Storage | Platform / Dataset Limits | Cloud Provider & Region | Interface Protocol |
| :--- | :--- | :--- | :--- | ---: | ---: | :--- | :--- | :--- | :--- |
| **CognoDB Cloud** | Managed Cloud | `c0` Free | Latest | 0.5 (burst) | 512 MB* | 1 GiB | 50,000 max result rows | AWS / `us-east4` | Bolt (`bolt+s://`) |
| **Neo4j AuraDB** | Managed Cloud | AuraDB Free | v2026.07 (Observed) | Not disclosed | Not disclosed | Storage Included | 200k nodes / 400k relationships | GCP / `us-central1` | Bolt (`neo4j+s://`) |
| **Memgraph Cloud** | Managed Cloud | Free Trial (14d) | v3.12.0 | 2 vCPU | 2 GB | 14 GB Disk (In-Memory) | Free trial quota | AWS / `us-east-1` | Bolt (`bolt+ssc://`) |
| **FalkorDB Cloud** | Managed Cloud | FalkorDB Free | v4.20.1 | Shared | Shared | Cloud Storage | Free tier quota | AWS / `us-east-1` | Bolt (`bolt://`) |
| **ArangoDB** | Remote Self-Hosted VM | Community | v3.12+ | **0.5 vCPU** | **512 MB** | 1 GiB Host Disk | Capped by Docker limits | Remote VM / US-East | HTTP / AQL |

\* *Note on CognoDB RAM:* The assignment text notes an expected baseline of 256 MB RAM, whereas the live console UI provisions `c0` instances with 512 MB RAM. The observed deployed specification (512 MB) is recorded above as the benchmark fact.

---

### 3.3 Fairness & Resource Parity Assessment

The benchmark separates controlled variables from platform-specific constraints.

#### Controlled Variables
* Identical dataset
* Identical dataset sampling procedure
* Equivalent logical workload definitions
* Database-specific query implementations executing the same logical operation
* Same benchmark client
* Same warm-up procedure
* Same measurement iteration count
* Same concurrency levels
* Same latency and throughput calculation methodology
* Same timeout and error-recording policy

#### Platform-Specific Variables
* Available CPU and memory tiers
* Storage and dataset limits
* Cloud-provider infrastructure and multi-tenant resource sharing
* Region availability and network routing paths
* Free-tier throttling or quotas
* Resource observability
* Query language and internal execution model

---

### 3.4 Regional Alignment & Network Topology

ArangoDB is hosted on a remote cloud VM rather than `localhost` so that all benchmarked databases are accessed remotely over public network paths from the same benchmark client:

```text
                       Benchmark Client
                              │
        ┌─────────────┬───────┼────────┬────────────┐
        ▼             ▼       ▼        ▼            ▼
    CognoDB         Neo4j  Memgraph  FalkorDB    ArangoDB
     Cloud          Cloud    Cloud    Cloud     Remote VM
```

---
## 4. Dataset & Schema

### 4.1 Dataset Source & Scale

The benchmark uses a deterministic sample derived from the **SNAP soc-Pokec social network dataset**, a public directed social graph representing anonymized users and their relationships.

- **Source:** Stanford Network Analysis Project (SNAP)
- **Dataset:** soc-Pokec
- **Dataset URL:** [https://snap.stanford.edu/data/soc-Pokec.html](https://snap.stanford.edu/data/soc-Pokec.html)
- **Full Dataset:** 1,632,803 nodes / 30,622,564 directed edges
- **Files Used:**
  - `soc-pokec-relationships.txt.gz`
  - `soc-pokec-profiles.txt.gz`
- **Graph Type:** Directed, unweighted social graph

---

### 4.2 Sampling Strategy & Determinism

The benchmark generates a deterministic subgraph from the public dataset using a fixed sampling configuration.

The preprocessing pipeline:
1. Selects a deterministic set of user IDs using a fixed random seed (`seed = 42`).
2. Builds the induced subgraph by retaining edges whose source and target nodes are both present in the selected node set.
3. Generates canonical `data/nodes.csv` and `data/relationships.csv` files.
4. Validates the resulting node and relationship counts.
5. Repeats the deterministic sampling configuration with a larger target if necessary until the final graph satisfies the assignment requirement of at least **100,000 relationships** and fits the selected benchmark environments.
6. Computes SHA-256 checksums and writes a `data/manifest.json` file.

> **Important:** The final node and relationship counts are measured directly from the generated dataset rather than assumed.

---

### 4.3 Benchmark Dataset Specification

The final sample values below are measured directly from the dataset prep step:

| Parameter | Benchmark Value | Full SNAP Value |
| :--- | ---: | ---: |
| **Nodes (Users)** | *Measured after sampling* | 1,632,803 |
| **Relationships (Friendships)** | *Measured after sampling* ($\ge 100,000$) | 30,622,564 |
| **Node Label** | `User` | — |
| **Relationship Type** | `FRIEND` | — |
| **Node Properties Used** | `id`, `age`, `gender` | Source profile data |
| **Graph Structure** | Directed, unweighted | Directed |

---

### 4.4 Graph Schema

The benchmark represents the canonical graph schema as:

```text
(:User {id, age, gender})
       │
       │ FRIEND
       ▼
(:User {id, age, gender})

```

---

### 4.5 Dataset Validation & Integrity

Before ingestion, the canonical dataset is validated independently of the target database. Validation checks include:
* Node count and relationship count match expected thresholds.
* No duplicate node IDs or invalid relationship endpoints exist.
* Canonical files (`nodes.csv`, `relationships.csv`) match SHA-256 manifest checksums.

After ingestion, every platform performs native node and relationship count queries. Benchmark execution begins only after the platform counts match the canonical dataset.

---

### 4.6 Platform Dataset Loading Strategy

Each platform receives the exact same canonical dataset. Only the ingestion mechanism varies according to the platform's supported driver or API:

| Platform | Ingestion Strategy | Batch / Transaction Size | Validation Query |
| :--- | :--- | :--- | :--- |
| **CognoDB Cloud** | Neo4j Go Driver / Cypher `UNWIND` | 5,000 records per batch | Node + Relationship count |
| **Neo4j AuraDB** | Neo4j Go Driver / Cypher `UNWIND` | 5,000 records per batch | Node + Relationship count |
| **Memgraph Cloud** | Neo4j Go Driver / Cypher `UNWIND` | 5,000 records per batch | Node + Relationship count |
| **FalkorDB Cloud** | Platform-supported graph driver/API | TBD | Node + Relationship count |
| **ArangoDB** | Go HTTP client / AQL bulk ingestion | 5,000 records per batch | Document + Edge count |

> **Ingestion Timing Policy:** Load-time measurements begin when the first database write is issued and end when the final write is confirmed. Dataset download, preprocessing, and CSV generation are excluded from database ingestion time.

---

