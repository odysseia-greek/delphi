package architect

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretRef struct {
	SecretName string
	Namespace  string // source namespace
}

func (p *PeriklesHandler) copySecretAllKeys(sourceNamespace, targetNamespace, secretName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	src, err := p.Kube.CoreV1().Secrets(sourceNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get source secret %s/%s: %w", sourceNamespace, secretName, err)
	}

	// Try get destination
	dst, err := p.Kube.CoreV1().Secrets(targetNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			// Create new
			immutable := false
			newAnnotations := map[string]string{
				IgnoreInGitOps:   "true",
				AnnotationUpdate: time.Now().UTC().Format(timeFormat),
			}

			s := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        secretName,
					Namespace:   targetNamespace,
					Annotations: newAnnotations,
					Labels:      src.Labels, // optional: keep labels
				},
				Immutable: &immutable,
				Type:      src.Type,
				Data:      cloneBytesMap(src.Data),
			}

			_, err := p.Kube.CoreV1().Secrets(targetNamespace).Create(ctx, s, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("create target secret %s/%s: %w", targetNamespace, secretName, err)
			}

			logging.System(fmt.Sprintf("secret created: %s/%s", targetNamespace, secretName))
			return nil
		}

		return fmt.Errorf("get target secret %s/%s: %w", targetNamespace, secretName, err)
	}

	// Destination exists: update only if different
	if secretsDataEqual(src.Data, dst.Data) && src.Type == dst.Type {
		// Still ensure our annotations exist (optional); avoid noisy updates.
		return nil
	}

	immutable := false
	newAnnotations := map[string]string{}
	for k, v := range dst.Annotations {
		newAnnotations[k] = v
	}
	newAnnotations[IgnoreInGitOps] = "true"
	newAnnotations[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)

	updated := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Namespace:   targetNamespace,
			Annotations: newAnnotations,
			Labels:      dst.Labels, // keep existing labels by default
		},
		Immutable: &immutable,
		Type:      src.Type,                // mirror source type
		Data:      cloneBytesMap(src.Data), // mirror source data (all keys)
	}

	_, err = p.Kube.CoreV1().Secrets(targetNamespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update target secret %s/%s: %w", targetNamespace, secretName, err)
	}

	logging.System(fmt.Sprintf("secret updated: %s/%s", targetNamespace, secretName))
	return nil
}

func cloneBytesMap(in map[string][]byte) map[string][]byte {
	if in == nil {
		return map[string][]byte{}
	}
	out := make(map[string][]byte, len(in))
	for k, v := range in {
		if v == nil {
			out[k] = nil
			continue
		}
		b := make([]byte, len(v))
		copy(b, v)
		out[k] = b
	}
	return out
}

func secretsDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
	}
	return true
}

func parseReplicateFrom(val string) []SecretRef {
	var refs []SecretRef

	entries := strings.Split(val, ";")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.Split(entry, ":")
		if len(parts) != 2 {
			logging.Warn(fmt.Sprintf(
				"invalid perikles/replicate-from entry %q (expected secret:namespace)", entry))
			continue
		}

		secretName := strings.TrimSpace(parts[0])
		sourceNS := strings.TrimSpace(parts[1])

		if secretName == "" || sourceNS == "" {
			logging.Warn(fmt.Sprintf(
				"invalid perikles/replicate-from entry %q (empty secret or namespace)", entry))
			continue
		}

		refs = append(refs, SecretRef{
			SecretName: secretName,
			Namespace:  sourceNS,
		})
	}

	return refs
}
