package store

type URLStorage interface {
	Store(id, url string)
	Get(id string) (string, bool)
}
