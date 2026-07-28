package localization

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tsc-forex/tsc_forex/internal/domain"
	"github.com/tsc-forex/tsc_forex/internal/models"
)

const (
	DefaultLocale = "tr"
	EnglishLocale = "en"
)

var localeOrder = []string{"tr", "en", "ar", "de", "it", "es"}

var localeDefinitions = []models.SupportedLocale{
	{Code: "tr", NativeName: "Türkçe", EnglishName: "Turkish", FallbackCode: "tr", SortOrder: 10, Enabled: true},
	{Code: "en", NativeName: "English", EnglishName: "English", FallbackCode: "tr", SortOrder: 20, Enabled: true},
	{Code: "ar", NativeName: "العربية", EnglishName: "Arabic", FallbackCode: "en", IsRTL: true, SortOrder: 30, Enabled: true},
	{Code: "de", NativeName: "Deutsch", EnglishName: "German", FallbackCode: "en", SortOrder: 40, Enabled: true},
	{Code: "it", NativeName: "Italiano", EnglishName: "Italian", FallbackCode: "en", SortOrder: 50, Enabled: true},
	{Code: "es", NativeName: "Español", EnglishName: "Spanish", FallbackCode: "en", SortOrder: 60, Enabled: true},
}

//go:generate node generate_currency_names.mjs

// currencyNames is generated from Unicode CLDR through Intl.DisplayNames.
// Turkish labels deliberately retain the existing product catalog wording.
//
//go:embed currency_names.json
var currencyNamesJSON []byte

var goldNames = map[string]map[string]string{
	"ONS_GOLD": {
		"tr": "Ons Altın", "en": "Gold Ounce", "ar": "أونصة الذهب",
		"de": "Goldunze", "it": "Oncia d'oro", "es": "Onza de oro",
	},
	"GRAM_GOLD": {
		"tr": "Gram Altın (Has)", "en": "Gram Gold (Fine)", "ar": "غرام الذهب (خالص)",
		"de": "Feingold (1 Gramm)", "it": "Oro fino (1 grammo)", "es": "Oro fino (1 gramo)",
	},
	"CEYREK_GOLD": {
		"tr": "Çeyrek Altın", "en": "Quarter Gold Coin", "ar": "ربع ليرة ذهبية",
		"de": "Viertelgoldmünze", "it": "Moneta d'oro da un quarto", "es": "Moneda de oro de cuarto",
	},
	"YARIM_GOLD": {
		"tr": "Yarım Altın", "en": "Half Gold Coin", "ar": "نصف ليرة ذهبية",
		"de": "Halbe Goldmünze", "it": "Mezza moneta d'oro", "es": "Media moneda de oro",
	},
	"TAM_GOLD": {
		"tr": "Tam Altın", "en": "Full Gold Coin", "ar": "ليرة ذهبية كاملة",
		"de": "Ganze Goldmünze", "it": "Moneta d'oro intera", "es": "Moneda de oro entera",
	},
	"CUMHURIYET_GOLD": {
		"tr": "Cumhuriyet Altını", "en": "Republic Gold Coin", "ar": "ذهب الجمهورية",
		"de": "Republik-Goldmünze", "it": "Oro della Repubblica", "es": "Oro de la República",
	},
	"ATA_GOLD": {
		"tr": "Ata Altın", "en": "Ata Gold Coin", "ar": "ذهب أتا",
		"de": "Ata-Goldmünze", "it": "Moneta d'oro Ata", "es": "Moneda de oro Ata",
	},
	"GRAM_22K": {
		"tr": "22 Ayar Bilezik", "en": "22K Gold Bracelet", "ar": "سوار ذهب عيار 22",
		"de": "22-Karat-Goldarmband", "it": "Bracciale d'oro 22 carati", "es": "Pulsera de oro de 22 quilates",
	},
	"GRAM_18K": {
		"tr": "18 Ayar Altın", "en": "18K Gold", "ar": "ذهب عيار 18",
		"de": "18-Karat-Gold", "it": "Oro 18 carati", "es": "Oro de 18 quilates",
	},
	"GRAM_14K": {
		"tr": "14 Ayar Altın", "en": "14K Gold", "ar": "ذهب عيار 14",
		"de": "14-Karat-Gold", "it": "Oro 14 carati", "es": "Oro de 14 quilates",
	},
}

// CatalogSeed materializes the code-defined calculation catalog into a
// normalized database registry plus exactly one translation row per language.
// Company and crypto names are proper names and intentionally remain invariant;
// fiat currencies use CLDR and Turkish gold products use reviewed terminology.
func CatalogSeed() ([]models.SupportedLocale, []models.InstrumentDefinition, []models.InstrumentTranslation, error) {
	var currencies map[string]map[string]string
	if err := json.Unmarshal(currencyNamesJSON, &currencies); err != nil {
		return nil, nil, nil, fmt.Errorf("decode currency translations: %w", err)
	}

	locales := append([]models.SupportedLocale(nil), localeDefinitions...)
	instruments := make([]models.InstrumentDefinition, 0, len(domain.Catalog))
	translations := make([]models.InstrumentTranslation, 0, len(domain.Catalog)*len(localeOrder))
	seen := make(map[string]struct{}, len(domain.Catalog))

	for _, instrument := range domain.Catalog {
		if _, exists := seen[instrument.Code]; exists {
			return nil, nil, nil, fmt.Errorf("duplicate instrument code %q", instrument.Code)
		}
		seen[instrument.Code] = struct{}{}
		instruments = append(instruments, models.InstrumentDefinition{
			Code: instrument.Code, Kind: string(instrument.Kind), Unit: instrument.Unit,
			DefaultLocaleCode: DefaultLocale, Active: true,
		})

		names := map[string]string{}
		for _, locale := range localeOrder {
			names[locale] = instrument.DisplayName
		}
		switch instrument.Kind {
		case domain.KindCurrency:
			localized, ok := currencies[instrument.Code]
			if !ok {
				return nil, nil, nil, fmt.Errorf("missing currency translations for %s", instrument.Code)
			}
			names = localized
		case domain.KindGold:
			localized, ok := goldNames[instrument.Code]
			if !ok {
				return nil, nil, nil, fmt.Errorf("missing gold translations for %s", instrument.Code)
			}
			names = localized
		}

		for _, locale := range localeOrder {
			displayName := strings.TrimSpace(names[locale])
			if displayName == "" {
				return nil, nil, nil, fmt.Errorf("empty %s translation for %s", locale, instrument.Code)
			}
			translations = append(translations, models.InstrumentTranslation{
				InstrumentCode: instrument.Code,
				LocaleCode:     locale,
				DisplayName:    displayName,
			})
		}
	}
	if len(currencies) != len(domain.Currencies) {
		return nil, nil, nil, fmt.Errorf(
			"currency translation catalog has %d entries, want %d",
			len(currencies), len(domain.Currencies),
		)
	}
	return locales, instruments, translations, nil
}

// SeedCatalog idempotently installs reference rows. Upserts make concurrent
// rolling-start pods safe, while the migration advisory lock serializes this
// operation on PostgreSQL.
func SeedCatalog(ctx context.Context, db *gorm.DB) error {
	locales, instruments, translations, err := CatalogSeed()
	if err != nil {
		return err
	}
	session := db.WithContext(ctx)
	if err := session.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"native_name", "english_name", "fallback_code", "is_rtl", "enabled", "sort_order", "updated_at",
		}),
	}).Create(&locales).Error; err != nil {
		return fmt.Errorf("seed supported locales: %w", err)
	}
	if err := session.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"kind", "unit", "default_locale_code", "active", "updated_at",
		}),
	}).CreateInBatches(&instruments, 250).Error; err != nil {
		return fmt.Errorf("seed instruments: %w", err)
	}
	if err := session.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "instrument_code"}, {Name: "locale_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"display_name", "updated_at",
		}),
	}).CreateInBatches(&translations, 400).Error; err != nil {
		return fmt.Errorf("seed instrument translations: %w", err)
	}
	return nil
}

// SupportedLocales returns the allow-listed language tags in deterministic
// product order.
func SupportedLocales() []string {
	return append([]string(nil), localeOrder...)
}
