package entities

type Dummy struct {
	ID   int64  `gorm:"primaryKey"`
	Text string `gorm:"not null"`
}
