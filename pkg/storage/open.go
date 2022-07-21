package storage

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/kfs"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"go.uber.org/zap"
	"io"
	"path/filepath"
	"syscall"
)

type Storage struct {
	// Cfg is the configuration for the storage provided to Open.
	Cfg Config
	// KV is the key-value store for the node.
	KV kv.DB
	// Cesium is the time-series engine for the node.
	Cesium cesium.DB
	// ReleaseLock is a function that releases the lock on the storage file system.
	ReleaseLock func() error
}

func (s *Storage) Close() error {
	c := errutil.NewCatchSimple(errutil.WithAggregation())
	c.Exec(s.Cesium.Close)
	c.Exec(s.KV.Close)
	c.Exec(s.ReleaseLock)
	return c.Error()
}

type Config struct {
	// Dirname defines the root directory the node will write its data to. Dirname
	// shouldn't be used by another other process while the node is running.
	Dirname string
	// MemBacked defines whether the node should use a memory-backed file system.
	MemBacked bool
	// Logger is the logger used by the node.
	Logger *zap.Logger
	// Experiment is the experiment used by the node for metrics, reports, and tracing.
	Experiment alamos.Experiment
}

func Open(cfg Config) (Storage, error) {
	// Open our two file system implementations. We use VFS for acquiring the directory
	// lock and for the key-value store. We use KFS for the time-series engine, as we
	// need seekable file handles.
	baseVFS, baseKFS := openBaseFS(cfg)

	s := Storage{}

	// Acquire the lock on the storage directory. If any other delta node is using the
	// same directory we return an error to the client.
	releaser, err := acquireLock(cfg, baseVFS)
	if err != nil {
		return s, err
	}
	// Allow the caller to release the lock when they finish using the storage.
	s.ReleaseLock = releaser.Close

	// Open the key-value storage engine.
	if s.KV, err = openKV(cfg, baseVFS); err != nil {
		return s, errors.CombineErrors(err, s.ReleaseLock())
	}

	// Open the time-series engine.
	if s.Cesium, err = openCesium(cfg, baseKFS, baseVFS, s.KV); err != nil {
		return s, errors.CombineErrors(err, s.ReleaseLock())
	}

	return s, nil
}

const (
	kvDirname     = "kv"
	lockFileName  = "LOCK"
	cesiumDirname = "cesium"
)

func openBaseFS(cfg Config) (vfs.FS, kfs.BaseFS) {
	if cfg.MemBacked {
		return vfs.NewMem(), kfs.NewMem()
	} else {
		return vfs.Default, kfs.NewOS()
	}
}

const (
	lockAlreadyAcquireMsg = `
	The storage directory is locked by another process. 
		
	Is there another Delta node using the same directory?
	`
)

func acquireLock(cfg Config, fs vfs.FS) (io.Closer, error) {
	fName := filepath.Join(cfg.Dirname, lockFileName)
	release, err := fs.Lock(fName)
	if err == nil {
		return release, nil
	}
	if err.(syscall.Errno) == syscall.EAGAIN {
		return release, errors.Wrap(err, lockAlreadyAcquireMsg)
	}
	return release, err
}

func openKV(cfg Config, fs vfs.FS) (kv.DB, error) {
	dirname := filepath.Join(cfg.Dirname, kvDirname)
	db, err := pebble.Open(dirname, &pebble.Options{FS: fs})
	return pebblekv.Wrap(db), err
}

func openCesium(
	cfg Config,
	fs kfs.BaseFS,
	vfs vfs.FS,
	kv kv.DB,
) (cesium.DB, error) {
	dirname := filepath.Join(cfg.Dirname, cesiumDirname)
	return cesium.Open(
		dirname,
		cesium.WithFS(vfs, fs),
		cesium.WithKVEngine(kv),
		cesium.WithLogger(cfg.Logger),
		cesium.WithExperiment(cfg.Experiment),
	)
}
