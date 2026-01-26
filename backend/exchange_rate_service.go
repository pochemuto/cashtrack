package cashtrack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	db "cashtrack/backend/gen/db"

	"github.com/jackc/pgx/v5/pgtype"
)

type ExchangeRateService struct {
	db            *Db
	httpClient    *http.Client
	rateCache     map[string]float64
	rateCacheLock sync.Mutex
}

func NewExchangeRateService(db *Db) *ExchangeRateService {
	return &ExchangeRateService{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		rateCache:  make(map[string]float64),
	}
}

func (s *ExchangeRateService) GetRateToCHF(ctx context.Context, baseCurrency string, date time.Time) (float64, error) {
	base := strings.ToUpper(strings.TrimSpace(baseCurrency))
	if base == "" || base == "CHF" {
		return 1, nil
	}
	dateKey := date.Format("2006-01-02")
	cacheKey := base + "|" + dateKey

	s.rateCacheLock.Lock()
	if rate, ok := s.rateCache[cacheKey]; ok {
		s.rateCacheLock.Unlock()
		return rate, nil
	}
	s.rateCacheLock.Unlock()

	ratedb, err := s.getRateFromDB(ctx, base, date)
	if err == nil && ratedb > 0 {
		s.rateCacheLock.Lock()
		s.rateCache[cacheKey] = ratedb
		s.rateCacheLock.Unlock()
		return ratedb, nil
	}

	rate, err := s.getRateFromAPI(ctx, base, date)
	if err != nil {
		return 0, err
	}

	if err := s.storeRate(ctx, base, "CHF", date, rate); err != nil {
		log.Warn().Err(err).Str("currency", base).Time("date", date).Msg("failed to store exchange rate")
	}

	s.rateCacheLock.Lock()
	s.rateCache[cacheKey] = rate
	s.rateCacheLock.Unlock()
	return rate, nil
}

func (s *ExchangeRateService) getRateFromAPI(ctx context.Context, baseCurrency string, date time.Time) (float64, error) {
	dateKey := date.Format("2006-01-02")
	url := fmt.Sprintf("https://api.exchangerate.host/%s?base=%s&symbols=CHF", dateKey, baseCurrency)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("rate request failed: %s", resp.Status)
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	rate, ok := payload.Rates["CHF"]
	if !ok || rate == 0 {
		return 0, fmt.Errorf("missing CHF rate for %s on %s", baseCurrency, dateKey)
	}
	return rate, nil
}

func (s *ExchangeRateService) getRateFromDB(ctx context.Context, baseCurrency string, date time.Time) (float64, error) {
	rate, err := s.db.Queries.GetExchangeRate(ctx, db.GetExchangeRateParams{
		RateDate:       pgtype.Date{Time: date, Valid: true},
		BaseCurrency:   baseCurrency,
		TargetCurrency: "CHF",
	})
	if err != nil {
		return 0, err
	}
	return numericToFloat(rate)
}

func (s *ExchangeRateService) storeRate(ctx context.Context, baseCurrency string, targetCurrency string, date time.Time, rate float64) error {
	numericRate, err := numericFromString(formatFloat(rate))
	if err != nil {
		return err
	}
	return s.db.Queries.UpsertExchangeRate(ctx, db.UpsertExchangeRateParams{
		RateDate:       pgtype.Date{Time: date, Valid: true},
		BaseCurrency:   baseCurrency,
		TargetCurrency: targetCurrency,
		Rate:           numericRate,
	})
}
