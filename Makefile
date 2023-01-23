proto:
	rm -f proto/*.go
	protoc --go_opt=module=github.com/medenzel/grpc-fileserver \
	--go-grpc_opt=module=github.com/medenzel/grpc-fileserver \
	--go_out=proto --go-grpc_out=proto \
	proto/*.proto

.PHONY: proto