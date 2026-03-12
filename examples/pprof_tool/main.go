package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/adnilis/x-hmsw/api"
	iface "github.com/adnilis/x-hmsw/interface"
	"github.com/adnilis/x-hmsw/types"
)

var (
	cpuprofile  = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile  = flag.String("memprofile", "", "write memory profile to file")
	benchmark   = flag.Bool("benchmark", false, "run benchmark test")
	pprofServer = flag.Bool("pprof", false, "start pprof http server")
	testVectors = flag.Int("vectors", 10000, "number of vectors for testing")
	testDim     = flag.Int("dim", 128, "vector dimension")
	testQueries = flag.Int("queries", 1000, "number of queries")
	iterations  = flag.Int("iterations", 3, "number of iterations for benchmarking")
)

const TestStoragePath = "./test_pprof_data"

func main() {
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         pprof Memory & CPU Optimization Tool                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	if *pprofServer {
		startPprofServer()
	}

	if *benchmark {
		runBenchmark()
	}

	printSystemInfo()
}

func startPprofServer() {
	fmt.Println("\n=== Starting pprof HTTP Server ===")
	fmt.Println("Access the following URLs for profiling:")
	fmt.Println("  - CPU profile: http://localhost:6060/debug/pprof/profile?seconds=30")
	fmt.Println("  - Heap profile: http://localhost:6060/debug/pprof/heap")
	fmt.Println("  - Goroutine profile: http://localhost:6060/debug/pprof/goroutine")
	fmt.Println("  - Block profile: http://localhost:6060/debug/pprof/block")
	fmt.Println("  - Mutex profile: http://localhost:6060/debug/pprof/mutex")
	fmt.Println("\nPress Ctrl+C to stop the server")

	http.ListenAndServe(":6060", nil)
}

func runBenchmark() {
	fmt.Printf("\n=== Running Benchmark with Profiling ===\n")
	fmt.Printf("Vectors: %d, Dimension: %d, Queries: %d\n", *testVectors, *testDim, *testQueries)

	var cpuFile *os.File
	var memFile *os.File
	var err error

	if *cpuprofile != "" {
		cpuFile, err = os.Create(*cpuprofile)
		if err != nil {
			fmt.Printf("Failed to create CPU profile: %v\n", err)
			return
		}
		defer cpuFile.Close()
		pprof.StartCPUProfile(cpuFile)
		defer pprof.StopCPUProfile()
	}

	if *memprofile != "" {
		memFile, err = os.Create(*memprofile)
		if err != nil {
			fmt.Printf("Failed to create memory profile: %v\n", err)
			return
		}
		defer memFile.Close()
	}

	printMemStats("Before test")

	var totalInsertTime, totalSearchTime time.Duration

	for i := 0; i < *iterations; i++ {
		fmt.Printf("\n--- Iteration %d/%d ---\n", i+1, *iterations)

		storagePath := filepath.Join(TestStoragePath, fmt.Sprintf("run_%d", i))

		db, err := api.NewBuilder().
			WithStoragePath(storagePath).
			WithIndexType(iface.HNSW).
			WithStorageType(iface.Memory).
			WithDimension(*testDim).
			WithAutoSave(false, 0).
			Build()
		if err != nil {
			fmt.Printf("Failed to create database: %v\n", err)
			return
		}

		vectors := generateVectors(*testVectors, *testDim)
		queries := generateQueries(*testQueries, *testDim)

		start := time.Now()
		if err := db.Insert(vectors); err != nil {
			fmt.Printf("Failed to insert vectors: %v\n", err)
			db.Close()
			return
		}
		insertTime := time.Since(start)
		totalInsertTime += insertTime
		fmt.Printf("Insert: %v (%.2f vectors/sec)\n", insertTime, float64(*testVectors)/insertTime.Seconds())

		printMemStats("After insert")

		start = time.Now()
		var searchCount int
		for _, query := range queries {
			_, err := db.Search(types.Vector{Vector: query}, 10)
			if err != nil {
				fmt.Printf("Search error: %v\n", err)
			}
			searchCount++
		}
		searchTime := time.Since(start)
		totalSearchTime += searchTime
		fmt.Printf("Search: %v (%.2f queries/sec)\n", searchTime, float64(searchCount)/searchTime.Seconds())

		printMemStats("After search")

		db.Close()

		if memFile != nil {
			runtime.GC()
			pprof.WriteHeapProfile(memFile)
		}

		os.RemoveAll(storagePath)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Average Insert Time: %v\n", totalInsertTime/time.Duration(*iterations))
	fmt.Printf("Average Search Time: %v\n", totalSearchTime/time.Duration(*iterations))
	fmt.Printf("Average QPS: %.2f\n", float64(*testQueries**iterations)/totalSearchTime.Seconds())

	printMemStats("Final")

	if *cpuprofile != "" {
		fmt.Printf("\nCPU profile saved to: %s\n", *cpuprofile)
		fmt.Println("To analyze: go tool pprof -http=:8080", *cpuprofile)
	}

	if *memprofile != "" {
		fmt.Printf("\nMemory profile saved to: %s\n", *memprofile)
		fmt.Println("To analyze: go tool pprof -http=:8080", *memprofile)
	}

	printOptimizationSuggestions()
}

func printSystemInfo() {
	fmt.Println("\n=== System Information ===")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\nMemory Stats (Before):\n")
	fmt.Printf("  Alloc: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("  TotalAlloc: %.2f MB\n", float64(m.TotalAlloc)/1024/1024)
	fmt.Printf("  Sys: %.2f MB\n", float64(m.Sys)/1024/1024)
	fmt.Printf("  NumGC: %d\n", m.NumGC)
}

func printMemStats(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("\n%s:\n", label)
	fmt.Printf("  Alloc: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("  TotalAlloc: %.2f MB\n", float64(m.TotalAlloc)/1024/1024)
	fmt.Printf("  Sys: %.2f MB\n", float64(m.Sys)/1024/1024)
	fmt.Printf("  NumGC: %d\n", m.NumGC)
}

func printOptimizationSuggestions() {
	fmt.Println("\n=== Optimization Suggestions ===")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.NumGC > 10 {
		fmt.Println("⚠ High GC frequency detected. Suggestions:")
		fmt.Println("  - Use object pooling (sync.Pool) for frequently allocated objects")
		fmt.Println("  - Reduce allocations in hot paths")
		fmt.Println("  - Consider using []float32 slices directly instead of types.Vector")
	}

	if m.Alloc > 100*1024*1024 {
		fmt.Println("⚠ High memory usage detected. Suggestions:")
		fmt.Println("  - Use vector compression (PQ, SQ, Binary)")
		fmt.Println("  - Consider using memory-mapped storage (MMap)")
		fmt.Println("  - Implement index tiering (hot/cold data)")
	}

	fmt.Println("\nProfiling Tips:")
	fmt.Println("  - Run: go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.")
	fmt.Println("  - Analyze: go tool pprof -http=:8080 cpu.prof")
	fmt.Println("  - View top functions: go tool pprof -top cpu.prof")
	fmt.Println("  - View flame graph: go tool pprof -flame cpu.prof")
}

func generateVectors(count, dim int) []types.Vector {
	vectors := make([]types.Vector, count)
	for i := 0; i < count; i++ {
		vec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			vec[j] = rand.Float32()*2 - 1
		}
		vectors[i] = types.Vector{
			ID:     fmt.Sprintf("vec_%d", i),
			Vector: vec,
		}
	}
	return vectors
}

func generateQueries(count, dim int) [][]float32 {
	queries := make([][]float32, count)
	for i := 0; i < count; i++ {
		query := make([]float32, dim)
		for j := 0; j < dim; j++ {
			query[j] = rand.Float32()*2 - 1
		}
		queries[i] = query
	}
	return queries
}

func init() {
	_ = time.Now()
}
