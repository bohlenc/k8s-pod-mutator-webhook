package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s-pod-mutator-webhook/internal/admission_review"
	"k8s-pod-mutator-webhook/internal/logger"
	"k8s-pod-mutator-webhook/pkg/mutator"
	admissionv1 "k8s.io/api/admission/v1"
	"net/http"
	"reflect"
)

const readyPath = "/ready"
const mutatePath = "/mutate"

type ServerSettings struct {
	Port        int
	Tls         bool
	TlsCertFile string
	TlsKeyFile  string
}

type Server struct {
	settings   ServerSettings
	httpServer http.Server
}

func CreateServer(settings ServerSettings, mutator mutator.Mutator) (*Server, error) {
	logger.Logger.WithFields(logrus.Fields{
		"settings": settings,
	}).Infoln("creating server")

	serveMux := http.NewServeMux()

	serveMux.HandleFunc(readyPath, readyHandleFunc)
	logger.Logger.Debugf("setup handler for %v", readyPath)

	serveMux.HandleFunc(mutatePath, mutateHandleFunc(mutator))
	logger.Logger.Debugf("setup handler for %v", mutatePath)

	server := Server{
		settings: settings,
		httpServer: http.Server{
			Addr:    fmt.Sprintf(":%v", settings.Port),
			Handler: serveMux,
		},
	}
	return &server, nil
}

func readyHandleFunc(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.WriteHeader(204)
}

func mutateHandleFunc(mutator mutator.Mutator) func(responseWriter http.ResponseWriter, request *http.Request) {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		logger.Logger.Debugln("handling mutation request")

		contentType := request.Header.Get("Content-Type")
		if contentType != "application/json" {
			logger.Logger.WithFields(logrus.Fields{
				"expected": "application/json",
				"actual":   contentType,
			}).Errorln("invalid header 'Content-Type'")
			http.Error(responseWriter, "Content-Type must be 'application/json'", http.StatusUnsupportedMediaType)
			return
		}

		var body []byte
		if request.Body != nil {
			if data, err := ioutil.ReadAll(request.Body); err == nil {
				body = data
			}
		}
		if len(body) == 0 {
			logger.Logger.WithFields(logrus.Fields{
				"body": string(body),
			}).Errorln("invalid body")
			http.Error(responseWriter, "body must not be empty", http.StatusBadRequest)
			return
		}

		var admissionResponse *admissionv1.AdmissionResponse
		reviewRequest := admissionv1.AdmissionReview{}
		if err := json.Unmarshal(body, &reviewRequest); err != nil {
			logger.Logger.WithFields(logrus.Fields{
				"error": err,
				"type":  reflect.TypeOf(reviewRequest),
			}).Errorln("decode failed")
			admissionResponse = admission_review.ErrorResponse(err)
		} else {
			admissionResponse = mutator.Mutate(reviewRequest.Request)
		}

		admissionResponse.UID = reviewRequest.Request.UID

		reviewResponse := admissionv1.AdmissionReview{
			TypeMeta: reviewRequest.TypeMeta,
			Response: admissionResponse,
		}

		response, err := json.Marshal(reviewResponse)
		if err != nil {
			logger.Logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorln("encode failed")
			http.Error(responseWriter, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(200)
		if _, err := responseWriter.Write(response); err != nil {
			logger.Logger.WithFields(logrus.Fields{
				"error": err,
			}).Errorln("write failed")
			http.Error(responseWriter, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
		}

		logger.Logger.Debugln("handled mutation request")
	}
}

func (s *Server) Start() error {
	if !s.settings.Tls {
		logger.Logger.WithFields(logrus.Fields{
			"port": s.settings.Port,
			"tls":  "disabled",
		}).Infoln("starting server")
		return s.httpServer.ListenAndServe()
	}

	logger.Logger.WithFields(logrus.Fields{
		"port": s.settings.Port,
		"tls":  "enabled",
	}).Infoln("starting server")
	return s.httpServer.ListenAndServeTLS(s.settings.TlsCertFile, s.settings.TlsKeyFile)
}

func (s *Server) Stop() error {
	logger.Logger.Infoln("stopping server")
	return s.httpServer.Shutdown(context.Background())
}
