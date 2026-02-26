This directory is intended to hold the generated CRD manifests for Perikles.

Generate the CRD YAML using controller-gen:

  go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
  cd perikles/pkg/service_mapping/crd/v1alpha
  go generate ./...

This will output odysseia-greek.com_servicemappings.yaml in perikles/config/crd.

You can then apply it to your cluster:

  kubectl apply -f perikles/config/crd/
