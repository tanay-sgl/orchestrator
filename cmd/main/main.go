package main

import "orchestrator/internal/api"

func main() {
	r := api.SetupRouter()
	r.Run(":8080")
}
