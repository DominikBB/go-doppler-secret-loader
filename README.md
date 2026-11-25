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
