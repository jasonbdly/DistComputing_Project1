# Golang Distributed Communication

Install Go (Golang)

 * https://golang.org/dl/


1. Run Server (From Terminal/Command Prompt)

```go run Server.go```

2. Run Client (From Terminal/Command Prompt)

```go run Client.go```

3. Once both programs have connected, text can be sent to the Server from the Client and back.


Run the System Manager

1. Ensure dependant executables have been built for your platform

```go build ServerRouter.go```

```go build P2P.go```

2. Run the system manager

```go run SystemManager.go```