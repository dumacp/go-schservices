#!/bin/bash
#protoc -I=. -I=$GOROOT/src --gogoslick_out=plugins=grpc:. messages.proto
protoc -I=. -I=$GOROOT/src --go_out=. --go_opt=paths=source_relative ./services.proto
protoc -I=. -I=$GOROOT/src --go_out=. --go_opt=paths=source_relative ./servicesmsg.proto
#protoc -I=. -I=$GOROOT/src --gogoslick_out=plugins=grpc:$GOPATH/src --proto_path=. messages.proto
