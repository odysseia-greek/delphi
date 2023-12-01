proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	./proto/ptolemaios.proto

generate_docs:
	docker run --rm \
		-v $(CURDIR)/docs:/out \
		-v $(CURDIR)/proto:/protos \
		pseudomuto/protoc-gen-doc --doc_opt=markdown,docs.md
