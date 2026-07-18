package spicedb

import (
	"context"
	"flag"
	"testing"
	"time"
)

var (
	liveEndpoint = flag.String("spicedb-endpoint", "", "SpiceDB integration test endpoint")
	liveToken    = flag.String("spicedb-token", "", "SpiceDB integration test token")
	liveInsecure = flag.Bool("spicedb-insecure", false, "use plaintext transport for the SpiceDB integration test")
)

func TestRuntimeCredentialReadsRequiredSchemaButCannotRewriteIt(t *testing.T) {
	if *liveEndpoint == "" {
		t.Skip("-spicedb-endpoint is required for live SpiceDB integration")
	}
	authorizer, err := New(*liveEndpoint, *liveToken, *liveInsecure)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := authorizer.Health(ctx); err != nil {
		t.Fatalf("required schema was not healthy: %v", err)
	}
	if err := authorizer.WriteSchema(ctx, "definition user {}"); err == nil {
		t.Fatal("runtime credential rewrote the authorization schema")
	}
}
