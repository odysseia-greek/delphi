
solon_certs:
	kubectl get secret solon-tls-certs -o jsonpath="{.data['tls\.crt']}" | base64 -d > /tmp/solon/tls.crt
	kubectl get secret solon-tls-certs -o jsonpath="{.data['tls\.pem']}" | base64 -d > /tmp/solon/tls.pem
	kubectl get secret solon-tls-certs -o jsonpath="{.data['tls\.key']}" | base64 -d > /tmp/solon/tls.key

vault_certs:
	kubectl get secret vault-server-tls -o jsonpath="{.data['vault\.crt']}" | base64 -d > /tmp/vault/vault.crt
	kubectl get secret vault-server-tls -o jsonpath="{.data['vault\.ca']}" | base64 -d > /tmp/vault/vault.ca
	kubectl get secret vault-server-tls -o jsonpath="{.data['vault\.key']}" | base64 -d > /tmp/vault/vault.key

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	./proto/ptolemaios.proto

dirs:
	mkdir -p /tmp/vault
	mkdir -p /tmp/solon

create_secret:
	curl --request POST \
	  --url https://vault:8200/v1/configs/data/ptolemaios \
	  --header 'Authorization: Bearer s.Far2HI1RZkobZKh7HuFQMNGt' \
	  --header 'Content-Type: application/json' \
	  --data '{ "elasticUsername": "test", \
	 "elasticPassword": "test", \
	 "ElasticCERT": "ssssss" \
	}'