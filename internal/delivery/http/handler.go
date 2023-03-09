package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openinfradev/tks-api/internal/helper"
	"github.com/openinfradev/tks-api/pkg/log"
)

type ResponseJson struct {
	Code    int         `json:"status_code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func ErrorJSON(w http.ResponseWriter, message string, code int) {
	ResponseJSON(w, "", message, code)
}

func InternalServerError(w http.ResponseWriter, err error) {
	log.Error(fmt.Sprintf("[INTERNAL SERVER ERROR] [%s]", err.Error()))

	ErrorJSON(w, err.Error(), http.StatusInternalServerError)
}

func ResponseJSON(w http.ResponseWriter, data interface{}, message string, code int) {
	//time.Sleep(time.Second * 1) // for test

	out := ResponseJson{
		Code:    code,
		Message: message,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	log.Info(fmt.Sprintf("[API_RESPONSE] [%s]", helper.ModelToJson(out)))
	json.NewEncoder(w).Encode(out)
}

func GetSession(r *http.Request) (string, string) {
	return r.Header.Get("ID"), r.Header.Get("AccountId")
}

/*
func (h *APIHandler) GetClientFromClusterId(clusterId string) (*kubernetes.Clientset, error) {
	const prefix = "CACHE_KEY_KUBE_CLIENT_"
	value, found := h.Cache.Get(prefix + clusterId)
	if found {
		return value.(*kubernetes.Clientset), nil
	}
	client, err := helper.GetClientFromClusterId(clusterId)
	if err != nil {
		return nil, err
	}

	h.Cache.Set(prefix+clusterId, client, gcache.DefaultExpiration)
	return client, nil
}

func (h *APIHandler) GetKubernetesVserion() (string, error) {
	const prefix = "CACHE_KEY_KUBE_VERSION_"
	value, found := h.Cache.Get(prefix)
	if found {
		return value.(string), nil
	}
	version, err := helper.GetKubernetesVserion()
	if err != nil {
		return "", err
	}

	h.Cache.Set(prefix, version, gcache.DefaultExpiration)
	return version, nil
}

func (h *APIHandler) GetSession(r *http.Request) (string, string) {
	return r.Header.Get("ID"), r.Header.Get("AccountId")
}

func (h *APIHandler) AddHistory(r *http.Request, projectId string, historyType string, description string) error {
		userId, _ := h.GetSession(r)

		err := h.Repository.AddHistory(userId, projectId, historyType, description)
		if err != nil {
			log.Error(err)
			return err
		}

	return nil
}
*/
