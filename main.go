package main

func main() {
	server := InitWebServer()
	err := server.Run("localhost:8080")
	if err != nil {
		panic(err)
	}
}
