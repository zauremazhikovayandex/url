package storage

var Store *Storage

type Storage struct {
	data map[string]string
}

func InitStorage() {
	Store = &Storage{data: make(map[string]string)}
}

func (s *Storage) Set(key, value string) {
	s.data[key] = value
}

func (s *Storage) Get(key string) (string, bool) {
	val, ok := s.data[key]
	return val, ok
}

func (s *Storage) Delete(key string) {
	delete(s.data, key)
}
