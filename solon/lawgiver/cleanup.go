package lawgiver

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (s *SolonHandler) StartCleanupService(cleanUpLoop time.Duration) error {
	ticker := time.NewTicker(cleanUpLoop)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logging.System("starting cleanup service...")
			err := s.safeCleanup()
			if err != nil {
				return err
			}
		}
	}
}

func (s *SolonHandler) safeCleanup() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during cleanup: %v", r)
		}
	}()

	err = s.cleanup()
	return
}

func (s *SolonHandler) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := s.Kube.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current pods: %w", err)
	}

	// Get all users in Elastic
	elasticUsers, err := s.Elastic.Access().ListUsers()
	if err != nil {
		return fmt.Errorf("failed to get Elastic users: %w", err)
	}

	// Get all secrets in Vault
	vaultSecrets, err := s.Vault.ListSecrets()
	if err != nil {
		return fmt.Errorf("failed to get Vault secrets: %w", err)
	}

	vaultPolicies, err := s.Vault.ListPolicies()
	if err != nil {
		return fmt.Errorf("failed to get Vault policies: %w", err)
	}

	// Identify and delete orphaned users/secrets
	if err := s.deleteOrphans(pods, elasticUsers, vaultSecrets, vaultPolicies); err != nil {
		return fmt.Errorf("failed to delete orphans: %w", err)
	}

	return nil
}

func (s *SolonHandler) deleteOrphans(currentPods *v1.PodList, elasticUsers []string, vaultSecrets, vaultPolicies []string) error {
	// Identify orphaned Elastic users
	numberOfCleanedResource := 0
	for _, user := range elasticUsers {
		if user == config.DefaultTracingName || user == config.DefaultMetricsName {
			continue
		}
		userFound := false
		for _, pod := range currentPods.Items {
			splitPodName := strings.Split(pod.Name, "-")
			var username string
			//this logic is from periandros because elastic does not accept hyphem in a username
			if len(splitPodName) > 1 {
				username = splitPodName[0] + splitPodName[len(splitPodName)-1]
			} else {
				username = splitPodName[0]
			}

			if user == username {
				userFound = true
				break
			}
		}

		if !userFound {
			_, err := s.Elastic.Access().DeleteUser(user)
			if err != nil {
				logging.Error(fmt.Sprintf("failed to delete orphaned user: %s, %s", user, err.Error()))
				continue
			}
			logging.System(fmt.Sprintf("deleted orphan elatisUser: %s", user))
			numberOfCleanedResource++
		}
	}

	// Identify orphaned Vault secrets
	for _, secret := range vaultSecrets {
		secretFound := false
		for _, pod := range currentPods.Items {
			if secret == pod.Name {
				secretFound = true
				break
			}
		}

		if !secretFound {
			err := s.Vault.DeleteSecret(secret)
			if err != nil {
				logging.Error(fmt.Sprintf("failed to delete orphaned secret: %s, %s", secret, err.Error()))
				continue
			}

			err = s.Vault.RemoveSecret(secret)
			if err != nil {
				logging.Error(fmt.Sprintf("failed to remove orphaned secret: %s, %s", secret, err.Error()))
				continue
			}

			logging.System(fmt.Sprintf("deleted orphan secret: %s", secret))
			numberOfCleanedResource++
		}
	}

	protectedPolicies := map[string]bool{
		"default": true,
		"root":    true,
		"solon":   true,
	}
	for _, policy := range vaultPolicies {
		if protectedPolicies[policy] {
			continue
		}
		policyFound := false
		for _, pod := range currentPods.Items {
			if policy == fmt.Sprintf("policy-%s", pod.Name) {
				policyFound = true
				break
			}
		}

		if !policyFound {
			deletedPolicy, err := s.Vault.DeletePolicy(policy)
			if err != nil || deletedPolicy != nil {
				logging.Error(fmt.Sprintf("failed to delete orphaned policy: %s, %s", policy, err.Error()))
				continue
			}
			logging.System(fmt.Sprintf("deleted orphan policy: %s", policy))
			numberOfCleanedResource++
		}
	}

	logging.System(fmt.Sprintf("finished cleanup service and cleaned up %d resources", numberOfCleanedResource))

	return nil
}
