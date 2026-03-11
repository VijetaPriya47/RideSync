PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
GO_OUT := .

.PHONY: generate-proto
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)\n.PHONY: docs-build\ndocs-build:\n\tcd docs-site && npm run build\n\n.PHONY: docs-serve\ndocs-serve:\n\tcd docs-site && npm start
\n.PHONY: docs-build\ndocs-build:\n\tcd docs-site && npm run build\n\n.PHONY: docs-serve\ndocs-serve:\n\tcd docs-site && npm start
