package types

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/klog/v2"
)

var (
	runtimeSchema = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeSchema)
	deserializer  = codecFactory.UniversalDeserializer()
)

type WebhookServerParams struct {
	Port     int64  `json:"port"`
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

type WebhookServer struct {
	Server              *http.Server
	WhiteListRegistries []string
}

type patchOperation struct {
	OP    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (ws *WebhookServer) Handler(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty data body")
		http.Error(w, "empty data body", http.StatusBadRequest)
		return
	}
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		klog.Error("content type is not application/json")
		http.Error(w, "content type is invalid", http.StatusBadRequest)
		return
	}

	var requestAdmissionReview admissionv1.AdmissionReview
	var admissionResponse *admissionv1.AdmissionResponse
	_, _, err := deserializer.Decode(body, nil, &requestAdmissionReview)
	if err != nil {
		klog.Errorf("failed deserializer decode body to admission review: %v", err)
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			},
		}
	} else {
		if strings.HasPrefix(r.URL.Path, "/mutate") {
			admissionResponse = ws.mutate(&requestAdmissionReview)
		} else if strings.HasPrefix(r.URL.Path, "/validate") {
			admissionResponse = ws.validate(&requestAdmissionReview)
		}
	}

	responseAdmissionReview := admissionv1.AdmissionReview{}
	responseAdmissionReview.APIVersion = requestAdmissionReview.APIVersion
	responseAdmissionReview.Kind = requestAdmissionReview.Kind
	if admissionResponse != nil {
		responseAdmissionReview.Response = admissionResponse
		if requestAdmissionReview.Request != nil {
			responseAdmissionReview.Response.UID = requestAdmissionReview.Request.UID
		}
	}
	klog.Infof("responseAdmissionReview response : %v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Errorf("json marshal responseAdmissionReview is err : %v", err)
		http.Error(w, "json marshal responseAdmissionReview is err", http.StatusBadRequest)
	}
	klog.Info("ready to write resp")
	if _, err := w.Write(respBytes); err != nil {
		klog.Errorf("cannot write respBytes is err : %v", err)
		http.Error(w, "cannot write respBytes is err ", http.StatusBadRequest)
	}
}

func (ws *WebhookServer) validate(requestAdmissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	req := requestAdmissionReview.Request
	var (
		allowed = true
		code    = http.StatusOK
		message = ""
	)
	klog.Infof("requestAdmissionReview request : %+v", req)

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Infof("json unmarshal req object raw is err : %v", err)
		code = http.StatusBadRequest
		allowed = false
		message = err.Error()
	} else {
		for _, container := range pod.Spec.Containers {
			var whitelisted = false
			for _, reg := range ws.WhiteListRegistries {
				if strings.HasPrefix(container.Image, reg) {
					whitelisted = true
				}
				if !whitelisted {
					allowed = false
					code = http.StatusForbidden
					message = fmt.Sprintf("%s image comes from an untrusted registry! Only images from %v are allowed.", container.Image, ws.WhiteListRegistries)
					break
				}
			}
		}
	}
	return &admissionv1.AdmissionResponse{
		Allowed: allowed,
		Result: &metav1.Status{
			Code:    int32(code),
			Message: message,
		},
	}
}

const (
	AnnotationMutateKey  = "bnblak.com.admission-registry/mutate"
	AnnotationStateKey   = "bnblak.com.admission-registry/state"
	AnnotationStateValue = "mutated"
)

func (ws *WebhookServer) mutate(requestAdmissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	req := requestAdmissionReview.Request
	var (
		objectMeta *metav1.ObjectMeta
	)
	klog.Infof("requestAdmissionReview request : %+v", req)
	switch req.Kind.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			klog.Infof("json unmarshal req object raw to deploy is err : %v", err)
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
		}
		objectMeta = &deployment.ObjectMeta
	case "Service":
		var service corev1.Service
		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			klog.Infof("json unmarshal req object raw to svc is err : %v", err)
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				},
			}
		}
		objectMeta = &service.ObjectMeta

	default:
	}
	if !mutationRequired(objectMeta) {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}
	var patchs []patchOperation
	patchs = append(patchs,
		mutateAnnotations(
			objectMeta.GetAnnotations(),
			map[string]string{
				AnnotationStateKey: AnnotationStateValue,
			})...,
	)
	patchsBytes, err := json.Marshal(patchs)
	if err != nil {
		klog.Errorf("json marshal patchs is err : %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			},
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchsBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func mutationRequired(md *metav1.ObjectMeta) bool {
	targetAnno := md.GetAnnotations()
	if targetAnno == nil {
		targetAnno = map[string]string{}
	}
	required := false

	mutateValue, ok := targetAnno[AnnotationMutateKey]
	if !ok {
		return false
	}
	switch mutateValue {
	case "n", "no", "false", "off":
		return false
	default:
		required = true
	}
	if targetAnno[AnnotationStateKey] == AnnotationStateValue {
		required = false
	}
	klog.Infof("mutation policy for %s/%s: required: %v", md.Name, md.Namespace, required)
	return required
}

func mutateAnnotations(target map[string]string, annotations map[string]string) (patchs []patchOperation) {
	if target == nil {
		target = map[string]string{}
	}
	for k, v := range annotations {
		if target[k] == "" {
			patchs = append(patchs, patchOperation{
				OP:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					k: v,
				},
			})
		} else {
			patchs = append(patchs, patchOperation{
				OP:    "replace",
				Path:  "/metadata/annotations/" + k,
				Value: v,
			})
		}
	}
	return
}
