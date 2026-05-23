package operation

import "time"

type AudioCategory string

const (
	AudioCategoryIVR      AudioCategory = "ivr"
	AudioCategoryHold     AudioCategory = "hold"
	AudioCategoryRingtone AudioCategory = "ringtone"
	AudioCategoryOther    AudioCategory = "other"
)

type AudioFile struct {
	ID        int64         `db:"id" json:"id"`
	TenantID  int64         `db:"tenant_id" json:"tenant_id"`
	Name      string        `db:"name" json:"name"`
	FileName  string        `db:"file_name" json:"file_name"`
	Category  AudioCategory `db:"category" json:"category"`
	FilePath  string        `db:"file_path" json:"file_path"`
	FileSize  int64         `db:"file_size" json:"file_size"`
	Duration  int           `db:"duration" json:"duration"`
	MimeType  string        `db:"mime_type" json:"mime_type"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
}

type BusinessHours struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Name      string    `db:"name" json:"name"`
	IsDefault bool      `db:"is_default" json:"is_default"`
	Timezone  string    `db:"timezone" json:"timezone"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type DayType string

const (
	DayTypeWeekday DayType = "weekday"
	DayTypeHoliday DayType = "holiday"
	DayTypeSpecial DayType = "special"
)

type BusinessHoursSchedule struct {
	ID              int64     `db:"id" json:"id"`
	BusinessHoursID int64     `db:"business_hours_id" json:"business_hours_id"`
	DayType         DayType   `db:"day_type" json:"day_type"`
	DayOfWeek       *int      `db:"day_of_week" json:"day_of_week,omitempty"`
	SpecificDate    *string   `db:"specific_date" json:"specific_date,omitempty"`
	StartTime       string    `db:"start_time" json:"start_time"`
	EndTime         string    `db:"end_time" json:"end_time"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}
