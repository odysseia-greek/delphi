package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/pkg/errors"
	v1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (k *KleisthenesHandler) vault() error {
	logging.Debug("Setting up TLS for Vault")

	// Step 1: Generate Key Pair
	privateKey, err := k.generatePrivateKey()
	if err != nil {
		return errors.Wrap(err, "failed to generate private key")
	}

	logging.Debug("Generated key")

	// Step 2: Generate Certificate Signing Request (CSR)
	csr, err := k.generateCSR("vault", k.namespace, privateKey)
	if err != nil {
		return errors.Wrap(err, "failed to generate CSR")
	}

	logging.Debug("CSR has been generated")

	// Step 3: Create and Approve CSR in Kubernetes
	csrName := "vault-csr"
	if err := k.createAndApproveCSR(csrName, csr); err != nil {
		return errors.Wrap(err, "failed to create and approve CSR")
	}

	logging.Debug("CSR has been created and approved")

	// Step 4: Get and Decode Approved Certificate
	certData, err := k.getAndDecodeCertificate(csrName)
	if err != nil {
		return errors.Wrap(err, "failed to get and decode certificate")
	}

	ca, err := k.getClusterCACertificate()
	if err != nil {
		return err
	}

	logging.Debug("Got ca crt and key from kube and csr")

	key, err := k.privateKeyToString(privateKey)
	if err != nil {
		return err
	}

	// Step 5: Create Secret with Certificate Data
	secretName := "vault-server-tls"
	data := make(map[string][]byte)
	data["vault.crt"] = certData
	data["vault.key"] = key

	if strings.Contains(string(ca), "-----BEGIN CERTIFICATE-----") {
		data["vault.ca"] = ca
	} else {
		decodedCa, err := base64.StdEncoding.DecodeString(string(ca))
		if err != nil {
			logging.Error(err.Error())
		}
		data["vault.ca"] = decodedCa
	}

	logging.Debug(fmt.Sprintf("vault.crt: %s", string(certData)))
	logging.Debug(fmt.Sprintf("vault.ca: %s", string(ca)))
	logging.Debug(fmt.Sprintf("vault.key: %s", string(key)))

	if err := k.createSecret(secretName, data, corev1.SecretTypeOpaque); err != nil {
		return errors.Wrap(err, "failed to create secret")
	}

	logging.Debug("Created secret")
	logging.Debug("Finished setting up TLS for Vault")
	return nil
}

func (k *KleisthenesHandler) generatePrivateKey() (*ecdsa.PrivateKey, error) {
	// Generate a private key using Go's crypto library.
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func (k *KleisthenesHandler) privateKeyToString(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	block, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: block,
	}

	// Convert PEM block to string.
	privateKeyString := pem.EncodeToMemory(privateKeyPEM)
	if privateKeyString == nil {
		return nil, errors.New("failed to convert private key to string")
	}
	return privateKeyString, nil
}

func (k *KleisthenesHandler) generateCSR(service, namespace string, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	// Subject Alternative Names (SANs) for the CSR
	altNames := fmt.Sprintf("DNS.1 = %s", service)
	altNames += fmt.Sprintf("\nDNS.2 = %s.%s", service, namespace)
	altNames += fmt.Sprintf("\nDNS.3 = %s.%s.svc", service, namespace)
	altNames += fmt.Sprintf("\nDNS.4 = %s.%s.svc.cluster.local", service, namespace)
	altNames += fmt.Sprintf("\nDNS.5 = %s-0.vault-internal", service)
	altNames += fmt.Sprintf("\nDNS.6 = %s-1.vault-internal", service)
	altNames += fmt.Sprintf("\nDNS.7 = %s-2.vault-internal", service)

	// Create a certificate signing request (CSR) using Go's crypto/x509 library.
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "system:node:" + service + "." + namespace + ".svc",
			Organization: []string{"system:nodes"},
		},
		DNSNames: []string{
			service,
			fmt.Sprintf("%s.%s", service, namespace),
			fmt.Sprintf("%s.%s.svc", service, namespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", service, namespace),
			fmt.Sprintf("%s-0.vault-internal", service),
			fmt.Sprintf("%s-1.vault-internal", service),
			fmt.Sprintf("%s-2.vault-internal", service),
			"IP.1 = 127.0.0.1",
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return nil, err
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	return csrPEM, nil
}

func (k *KleisthenesHandler) createAndApproveCSR(csrName string, csrData []byte) error {
	// Create a CertificateSigningRequest object.
	csr := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{Name: csrName},
		Spec: v1.CertificateSigningRequestSpec{
			Request:    csrData,
			Usages:     []v1.KeyUsage{"digital signature", "key encipherment", "server auth"},
			SignerName: "kubernetes.io/kubelet-serving",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	csrExists := true
	_, err := k.Kube.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			csrExists = false
		}
	}
	if csrExists {
		err = k.Kube.CertificatesV1().CertificateSigningRequests().Delete(ctx, csrName, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// Create the CSR in Kubernetes.
	_, err = k.Kube.CertificatesV1().CertificateSigningRequests().Create(ctx, csr, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Retrieve the CSR after creation.
	csr, err = k.Kube.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Approve the CSR.
	csr.Status.Conditions = []v1.CertificateSigningRequestCondition{
		{
			Type:           v1.CertificateApproved,
			Reason:         "AutoApproved",
			Message:        "This CSR has been auto-approved by TLSManager.",
			LastUpdateTime: metav1.Now(),
			Status:         corev1.ConditionTrue,
		},
	}

	_, err = k.Kube.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, csrName, csr, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (k *KleisthenesHandler) getAndDecodeCertificate(csrName string) ([]byte, error) {
	logging.Debug("sleeping 10 seconds to wait for cert to arrive")
	time.Sleep(10 * time.Second)
	logging.Debug("slept")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	csr, err := k.Kube.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if !isCSRApproved(csr) {
		return nil, errors.New("CSR has not been approved yet")
	}

	return csr.Status.Certificate, nil
}

func isCSRApproved(csr *v1.CertificateSigningRequest) bool {
	for _, condition := range csr.Status.Conditions {
		if condition.Type == v1.CertificateApproved {
			return true
		}
	}
	return false
}

func (k *KleisthenesHandler) getClusterCACertificate() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	caBundle, err := k.Kube.CoreV1().ConfigMaps("kube-system").Get(ctx, "extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	caCertificateData := caBundle.Data["client-ca-file"]

	return []byte(caCertificateData), nil
}