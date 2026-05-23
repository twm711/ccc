package operation

import "context"

type AudioFileRepository interface {
	Create(ctx context.Context, af *AudioFile) error
	GetByID(ctx context.Context, id int64) (*AudioFile, error)
	List(ctx context.Context, tenantID int64, category AudioCategory) ([]*AudioFile, error)
	Delete(ctx context.Context, id int64) error
}

type BusinessHoursRepository interface {
	Create(ctx context.Context, bh *BusinessHours) error
	GetByID(ctx context.Context, id int64) (*BusinessHours, error)
	Update(ctx context.Context, bh *BusinessHours) error
	List(ctx context.Context, tenantID int64) ([]*BusinessHours, error)
	Delete(ctx context.Context, id int64) error
}

type BusinessHoursScheduleRepository interface {
	Create(ctx context.Context, s *BusinessHoursSchedule) error
	GetByBusinessHoursID(ctx context.Context, bhID int64) ([]*BusinessHoursSchedule, error)
	DeleteByBusinessHoursID(ctx context.Context, bhID int64) error
}
