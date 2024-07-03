package main

func main() {
	r := SetupRouter()
	r.Run(":8080")
}
