package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func main() {
	candidatePath := flag.String("candidate", "", "path to a validated candidate JSON artifact")
	databaseURL := flag.String("database-url", "", "explicit local test-only PostgreSQL URL")
	syntheticProfile := flag.Bool("synthetic-test-profile", false, "bootstrap only the internal synthetic regulatory test profile")
	flag.Parse()
	if *candidatePath == "" || *databaseURL == "" || os.Getenv("AVIA_REGULATORY_TEST_MODE") != "1" {
		fmt.Fprintln(os.Stderr, "candidate and explicit local database-url are required")
		os.Exit(2)
	}
	parsed, err := url.Parse(*databaseURL)
	if err != nil || !loopbackHost(parsed.Hostname()) || parsed.Path == "/" || !(contains(parsed.Path, "task") || contains(parsed.Path, "test")) {
		fmt.Fprintln(os.Stderr, "only explicit loopback test-only PostgreSQL databases are allowed")
		os.Exit(2)
	}
	bytes, err := os.ReadFile(*candidatePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var candidate regulatory.CandidateBundle
	if err := json.Unmarshal(bytes, &candidate); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pool, err := database.Open(context.Background(), *databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := migrations.Apply(context.Background(), pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *syntheticProfile {
		if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	result, err := (regulatory.ImportStore{Pool: pool}).Import(context.Background(), candidate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("imported %s %s %s replayed=%t\n", result.GenerationRunID, result.InputDigest, result.OutputDigest, result.Replayed)
}

func loopbackHost(host string) bool { return host == "localhost" || net.ParseIP(host).IsLoopback() }
func contains(value, marker string) bool {
	for index := 0; index+len(marker) <= len(value); index++ {
		if value[index:index+len(marker)] == marker {
			return true
		}
	}
	return false
}
