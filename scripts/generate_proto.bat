@echo off
REM generate_proto.bat

REM Install protoc-gen-go and protoc-gen-go-grpc if not installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

REM Generate Go files from proto (api folder structure)
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/v1/notif.proto

echo Proto files generated successfully in api/v1/
echo Generated files:
echo   - api/v1/notif.pb.go
echo   - api/v1/notif_grpc.pb.go
