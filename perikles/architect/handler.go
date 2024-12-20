package architect

import (
	"encoding/json"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/agora/plato/models"
	"io"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"sync"
)

type PeriklesHandler struct {
	Config         *Config
	Mutex          sync.Mutex
	PendingUpdates map[string][]MappingUpdate
}

// pingPong pongs the ping
func (p *PeriklesHandler) pingPong(w http.ResponseWriter, req *http.Request) {
	pingPong := models.ResultModel{Result: "pong"}
	middleware.ResponseWithJson(w, pingPong)
}

// validate that new deployments have the correct secret attached to them
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
					Message: "incorrect body was send: cannot unmarshal request into AdmissionReview",
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

	switch kubeType {
	case "Deployment":
		deploy := v1.Deployment{}
		if err := json.Unmarshal(raw, &deploy); err != nil {
			e := models.ValidationError{
				ErrorModel: models.ErrorModel{UniqueCode: requestId},
				Messages: []models.ValidationMessages{
					{
						Field:   "body",
						Message: "incorrect body was send: cannot unmarshal request into Deployment",
					},
				},
			}
			middleware.ResponseWithJson(w, e)
			return
		}

		go func() {
			err := p.checkForAnnotations(&deploy, nil)
			if err != nil {
				logging.Error(err.Error())
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
						Message: "incorrect body was send: cannot unmarshal request into Job",
					},
				},
			}
			middleware.ResponseWithJson(w, e)
			return
		}

		go func() {
			if !strings.Contains(job.Name, "mirrord") {
				err := p.checkForAnnotations(nil, &job)
				if err != nil {
					logging.Error(err.Error())
				}
			}
		}()
	}

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
}
