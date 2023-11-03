vault_certs:
	kubectl get secret vault-server-tls -n odysseia -o=jsonpath='{.data.vault\.ca}' | base64 --decode > /tmp/vault.ca
	kubectl get secret vault-server-tls -n odysseia -o=jsonpath='{.data.vault\.crt}' | base64 --decode > /tmp/vault.crt
	kubectl get secret vault-server-tls -n odysseia -o=jsonpath='{.data.vault\.key}' | base64 --decode > /tmp/vault.key
