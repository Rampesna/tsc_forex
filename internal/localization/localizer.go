package localization

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

const ClientLocaleHeader = "X-TSC-Client-Locale"

var supportedLocaleSet = func() map[string]struct{} {
	out := make(map[string]struct{}, len(localeOrder))
	for _, locale := range localeOrder {
		out[locale] = struct{}{}
	}
	return out
}()

type instrumentNameSnapshot struct {
	names     map[string]map[string]string
	fallbacks map[string]string
}

// InstrumentLocalizer keeps the complete translation table in an immutable
// in-memory snapshot. Requests are O(1) map reads and never hit PostgreSQL;
// Reload atomically swaps a fully validated snapshot.
type InstrumentLocalizer struct {
	snapshot atomic.Value // instrumentNameSnapshot
}

func NewInstrumentLocalizer(db *gorm.DB) (*InstrumentLocalizer, error) {
	localizer := &InstrumentLocalizer{}
	if err := localizer.Reload(db); err != nil {
		return nil, err
	}
	return localizer, nil
}

func (l *InstrumentLocalizer) Reload(db *gorm.DB) error {
	if db == nil {
		return errors.New("instrument localization database is nil")
	}
	var locales []models.SupportedLocale
	if err := db.Where("enabled = ?", true).Order("sort_order ASC").Find(&locales).Error; err != nil {
		return fmt.Errorf("load supported locales: %w", err)
	}
	var rows []models.InstrumentTranslation
	if err := db.
		Table("instrument_translations AS translation").
		Select("translation.instrument_code, translation.locale_code, translation.display_name").
		Joins("JOIN instruments AS instrument ON instrument.code = translation.instrument_code AND instrument.active = ?", true).
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load instrument translations: %w", err)
	}

	snapshot := instrumentNameSnapshot{
		names:     make(map[string]map[string]string, len(domain.Catalog)),
		fallbacks: make(map[string]string, len(locales)),
	}
	for _, locale := range locales {
		if _, supported := supportedLocaleSet[locale.Code]; !supported {
			continue
		}
		fallback := locale.FallbackCode
		if _, supported := supportedLocaleSet[fallback]; !supported {
			fallback = DefaultLocale
		}
		snapshot.fallbacks[locale.Code] = fallback
	}
	for _, row := range rows {
		if _, supported := supportedLocaleSet[row.LocaleCode]; !supported {
			continue
		}
		names := snapshot.names[row.InstrumentCode]
		if names == nil {
			names = make(map[string]string, len(localeOrder))
			snapshot.names[row.InstrumentCode] = names
		}
		names[row.LocaleCode] = strings.TrimSpace(row.DisplayName)
	}
	if err := validateSnapshot(snapshot); err != nil {
		return err
	}
	l.snapshot.Store(snapshot)
	return nil
}

func validateSnapshot(snapshot instrumentNameSnapshot) error {
	for _, locale := range localeOrder {
		if _, ok := snapshot.fallbacks[locale]; !ok {
			return fmt.Errorf("enabled locale %s is missing from localization registry", locale)
		}
	}
	for _, instrument := range domain.Catalog {
		names := snapshot.names[instrument.Code]
		for _, locale := range localeOrder {
			if strings.TrimSpace(names[locale]) == "" {
				return fmt.Errorf("instrument %s is missing locale %s", instrument.Code, locale)
			}
		}
	}
	return nil
}

// DisplayName resolves requested locale -> configured fallback -> Turkish
// compatibility label -> static calculation-catalog label -> instrument code.
func (l *InstrumentLocalizer) DisplayName(code, requestedLocale string) string {
	if l == nil {
		return staticDisplayName(code)
	}
	snapshot, ok := l.snapshot.Load().(instrumentNameSnapshot)
	if !ok {
		return staticDisplayName(code)
	}
	locale := NormalizeLocale(requestedLocale)
	names := snapshot.names[code]
	if name := strings.TrimSpace(names[locale]); name != "" {
		return name
	}
	if fallback := snapshot.fallbacks[locale]; fallback != "" {
		if name := strings.TrimSpace(names[fallback]); name != "" {
			return name
		}
	}
	if name := strings.TrimSpace(names[DefaultLocale]); name != "" {
		return name
	}
	return staticDisplayName(code)
}

// DisplayNames returns the immutable six-language map stored in the current
// snapshot. Callers must not mutate it.
func (l *InstrumentLocalizer) DisplayNames(code string) map[string]string {
	if l == nil {
		return map[string]string{DefaultLocale: staticDisplayName(code)}
	}
	snapshot, ok := l.snapshot.Load().(instrumentNameSnapshot)
	if !ok || snapshot.names[code] == nil {
		return map[string]string{DefaultLocale: staticDisplayName(code)}
	}
	return snapshot.names[code]
}

func staticDisplayName(code string) string {
	if instrument, ok := domain.CatalogByCode[code]; ok && instrument.DisplayName != "" {
		return instrument.DisplayName
	}
	return code
}

// NormalizeLocale reduces BCP-47/Accept-Language values to the supported base
// language. Unsupported input preserves the historical Turkish default.
func NormalizeLocale(raw string) string {
	if locale, ok := matchLocale(raw); ok {
		return locale
	}
	return DefaultLocale
}

func matchLocale(raw string) (string, bool) {
	bestLocale := ""
	bestQuality := -1.0
	for _, item := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(item), ";")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		quality := 1.0
		if len(parts) > 1 {
			for _, parameter := range parts[1:] {
				keyValue := strings.SplitN(strings.TrimSpace(parameter), "=", 2)
				if len(keyValue) != 2 || !strings.EqualFold(keyValue[0], "q") {
					continue
				}
				parsed, err := strconv.ParseFloat(keyValue[1], 64)
				if err != nil {
					quality = 0
				} else {
					quality = parsed
				}
			}
		}
		if quality <= 0 {
			continue
		}
		tag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(parts[0]), "_", "-"))
		base := strings.SplitN(tag, "-", 2)[0]
		if _, ok := supportedLocaleSet[base]; ok && quality > bestQuality {
			bestLocale = base
			bestQuality = quality
		}
	}
	return bestLocale, bestLocale != ""
}

// RequestLocale accepts the app-specific header, normal HTTP negotiation, and
// a WebSocket query parameter. The query is useful because not every native
// WebSocket implementation permits custom headers.
func RequestLocale(request *http.Request) string {
	if request == nil {
		return DefaultLocale
	}
	for _, raw := range []string{
		request.URL.Query().Get("locale"),
		request.Header.Get(ClientLocaleHeader),
		request.Header.Get("Accept-Language"),
	} {
		if locale, ok := matchLocale(raw); ok {
			return locale
		}
	}
	return DefaultLocale
}
