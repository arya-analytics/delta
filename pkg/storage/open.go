// Package storage provides entities for managing node local stores. Delta uses two
// database classes for storing its data:
//
// 	1. A key-value store (implementing the kv.DB interface) for storing cluster wide
// 	metadata.
//	2. A time-series engine (implementing the cesium.DB interface) for writing chunks
// 	of time-series data.
//
// It's important to node the storage package does NOT manage any sort of distributed
// storage implementation.
package storage

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/filter"
	"github.com/arya-analytics/x/fsutil"
	"github.com/arya-analytics/x/kfs"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/pebblekv"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

type (
	// KVEngine represents the available key-value storage engines delta can use.
	KVEngine uint8
	// TSEngine represents the available time-series storage engines delta can use.
	TSEngine uint8
)

//go:generate stringer -type=KVEngine
const (
	// PebbleKV uses cockroach's pebble key-value store.
	PebbleKV KVEngine = iota + 1
)

var kvEngines = []KVEngine{PebbleKV}

//go:generate stringer -type=TSEngine
const (
	// CesiumTS uses arya analytics' cesium time-series engine.
	CesiumTS TSEngine = iota + 1
)

var tsEngines = []TSEngine{CesiumTS}

// Store represents a node's local storage. The provided KV and TS engines can be
// used to read and write data. A Store must be closed when it is no longer in use.
type Store struct {
	// Cfg is the configuration for the storage provided to Open.
	Cfg Config
	// KV is the key-value store for the node.
	KV kv.DB
	// TS is the time-series engine for the node.
	TS cesium.DB
	// releaseLock is a function that releases the lock on the storage file system.
	releaseLock func() error
}

// Close closes the Store, releasing the lock on the storage directory. Close
// MUST be called when the Store is no longer in use. The caller must ensure that
// all processes interacting the Store have finished before calling Close.
func (s *Store) Close() error {
	// We execute with aggregation here to ensure that we close all engines and release
	// the lock regardless if one engine fails to close. This may cause unexpected
	// behavior in the future, so we need to track it.
	c := errutil.NewCatchSimple(errutil.WithAggregation())
	c.Exec(s.TS.Close)
	c.Exec(s.KV.Close)
	c.Exec(s.releaseLock)
	return c.Error()
}

// Config is used to configure delta's storage layer.
type Config struct {
	// Dirname defines the root directory the Store resides. The given directory
	// shouldn't be used by another process while the node is running.
	Dirname string
	// Perm is the file permissions to use for the storage directory.
	Perm fs.FileMode
	// MemBacked defines whether the node should use a memory-backed file system.
	MemBacked bool
	// Logger is the logger used by the node.
	Logger *zap.Logger
	// Experiment is the experiment used by the node for metrics, reports, and tracing.
	Experiment alamos.Experiment
	// KVEngine is the key-value engine storage will use.
	KVEngine KVEngine
	// TSEngine is the time-series engine storage will use.
	TSEngine TSEngine
}

func (cfg Config) Merge(other Config) Config {
	if cfg.Dirname == "" {
		cfg.Dirname = other.Dirname
	}
	if cfg.Perm == 0 {
		cfg.Perm = other.Perm
	}
	if cfg.Logger == nil {
		cfg.Logger = other.Logger
	}
	if cfg.Experiment == nil {
		cfg.Experiment = other.Experiment
	}
	if cfg.KVEngine == 0 {
		cfg.KVEngine = other.KVEngine
	}
	if cfg.TSEngine == 0 {
		cfg.TSEngine = other.TSEngine
	}
	return cfg
}

func (cfg Config) Validate() error {
	if !cfg.MemBacked && cfg.Dirname == "" {
		return errors.New("[storage] - dirname must be set")
	}

	if !filter.ElementOf[KVEngine](kvEngines, cfg.KVEngine) {
		return errors.Newf("[storage] - key-value engine must be one of: %v", kvEngines)
	}

	if !filter.ElementOf[TSEngine](tsEngines, cfg.TSEngine) {
		return errors.Newf("[storage] - time-series engine must be one of: %v", tsEngines)
	}

	if cfg.TSEngine == 0 {
		return errors.New("[storage] - time-series engine must be provided")
	}

	if cfg.Perm == 0 {
		return errors.New("[storage] - insufficient permission bits configured")
	}

	return nil
}

// DefaultConfig returns the default configuration for the storage layer.
func DefaultConfig() Config {
	return Config{
		Perm:       fsutil.OS_USER_RW,
		MemBacked:  false,
		Logger:     zap.NewNop(),
		Experiment: nil,
		KVEngine:   PebbleKV,
		TSEngine:   CesiumTS,
	}
}

// Open opens a new Store with the given Config. Open acquires a lock on the directory
// specified in the Config. If the lock cannot be acquired, Open returns an error.
// The lock is released when the Store is/closed. Store MUST be closed when it is no
// longer in use.
func Open(cfg Config) (s Store, err error) {
	cfg = cfg.Merge(DefaultConfig())
	if err := cfg.Validate(); err != nil {
		return s, err
	}

	// Open our two file system implementations. We use VFS for acquiring the directory
	// lock and for the key-value store. We use KFS for the time-series engine, as we
	// need seekable file handles.
	baseVFS, baseKFS := openBaseFS(cfg)

	// Configure our storage directory with the correct permissions.
	if err := configureStorageDir(cfg, baseVFS); err != nil {
		return s, err
	}

	// Acquire the lock on the storage directory. If any other delta node is using the
	// same directory we return an error to the client.
	releaser, err := acquireLock(cfg, baseVFS)
	if err != nil {
		return s, err
	}
	// Allow the caller to release the lock when they finish using the storage.
	s.releaseLock = releaser.Close

	// Open the key-value storage engine.
	if s.KV, err = openKV(cfg, baseVFS); err != nil {
		return s, errors.CombineErrors(err, s.releaseLock())
	}

	// Open the time-series engine.
	if s.TS, err = openTS(cfg, baseKFS, baseVFS, s.KV); err != nil {
		return s, errors.CombineErrors(err, s.releaseLock())
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

func configureStorageDir(cfg Config, vfs vfs.FS) error {
	if err := vfs.MkdirAll(cfg.Dirname, cfg.Perm); err != nil {
		return errors.Wrapf(
			err,
			"failed to create storage directory %s",
			cfg.Dirname,
		)
	}
	return validateSufficientDirPermissions(cfg)
}

const insufficientDirPermissions = `
    Existing storage directory has permissions

	%v

	Delta requires the storage directory to have at least 

	%v

	permissions.
`

func validateSufficientDirPermissions(cfg Config) error {
	stat, err := os.Stat(cfg.Dirname)
	if err != nil {
		return err
	}
	// We need the director to have at least the permissions set in Config.Perm.
	if stat.Mode().Perm()&fsutil.OS_OTH_RWX != cfg.Perm {
		return errors.Newf(insufficientDirPermissions, stat.Mode().Perm(), cfg.Perm)
	}
	return nil
}

const lockAlreadyAcquiredMsg = `
	The storage directory 

		%s

	is locked by another process. 
		
	Is there another Delta node using the same directory?
`

func acquireLock(cfg Config, fs vfs.FS) (io.Closer, error) {
	fName := filepath.Join(cfg.Dirname, lockFileName)
	release, err := fs.Lock(fName)
	if err == nil {
		return release, nil
	}
	if err.(syscall.Errno) == syscall.EAGAIN {
		return release, errors.Newf(lockAlreadyAcquiredMsg, cfg.Dirname)
	}
	return release, errors.Wrapf(
		err,
		"[storage] - failed to acquire lock on %s",
		fName,
	)
}

func openKV(cfg Config, fs vfs.FS) (kv.DB, error) {
	dirname := filepath.Join(cfg.Dirname, kvDirname)
	db, err := pebble.Open(dirname, &pebble.Options{FS: fs})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"[storage] - failed to open key-value store in %s",
			dirname,
		)
	}
	return pebblekv.Wrap(db), err
}

func openTS(
	cfg Config,
	fs kfs.BaseFS,
	vfs vfs.FS,
	kv kv.DB,
) (cesium.DB, error) {
	if cfg.TSEngine != CesiumTS {
		return nil, errors.Newf("[storage] - unsupported time-series engine: %v", cfg.TSEngine)
	}
	dirname := filepath.Join(cfg.Dirname, cesiumDirname)
	return cesium.Open(
		dirname,
		cesium.WithFS(vfs, fs),
		cesium.WithKVEngine(kv),
		cesium.WithLogger(cfg.Logger),
		cesium.WithExperiment(cfg.Experiment),
	)
}
