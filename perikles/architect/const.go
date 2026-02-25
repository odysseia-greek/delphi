package architect

const (
	IgnoreInGitOps               = "gitops.ignore"
	AnnotationUpdate             = "perikles/updated"
	AnnotationHost               = "perikles/hostname"
	AnnotationAccesses           = "perikles/accesses"
	AnnotationNamespaceBootstrap = "perikles/bootstrap"
	AnnotationReplicateFrom      = "perikles/replicate-from"

	BootstrapSourceNamespace = "delphi"

	VaultTLSSourceSecret = "vault-server-tls"
	SolonTLSSourceSecret = "solon-tls-certs"

	timeFormat string = "2006-01-02 15:04:05"
)
