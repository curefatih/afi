package snapshot

type Manager interface {
	Current() *Snapshot

	Swap(*Snapshot)
}
