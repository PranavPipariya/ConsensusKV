package fsm

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

type Command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type FSM struct {
	mu sync.RWMutex
	db map[string]string
}

func New() *FSM {
	return &FSM{db: make(map[string]string)}
}

func (f *FSM) Apply(logEntry *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cmd Command
	err := json.Unmarshal(logEntry.Data, &cmd)
	if err != nil {
		panic(fmt.Sprintf("unmarshal fail: %v", err))
	}

	if cmd.Op == "set" {
		f.db[cmd.Key] = cmd.Value
	} else if cmd.Op == "delete" {
		delete(f.db, cmd.Key)
	} else {
		panic(fmt.Sprintf("invalid op: %s", cmd.Op))
	}

	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	clone := make(map[string]string)
	for k, v := range f.db {
		clone[k] = v
	}
	return &fsmSnapshot{store: clone}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	snapshot := make(map[string]string)
	err := json.NewDecoder(rc).Decode(&snapshot)
	if err != nil {
		return err
	}
	f.db = snapshot
	return nil
}

type fsmSnapshot struct {
	store map[string]string
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		b, err := json.Marshal(s.store)
		if err != nil {
			return err
		}
		_, err = sink.Write(b)
		if err != nil {
			return err
		}
		return sink.Close()
	}()
	if err != nil {
		sink.Cancel()
	}
	return err
}

func (s *fsmSnapshot) Release() {}

func (f *FSM) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	val, ok := f.db[key]
	return val, ok
}
