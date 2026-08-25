package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled                bool
	ScheduleEnabledDefault bool
	AdminChatIDs           []int64
	Schedule               DailySchedule
	Timezone               *time.Location
	TimezoneName           string
	DatabaseURL            string
	TmpDir                 string
	BotToken               string
	VLESSURL               string
}

type DailySchedule struct {
	Hour   int
	Minute int
}

func LoadConfig() (Config, error) {
	enabled := getenvBool("BACKUP_ENABLED", true)
	scheduleEnabled := getenvBool("BACKUP_SCHEDULE_ENABLED", true)

	adminChatIDs, err := parseAdminChatIDs(os.Getenv("BACKUP_ADMIN_CHAT_IDS"))
	if err != nil {
		return Config{}, err
	}

	schedule, err := ParseDailySchedule(getenv("BACKUP_SCHEDULE", "03:00"))
	if err != nil {
		return Config{}, err
	}

	timezoneName := getenv("BACKUP_TIMEZONE", "Asia/Krasnoyarsk")
	timezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		return Config{}, fmt.Errorf("load backup timezone: %w", err)
	}

	databaseURL := getenv("BACKUP_DATABASE_URL", os.Getenv("DATABASE_URL"))
	tmpDir := filepath.Clean(getenv("BACKUP_TMP_DIR", "/tmp/movies-backups"))
	botToken := os.Getenv("BOT_TOKEN")

	if enabled {
		if botToken == "" {
			return Config{}, fmt.Errorf("BOT_TOKEN is required when BACKUP_ENABLED=true")
		}
		if len(adminChatIDs) == 0 {
			return Config{}, fmt.Errorf("BACKUP_ADMIN_CHAT_IDS is required when BACKUP_ENABLED=true")
		}
		if databaseURL == "" {
			return Config{}, fmt.Errorf("BACKUP_DATABASE_URL or DATABASE_URL is required when BACKUP_ENABLED=true")
		}
	}

	return Config{
		Enabled:                enabled,
		ScheduleEnabledDefault: scheduleEnabled,
		AdminChatIDs:           adminChatIDs,
		Schedule:               schedule,
		Timezone:               timezone,
		TimezoneName:           timezoneName,
		DatabaseURL:            databaseURL,
		TmpDir:                 tmpDir,
		BotToken:               botToken,
		VLESSURL:               strings.TrimSpace(os.Getenv("BACKUP_VLESS_URL")),
	}, nil
}

func ParseDailySchedule(value string) (DailySchedule, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return DailySchedule{}, fmt.Errorf("invalid BACKUP_SCHEDULE %q, expected HH:MM", value)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return DailySchedule{}, fmt.Errorf("invalid BACKUP_SCHEDULE hour %q", parts[0])
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return DailySchedule{}, fmt.Errorf("invalid BACKUP_SCHEDULE minute %q", parts[1])
	}

	return DailySchedule{Hour: hour, Minute: minute}, nil
}

func (s DailySchedule) String() string {
	return fmt.Sprintf("%02d:%02d", s.Hour, s.Minute)
}

func parseAdminChatIDs(value string) ([]int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	chatIDs := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		chatID, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse BACKUP_ADMIN_CHAT_IDS value %q: %w", part, err)
		}
		chatIDs = append(chatIDs, chatID)
	}
	return chatIDs, nil
}

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
