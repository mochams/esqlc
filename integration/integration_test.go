package integration

import (
	"log"
	"os"
	"testing"

	esqlc "github.com/mochams/esqlc"
)

// testState provides test state that can be shared across test functions.
// It is initialized in TestMain.
var testState struct {
	registry *esqlc.Registry
}

// TestMain is the entry point for testing. It initializes the test environment and runs the tests.
func TestMain(m *testing.M) {
	testState.registry = esqlc.NewRegistry(esqlc.DialectPostgres)

	if err := testState.registry.LoadDir("../testdata/queries"); err != nil {
		log.Fatalf("failed to load queries: %v", err)
	}

	setup()
	code := m.Run()
	teardown()

	os.Exit(code)
}

func setup()    {}
func teardown() {}
