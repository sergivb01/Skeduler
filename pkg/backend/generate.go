package backend

//go:generate protoc -I=../../proto/ --go_out=. --go-grpc_out=. backend.proto
