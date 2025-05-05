package repository

// Repository определяет интерфейс для работы с хранилищем URL
type Repository interface {
	Save(id, url string) error
	Get(id string) (string, bool)
	Clear()
}
