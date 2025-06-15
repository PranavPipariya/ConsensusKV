package raftnode

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/your/module/internal/fsm"
)

type Store struct {
	Raft *raft.Raft
	FSM  *fsm.FSM
}

func NewStore(nodeAddr, dataDir string, inmem bool) (*Store, raft.Transport, error) {
	store := &Store{FSM: fsm.New()}

	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeAddr)

	var logStore raft.LogStore
	var stableStore raft.StableStore
	var snapshotStore raft.SnapshotStore
	var err error

	if inmem {
		logStore = raft.NewInmemStore()
		stableStore = raft.NewInmemStore()
		snapshotStore = raft.NewInmemSnapshotStore()
	} else {
		boltPath := filepath.Join(dataDir, "raft.db")
		boltDB, err := raftboltdb.NewBoltStore(boltPath)
		if err != nil {
			return nil, nil, fmt.Errorf("bolt store: %s", err)
		}
		logStore = boltDB
		stableStore = boltDB
		snapshotStore, err = raft.NewFileSnapshotStore(dataDir, 1, os.Stderr)
		if err != nil {
			return nil, nil, fmt.Errorf("snapshot store: %s", err)
		}
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", nodeAddr)
	if err != nil {
		return nil, nil, err
	}

	transport, err := raft.NewTCPTransport(nodeAddr, tcpAddr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	r, err := raft.NewRaft(config, store.FSM, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, nil, fmt.Errorf("raft init: %s", err)
	}

	store.Raft = r

	return store, transport, nil
}

func (s *Store) BootstrapSelf(selfAddr string) error {
	cfg := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(selfAddr),
				Address: raft.ServerAddress(selfAddr),
			},
		},
	}
	return s.Raft.BootstrapCluster(cfg).Error()
}
