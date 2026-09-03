package utils

import (
	"time"

	"github.com/gogf/gf/v2/os/gtime"
)

const (
	DateTimeFormat = "Y-m-d H:i:s"
	DateFormat     = "Y-m-d"
	TimeFormat     = "H:i:s"
)

func Now() *gtime.Time {
	return gtime.Now()
}

func FormatTime(t *gtime.Time, layout string) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}

func FormatDateTime(t *gtime.Time) string {
	return FormatTime(t, DateTimeFormat)
}

func FormatDate(t *gtime.Time) string {
	return FormatTime(t, DateFormat)
}

func ParseTime(str string, layout ...string) (*gtime.Time, error) {
	format := DateTimeFormat
	if len(layout) > 0 && layout[0] != "" {
		format = layout[0]
	}
	return gtime.StrToTimeFormat(str, format)
}

func ToTime(t time.Time) *gtime.Time {
	return gtime.New(t)
}

func FromUnix(sec int64) *gtime.Time {
	return gtime.NewFromTimeStamp(sec)
}

func FromUnixMilli(msec int64) *gtime.Time {
	return gtime.NewFromTimeStamp(msec / 1000)
}

func BeginOfDay(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).StartOfDay()
}

func EndOfDay(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).EndOfDay()
}

func BeginOfWeek(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).StartOfWeek()
}

func EndOfWeek(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).EndOfWeek()
}

func BeginOfMonth(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).StartOfMonth()
}

func EndOfMonth(t *gtime.Time) *gtime.Time {
	if t == nil {
		t = Now()
	}
	return gtime.New(t.Time).EndOfMonth()
}
