package architect

import "time"

const (
	IgnoreInGitOps               = "gitops.ignore"
	AnnotationUpdate             = "perikles/updated"
	AnnotationSourceKind         = "perikles/source-kind"
	AnnotationSourceName         = "perikles/source-name"
	AnnotationSourceNamespace    = "perikles/source-namespace"
	AnnotationSourceUID          = "perikles/source-uid"
	AnnotationHost               = "perikles/hostname"
	AnnotationAccesses           = "perikles/accesses"
	AnnotationNamespaceBootstrap = "perikles/bootstrap"
	AnnotationReplicateFrom      = "perikles/replicate-from"

	BootstrapSourceNamespace = "delphi"

	VaultTLSSourceSecret = "vault-server-tls"
	SolonTLSSourceSecret = "solon-tls-certs"

	staleJobPolicyAge = 24 * time.Hour

	timeFormat string = "2006-01-02 15:04:05"
)
