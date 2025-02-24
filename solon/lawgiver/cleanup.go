package lawgiver

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	"strings"
)

func (s *SolonHandler) deleteOrphans(pod *v1.Pod) error {
	// Identify orphaned Elastic users
	numberOfCleanedResource := 0

	splitPodName := strings.Split(pod.Name, "-")
	var username string
	//this logic is from periandros because elastic does not accept hyphem in a username
	if len(splitPodName) > 1 {
		username = splitPodName[0] + splitPodName[len(splitPodName)-1]
	} else {
		username = splitPodName[0]
	}

	_, err := s.Elastic.Access().DeleteUser(username)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to delete orphaned user: %s, %s", username, err.Error()))
	} else {
		logging.System(fmt.Sprintf("deleted orphan user in elastic: %s", username))
		numberOfCleanedResource++
	}

	err = s.Vault.DeleteSecret(pod.Name)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to delete orphaned secret: %s, %s", pod.Name, err.Error()))
	}

	err = s.Vault.RemoveSecret(pod.Name)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to remove orphaned secret: %s, %s", pod.Name, err.Error()))
	} else {
		logging.System(fmt.Sprintf("deleted orphan secret: %s", pod.Name))
		numberOfCleanedResource++
	}

	policy := fmt.Sprintf("policy-%s", pod.Name)

	deletedPolicy, err := s.Vault.DeletePolicy(policy)
	if err != nil || deletedPolicy != nil {
		logging.Error(fmt.Sprintf("failed to delete orphaned policy: %s, %s", policy, err.Error()))
	} else {
		logging.System(fmt.Sprintf("deleted orphan policy: %s", policy))
		numberOfCleanedResource++
	}

	logging.System(fmt.Sprintf("finished cleanup service and cleaned up %d resources", numberOfCleanedResource))

	return nil
}
