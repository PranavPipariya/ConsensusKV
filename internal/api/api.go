package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	"github.com/your/module/internal/raftnode"
)

type API struct {
	Store *raftnode.Store
	Addr  string
}

func New(store *raftnode.Store, addr string) *API {
	return &API{Store: store, Addr: addr}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/get", a.handleGet)
	mux.HandleFunc("/set", a.handleSet)
	mux.HandleFunc("/delete", a.handleDelete)
	mux.HandleFunc("/join", a.handleJoin)
	mux.HandleFunc("/status", a.handleStatus)
}

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}
	val, found := a.Store.FSM.Get(key)
	if !found {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{key: val})
}

func (a *API) handleSet(w http.ResponseWriter, r *http.Request) {
	if a.Store.Raft.State() != raft.Leader {
		leader := string(a.Store.Raft.Leader())
		leaderURL := "http://" + strings.Replace(leader, ":5000", ":8000", 1) + "/set"

		bodyBytes, _ := io.ReadAll(r.Body)
		proxyReq, _ := http.NewRequest(r.Method, leaderURL, bytes.NewReader(bodyBytes))
		proxyReq.Header = r.Header

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	type cmd struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var c cmd
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, _ := json.Marshal(map[string]string{"op": "set", "key": c.Key, "value": c.Value})
	future := a.Store.Raft.Apply(data, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleDelete(w http.ResponseWriter, r *http.Request) {
	if a.Store.Raft.State() != raft.Leader {
		leader := string(a.Store.Raft.Leader())
		leaderURL := "http://" + strings.Replace(leader, ":5000", ":8000", 1) + "/delete"

		bodyBytes, _ := io.ReadAll(r.Body)
		proxyReq, _ := http.NewRequest(r.Method, leaderURL, bytes.NewReader(bodyBytes))
		proxyReq.Header = r.Header

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	type cmd struct {
		Key string `json:"key"`
	}
	var c cmd
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, _ := json.Marshal(map[string]string{"op": "delete", "key": c.Key})
	future := a.Store.Raft.Apply(data, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleJoin(w http.ResponseWriter, r *http.Request) {
	peerAddr := r.URL.Query().Get("peerAddress")
	if peerAddr == "" {
		http.Error(w, "missing peerAddress", http.StatusBadRequest)
		return
	}
	f := a.Store.Raft.AddVoter(raft.ServerID(peerAddr), raft.ServerAddress(peerAddr), 0, 0)
	if err := f.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"isLeader": a.Store.Raft.State() == raft.Leader,
		"leader":   a.Store.Raft.Leader(),
		"term":     a.Store.Raft.Stats()["term"],
	}
	json.NewEncoder(w).Encode(stats)
}
