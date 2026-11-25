# go-doppler-secret-loader

A dependency-free Go client that loads computed Doppler secrets over defaults.

```go
import (
    "context"
    "log"

    doppler "github.com/dominikbb/go-doppler-secret-loader"
)

func main() {
    // Reads DOPPLER_SERVICE_TOKEN. Override with doppler.WithAPIKey("...").
    client, err := doppler.New("my-project")
    if err != nil {
        log.Fatal(err)
    }

    secrets, err := client.ListSecrets(context.Background(), "dev", map[string]string{
        "DB_PASS": "",         // Required from Doppler.
        "DB_USER": "postgres", // Fallback if Doppler omits it or returns empty.
    })
    if err != nil {
        log.Fatal(err)
    }
    _ = secrets // Pass to application configuration. Do not log secret values.
}
```

Only supplied keys are requested using the list endpoint's `secrets` filter.
Non-empty computed values override defaults. Missing or empty values preserve
those defaults. If any final value is empty, the method returns a nil map and an
error listing the unresolved keys. The input map is never modified and extra
response keys are ignored. An empty input returns an empty map without a request.

The project is configured once in `New`, while each call selects its config.
`WithHTTPClient` supports a custom transport or timeout. The default timeout is
30 seconds. Request contexts support cancellation. HTTP and decoding failures
are returned even when all keys have defaults. Error response bodies are not
included in errors to avoid exposing secrets.

Run checks with `go test -race ./...` and `go vet ./...`.

## Live integration test

Skipped by default. Set your token in `DOPPLER_SERVICE_TOKEN` through your shell
or secret manager, then supply your project, config, and defaults:

```sh
DOPPLER_INTEGRATION=1 \
DOPPLER_TEST_PROJECT='my-project' \
DOPPLER_TEST_CONFIG='dev' \
DOPPLER_TEST_DEFAULTS='{"DB_PASS":"","DB_USER":"postgres"}' \
go test -run '^TestIntegrationListSecrets$' -count=1 -v
```

Use at least one empty default to verify that a value actually comes from Doppler.
Optionally set `DOPPLER_TEST_EXPECTED` to a JSON object of expected key/value pairs
for exact assertions. Secret values are never printed by the test. Do not commit
credentials or expected secret values. `go test -short` skips the live test even
when enabled. The test only reads secrets and has a 30-second timeout.

