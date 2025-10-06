package pkg

import (
    "front_start/internal/api"
)

// App is a thin wrapper to mirror the reference project layout.
// It starts the HTTP server.
func App() {
    api.StartServer()
}


