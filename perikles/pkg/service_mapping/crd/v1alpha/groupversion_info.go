package v1alpha

import (
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeBuilder = k8sruntime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
	GroupVersion  = schema.GroupVersion{Group: GroupName, Version: Version}
)

func addKnownTypes(scheme *k8sruntime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Mapping{},
		&MappingList{},
	)
	return nil
}
