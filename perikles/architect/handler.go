package architect

import (
	"encoding/json"
	"fmt"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/plato/certificates"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
	"io"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sync"
	"time"
)

type PeriklesHandler struct {
	Mutex              sync.Mutex
	PendingUpdateTimer time.Duration
	TLSCheckTimer      time.Duration
	ReconcileTimer     time.Duration
	PendingUpdates     map[string][]MappingUpdate
	Kube               *thales.KubeClient
	Mapping            odysseia.ServiceMapping
	Cert               certificates.CertClient
	CiliumClient       *versioned.Clientset
	Namespace          string
	CrdName            string
	TLSFiles           string
	L7Mode             bool
}

// pingPong pongs the ping
func (p *PeriklesHandler) pingPong(w http.ResponseWriter, req *http.Request) {
	pingPong := models.ResultModel{Result: "pong"}
	middleware.ResponseWithJson(w, pingPong)
}

func (p *PeriklesHandler) validate(w http.ResponseWriter, req *http.Request) {
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)

	var body []byte
	if req.Body != nil {
		if data, err := io.ReadAll(req.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: requestId},
			Messages: []models.ValidationMessages{
				{
					Field:   "body",
					Message: "request body was empty",
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	arRequest := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &arRequest); err != nil {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: requestId},
			Messages: []models.ValidationMessages{
				{
					Field:   "body",
					Message: "incorrect body was sent: cannot unmarshal request into AdmissionReview",
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	if arRequest.Request == nil {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: requestId},
			Messages: []models.ValidationMessages{
				{
					Field:   "admission request",
					Message: "cannot work with a nil request in an AdmissionReview",
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	kubeType := arRequest.Request.RequestKind.Kind
	raw := arRequest.Request.Object.Raw

	// buffered channel for goroutine error reporting
	errCh := make(chan error, 2)
	wg := &sync.WaitGroup{}

	switch kubeType {
	case "Deployment":
		deploy := v1.Deployment{}
		if err := json.Unmarshal(raw, &deploy); err != nil {
			e := models.ValidationError{
				ErrorModel: models.ErrorModel{UniqueCode: requestId},
				Messages: []models.ValidationMessages{
					{
						Field:   "body",
						Message: "incorrect body was sent: cannot unmarshal request into Deployment",
					},
				},
			}
			middleware.ResponseWithJson(w, e)
			return
		}

		wg.Add(2)
		go func() {
			defer wg.Done()
			err := p.checkForAnnotations(&deploy)
			if err != nil {
				errCh <- fmt.Errorf("checkForAnnotations error: %w", err)
			}
		}()

		go func() {
			defer wg.Done()
			err := p.checkForElasticAnnotations(&deploy, nil)
			if err != nil {
				errCh <- fmt.Errorf("checkForElasticAnnotations error: %w", err)
			}
		}()
	case "Job":
		job := batchv1.Job{}
		if err := json.Unmarshal(raw, &job); err != nil {
			e := models.ValidationError{
				ErrorModel: models.ErrorModel{UniqueCode: requestId},
				Messages: []models.ValidationMessages{
					{
						Field:   "body",
						Message: "incorrect body was sent: cannot unmarshal request into Job",
					},
				},
			}
			middleware.ResponseWithJson(w, e)
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := p.checkForElasticAnnotations(nil, &job)
			if err != nil {
				errCh <- fmt.Errorf("checkForElasticAnnotations error: %w", err)
			}
		}()
	}

	// Send the AdmissionReview response immediately
	review := v1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       arRequest.Kind,
			APIVersion: arRequest.APIVersion,
		},
		Response: &v1beta1.AdmissionResponse{
			UID:     arRequest.Request.UID,
			Allowed: true,
		},
	}

	middleware.ResponseWithCustomCode(w, 200, review)

	// Wait for background processing to complete
	go func() {
		wg.Wait()
		close(errCh)

		// Log errors after completion
		for err := range errCh {
			logging.Error(fmt.Sprintf("Request ID: %s, Error: %v", requestId, err))
		}
	}()
}
