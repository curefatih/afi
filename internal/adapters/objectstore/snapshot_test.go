package objectstore_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/objectstore"
	"github.com/curefatih/afi/internal/snapshot"
)

type memBlob struct {
	objects map[string][]byte
}

func (m *memBlob) Put(_ context.Context, key string, body io.Reader, _ int64, _ objectstore.PutOptions) error {
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.objects[key] = b
	return nil
}

func (m *memBlob) Get(_ context.Context, key string) ([]byte, error) {
	b, ok := m.objects[key]
	if !ok {
		return nil, io.EOF
	}
	return b, nil
}

func (m *memBlob) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func TestSnapshotStorePutLatestWatch(t *testing.T) {
	blob := &memBlob{}
	store := objectstore.NewSnapshotStore(blob, "snapshots")
	snap := &snapshot.Snapshot{Version: 3, APIKeys: map[string]snapshot.APIKey{}}
	if _, err := store.Put(context.Background(), snap); err != nil {
		t.Fatal(err)
	}
	got, err := store.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 3 {
		t.Fatalf("version=%d", got.Version)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen := make(chan int64, 1)
	go func() {
		_ = store.Watch(ctx, 20*time.Millisecond, func(s *snapshot.Snapshot) {
			select {
			case seen <- s.Version:
			default:
			}
		})
	}()
	select {
	case v := <-seen:
		if v != 3 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch timeout")
	}
	snap2 := &snapshot.Snapshot{Version: 4}
	if _, err := store.Put(context.Background(), snap2); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-seen:
		if v != 4 {
			t.Fatalf("got %d", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch update timeout")
	}
}

func TestFanoutStore(t *testing.T) {
	primary := &memSnapStore{}
	blob := &memBlob{}
	mirror := objectstore.NewSnapshotStore(blob, "snap")
	f := &objectstore.FanoutStore{Primary: primary, Mirror: mirror}
	snap := &snapshot.Snapshot{APIKeys: map[string]snapshot.APIKey{}}
	v, err := f.Put(context.Background(), snap)
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("version=%d", v)
	}
	got, err := mirror.Latest(context.Background())
	if err != nil || got.Version != 1 {
		t.Fatalf("mirror=%v err=%v", got, err)
	}
}

func TestSnapshotStoreForRegion(t *testing.T) {
	blob := &memBlob{}
	root := objectstore.NewSnapshotStore(blob, "snapshots")
	eu := root.ForRegion("eu-west")
	snap := &snapshot.Snapshot{Version: 7, AllowedOrganizationIDs: []string{"org_a"}}
	if _, err := eu.Put(context.Background(), snap); err != nil {
		t.Fatal(err)
	}
	if _, ok := blob.objects["snapshots/eu-west/7.json"]; !ok {
		t.Fatalf("keys=%v", blob.objects)
	}
	got, err := eu.Latest(context.Background())
	if err != nil || got.Version != 7 {
		t.Fatalf("%v %v", got, err)
	}
}

type memSnapStore struct {
	version int64
	latest  *snapshot.Snapshot
}

func (m *memSnapStore) Put(_ context.Context, snap *snapshot.Snapshot) (int64, error) {
	m.version++
	cp := *snap
	cp.Version = m.version
	m.latest = &cp
	return m.version, nil
}

func (m *memSnapStore) Latest(context.Context) (*snapshot.Snapshot, error) {
	return m.latest, nil
}

func (m *memSnapStore) Watch(context.Context, time.Duration, func(*snapshot.Snapshot)) error {
	return nil
}
