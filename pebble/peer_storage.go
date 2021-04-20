package pebble

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/pebble"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"

	"github.com/gotd/contrib/internal/bytesconv"
	"github.com/gotd/contrib/storage"
)

// PeerStorage is a peer resolver cache.
type PeerStorage struct {
	pebble *pebble.DB
}

// NewPeerStorage creates new peer storage using pebble.
func NewPeerStorage(db *pebble.DB) *PeerStorage {
	return &PeerStorage{pebble: db}
}

// Add adds given peer to the storage.
func (r PeerStorage) Add(ctx context.Context, value storage.Peer) error {
	data, err := json.Marshal(value)
	if err != nil {
		return xerrors.Errorf("marshal: %w", err)
	}

	id := storage.KeyFromPeer(value).Bytes(nil)
	if err := r.pebble.Set(id, data, nil); err != nil {
		return xerrors.Errorf("set id <-> data: %w", err)
	}

	return nil
}

// Find finds peer using given key.
func (r PeerStorage) Find(ctx context.Context, key storage.Key) (_ storage.Peer, rerr error) {
	id := key.Bytes(nil)

	data, closer, err := r.pebble.Get(id)
	if err != nil {
		if xerrors.Is(err, pebble.ErrNotFound) {
			return storage.Peer{}, storage.ErrPeerNotFound
		}
		return storage.Peer{}, xerrors.Errorf("get %q: %w", id, err)
	}
	defer func() {
		multierr.AppendInto(&rerr, closer.Close())
	}()

	var b storage.Peer
	if err := json.Unmarshal(data, &b); err != nil {
		return storage.Peer{}, xerrors.Errorf("unmarshal: %w", err)
	}

	return b, nil
}

// Assign adds given peer to the storage and associate it to the given key.
func (r PeerStorage) Assign(ctx context.Context, key string, value storage.Peer) (rerr error) {
	data, err := json.Marshal(value)
	if err != nil {
		return xerrors.Errorf("marshal: %w", err)
	}

	b := r.pebble.NewBatch()
	defer func() {
		multierr.AppendInto(&rerr, b.Close())
	}()

	id := storage.KeyFromPeer(value).Bytes(nil)
	if err := b.Set(id, data, nil); err != nil {
		return xerrors.Errorf("set id <-> data: %w", err)
	}

	if err := b.Set(bytesconv.S2B(key), id, nil); err != nil {
		return xerrors.Errorf("set key <-> id: %w", err)
	}

	if err := b.Commit(nil); err != nil {
		return xerrors.Errorf("commit changes: %w", err)
	}

	return nil
}

// Resolve finds peer using associated key.
func (r PeerStorage) Resolve(ctx context.Context, key string) (_ storage.Peer, rerr error) {
	// Convert key string to the byte slice.
	// Pebble copies key, so we can use unsafe conversion here.
	k := bytesconv.S2B(key)

	// Create database snapshot.
	snap := r.pebble.NewSnapshot()
	defer func() {
		multierr.AppendInto(&rerr, snap.Close())
	}()

	// Find id by key.
	id, idCloser, err := snap.Get(k)
	if err != nil {
		if xerrors.Is(err, pebble.ErrNotFound) {
			return storage.Peer{}, storage.ErrPeerNotFound
		}
		return storage.Peer{}, xerrors.Errorf("get %q: %w", k, err)
	}
	defer func() {
		multierr.AppendInto(&rerr, idCloser.Close())
	}()

	// Find object by id.
	data, dataCloser, err := snap.Get(id)
	if err != nil {
		if xerrors.Is(err, pebble.ErrNotFound) {
			return storage.Peer{}, storage.ErrPeerNotFound
		}
		return storage.Peer{}, xerrors.Errorf("get %q: %w", id, err)
	}
	defer func() {
		multierr.AppendInto(&rerr, dataCloser.Close())
	}()

	var b storage.Peer
	if err := json.Unmarshal(data, &b); err != nil {
		return storage.Peer{}, xerrors.Errorf("unmarshal: %w", err)
	}

	return b, nil
}
