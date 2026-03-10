package mcpdemo

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Runtime struct {
	clientSession *mcp.ClientSession
	serverName    string
	closeFn       func() error
	mu            sync.Mutex
}

type AddNumbersInput struct {
	A float64 `json:"a" jsonschema:"first number"`
	B float64 `json:"b" jsonschema:"second number"`
}

type AddNumbersOutput struct {
	Sum float64 `json:"sum" jsonschema:"sum of a and b"`
}

type CityTimeInput struct {
	City string `json:"city" jsonschema:"city name: utc, london, new_york, tokyo, sydney"`
}

type CityTimeOutput struct {
	City     string `json:"city" jsonschema:"resolved city key"`
	Timezone string `json:"timezone" jsonschema:"IANA timezone"`
	RFC3339  string `json:"rfc3339" jsonschema:"current time in RFC3339 format"`
	Unix     int64  `json:"unix" jsonschema:"unix timestamp in seconds"`
}

type ShiftTimeInput struct {
	RFC3339 string  `json:"rfc3339" jsonschema:"RFC3339 timestamp to shift"`
	Hours   float64 `json:"hours" jsonschema:"hours to add or subtract"`
}

type ShiftTimeOutput struct {
	Original string  `json:"original" jsonschema:"original RFC3339 timestamp"`
	Shifted  string  `json:"shifted" jsonschema:"shifted RFC3339 timestamp"`
	Hours    float64 `json:"hours" jsonschema:"shift amount in hours"`
}

type ListCarriersInput struct {
	OriginCountry      string `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
}

type ListCarriersOutput struct {
	OriginCountry      string   `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string   `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	Carriers           []string `json:"carriers" jsonschema:"available carrier identifiers for this route"`
}

type QuoteRateInput struct {
	Carrier            string  `json:"carrier" jsonschema:"carrier identifier, for example correos_priority"`
	OriginCountry      string  `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string  `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	WeightKG           float64 `json:"weight_kg" jsonschema:"package weight in kilograms"`
}

type QuoteRateOutput struct {
	Carrier            string  `json:"carrier" jsonschema:"carrier identifier"`
	Currency           string  `json:"currency" jsonschema:"quote currency"`
	BasePriceEUR       float64 `json:"base_price_eur" jsonschema:"base rate in euros before surcharges"`
	PricingBasis       string  `json:"pricing_basis" jsonschema:"human-readable pricing formula summary"`
	WeightKG           float64 `json:"weight_kg" jsonschema:"normalized package weight in kilograms"`
	OriginCountry      string  `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string  `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
}

type EstimateDeliveryInput struct {
	Carrier            string `json:"carrier" jsonschema:"carrier identifier"`
	OriginCountry      string `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
}

type EstimateDeliveryOutput struct {
	Carrier            string `json:"carrier" jsonschema:"carrier identifier"`
	ServiceLevel       string `json:"service_level" jsonschema:"carrier service level label"`
	MinDays            int    `json:"min_days" jsonschema:"minimum delivery estimate in business days"`
	MaxDays            int    `json:"max_days" jsonschema:"maximum delivery estimate in business days"`
	OriginCountry      string `json:"origin_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
	DestinationCountry string `json:"destination_country" jsonschema:"ISO 3166-1 alpha-2 country code"`
}

type ApplySurchargeInput struct {
	Carrier      string  `json:"carrier" jsonschema:"carrier identifier"`
	WeightKG     float64 `json:"weight_kg" jsonschema:"package weight in kilograms"`
	IsRemoteArea bool    `json:"is_remote_area" jsonschema:"whether the delivery address is in a remote area"`
	IsFragile    bool    `json:"is_fragile" jsonschema:"whether the parcel requires fragile handling"`
}

type SurchargeLine struct {
	Code      string  `json:"code" jsonschema:"machine-readable surcharge code"`
	Label     string  `json:"label" jsonschema:"human-readable surcharge label"`
	AmountEUR float64 `json:"amount_eur" jsonschema:"surcharge amount in euros"`
}

type ApplySurchargeOutput struct {
	Carrier           string          `json:"carrier" jsonschema:"carrier identifier"`
	WeightKG          float64         `json:"weight_kg" jsonschema:"normalized package weight in kilograms"`
	TotalSurchargeEUR float64         `json:"total_surcharge_eur" jsonschema:"sum of all applied surcharges in euros"`
	AppliedSurcharges []SurchargeLine `json:"applied_surcharges" jsonschema:"list of applied surcharge items"`
}

type QuoteSummaryInput struct {
	Carrier      string  `json:"carrier" jsonschema:"carrier identifier"`
	BasePriceEUR float64 `json:"base_price_eur" jsonschema:"base rate in euros before surcharges"`
	SurchargeEUR float64 `json:"surcharge_eur" jsonschema:"sum of applied surcharges in euros"`
	MinDays      int     `json:"min_days" jsonschema:"minimum delivery estimate in business days"`
	MaxDays      int     `json:"max_days" jsonschema:"maximum delivery estimate in business days"`
}

type QuoteSummaryOutput struct {
	Carrier        string  `json:"carrier" jsonschema:"carrier identifier"`
	Currency       string  `json:"currency" jsonschema:"quote currency"`
	TotalPriceEUR  float64 `json:"total_price_eur" jsonschema:"base rate plus surcharges in euros"`
	BasePriceEUR   float64 `json:"base_price_eur" jsonschema:"base rate in euros before surcharges"`
	SurchargeEUR   float64 `json:"surcharge_eur" jsonschema:"sum of applied surcharges in euros"`
	MinDays        int     `json:"min_days" jsonschema:"minimum delivery estimate in business days"`
	MaxDays        int     `json:"max_days" jsonschema:"maximum delivery estimate in business days"`
	DeliveryWindow string  `json:"delivery_window" jsonschema:"formatted delivery window summary"`
}

type carrierQuoteProfile struct {
	ServiceLevel      string
	BasePriceEUR      float64
	PerKGRateEUR      float64
	WeightHandlingEUR float64
	RemoteAreaEUR     float64
	FragileEUR        float64
	MinDays           int
	MaxDays           int
}

var cityZones = map[string]string{
	"utc":      "UTC",
	"london":   "Europe/London",
	"new_york": "America/New_York",
	"tokyo":    "Asia/Tokyo",
	"sydney":   "Australia/Sydney",
}

var shippingProfiles = map[string]carrierQuoteProfile{
	"correos_priority": {
		ServiceLevel:      "Priority",
		BasePriceEUR:      8.80,
		PerKGRateEUR:      1.40,
		WeightHandlingEUR: 0.75,
		RemoteAreaEUR:     2.60,
		FragileEUR:        1.80,
		MinDays:           3,
		MaxDays:           4,
	},
	"dhl_economy": {
		ServiceLevel:      "Economy",
		BasePriceEUR:      9.40,
		PerKGRateEUR:      1.55,
		WeightHandlingEUR: 0.95,
		RemoteAreaEUR:     2.90,
		FragileEUR:        1.65,
		MinDays:           3,
		MaxDays:           4,
	},
	"ups_standard": {
		ServiceLevel:      "Standard",
		BasePriceEUR:      11.20,
		PerKGRateEUR:      1.90,
		WeightHandlingEUR: 0.85,
		RemoteAreaEUR:     3.20,
		FragileEUR:        1.40,
		MinDays:           2,
		MaxDays:           3,
	},
	"gls_euro_business": {
		ServiceLevel:      "Euro Business",
		BasePriceEUR:      8.50,
		PerKGRateEUR:      1.30,
		WeightHandlingEUR: 0.65,
		RemoteAreaEUR:     2.50,
		FragileEUR:        1.50,
		MinDays:           4,
		MaxDays:           5,
	},
}

func NewDemo(ctx context.Context) (*Runtime, error) {
	server := mcp.NewServer(&mcp.Implementation{Name: "demo-mcp", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "add_numbers", Description: "Add two numbers together."}, addNumbers)
	mcp.AddTool(server, &mcp.Tool{Name: "city_time", Description: "Get the current time for a supported city."}, cityTime)
	mcp.AddTool(server, &mcp.Tool{Name: "shift_time", Description: "Shift an RFC3339 timestamp by a number of hours."}, shiftTime)
	mcp.AddTool(server, &mcp.Tool{Name: "list_carriers", Description: "List the supported carriers for a shipping route."}, listCarriers)
	mcp.AddTool(server, &mcp.Tool{Name: "quote_rate", Description: "Get a deterministic base shipping quote for a carrier and package weight."}, quoteRate)
	mcp.AddTool(server, &mcp.Tool{Name: "estimate_delivery", Description: "Get a deterministic delivery window for a carrier and route."}, estimateDelivery)
	mcp.AddTool(server, &mcp.Tool{Name: "apply_surcharge", Description: "Calculate deterministic surcharges for package traits such as weight, remote areas, and fragile handling."}, applySurcharge)
	mcp.AddTool(server, &mcp.Tool{Name: "quote_summary", Description: "Normalize a shipping quote into a sortable final summary."}, quoteSummary)

	client := mcp.NewClient(&mcp.Implementation{Name: "demo-client", Version: "0.1.0"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect server: %w", err)
	}
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		if closeErr := serverSession.Close(); closeErr != nil {
			return nil, fmt.Errorf("connect client: %w (close server session: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("connect client: %w", err)
	}

	return &Runtime{
		clientSession: clientSession,
		serverName:    "demo",
		closeFn: func() error {
			return firstError(clientSession.Close(), serverSession.Close())
		},
	}, nil
}

func (r *Runtime) Close() error {
	if r.closeFn == nil {
		return nil
	}
	return r.closeFn()
}

func (r *Runtime) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result, err := r.clientSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (r *Runtime) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.clientSession.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

func (r *Runtime) ServerName() string {
	if r == nil || r.serverName == "" {
		return "mcp"
	}
	return r.serverName
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func addNumbers(_ context.Context, _ *mcp.CallToolRequest, input AddNumbersInput) (*mcp.CallToolResult, AddNumbersOutput, error) {
	return nil, AddNumbersOutput{Sum: input.A + input.B}, nil
}

func cityTime(_ context.Context, _ *mcp.CallToolRequest, input CityTimeInput) (*mcp.CallToolResult, CityTimeOutput, error) {
	zone, ok := cityZones[input.City]
	if !ok {
		return nil, CityTimeOutput{}, fmt.Errorf("unsupported city %q", input.City)
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, CityTimeOutput{}, fmt.Errorf("load location: %w", err)
	}
	now := time.Now().In(loc)
	return nil, CityTimeOutput{
		City:     input.City,
		Timezone: zone,
		RFC3339:  now.Format(time.RFC3339),
		Unix:     now.Unix(),
	}, nil
}

func shiftTime(_ context.Context, _ *mcp.CallToolRequest, input ShiftTimeInput) (*mcp.CallToolResult, ShiftTimeOutput, error) {
	ts, err := time.Parse(time.RFC3339, input.RFC3339)
	if err != nil {
		return nil, ShiftTimeOutput{}, fmt.Errorf("parse rfc3339: %w", err)
	}
	shifted := ts.Add(time.Duration(input.Hours * float64(time.Hour)))
	return nil, ShiftTimeOutput{
		Original: ts.Format(time.RFC3339),
		Shifted:  shifted.Format(time.RFC3339),
		Hours:    input.Hours,
	}, nil
}

func listCarriers(_ context.Context, _ *mcp.CallToolRequest, input ListCarriersInput) (*mcp.CallToolResult, ListCarriersOutput, error) {
	origin, destination, err := normalizedShippingRoute(input.OriginCountry, input.DestinationCountry)
	if err != nil {
		return nil, ListCarriersOutput{}, err
	}

	carriers := make([]string, 0, len(shippingProfiles))
	for carrier := range shippingProfiles {
		carriers = append(carriers, carrier)
	}
	sort.Strings(carriers)

	return nil, ListCarriersOutput{
		OriginCountry:      origin,
		DestinationCountry: destination,
		Carriers:           carriers,
	}, nil
}

func quoteRate(_ context.Context, _ *mcp.CallToolRequest, input QuoteRateInput) (*mcp.CallToolResult, QuoteRateOutput, error) {
	origin, destination, profile, weightKG, err := resolveShippingQuote(input.Carrier, input.OriginCountry, input.DestinationCountry, input.WeightKG)
	if err != nil {
		return nil, QuoteRateOutput{}, err
	}

	basePrice := roundEUR(profile.BasePriceEUR + profile.PerKGRateEUR*weightKG)
	return nil, QuoteRateOutput{
		Carrier:            input.Carrier,
		Currency:           "EUR",
		BasePriceEUR:       basePrice,
		PricingBasis:       fmt.Sprintf("%.2f base + %.2f/kg * %.2fkg", profile.BasePriceEUR, profile.PerKGRateEUR, weightKG),
		WeightKG:           weightKG,
		OriginCountry:      origin,
		DestinationCountry: destination,
	}, nil
}

func estimateDelivery(_ context.Context, _ *mcp.CallToolRequest, input EstimateDeliveryInput) (*mcp.CallToolResult, EstimateDeliveryOutput, error) {
	origin, destination, profile, err := resolveCarrierRoute(input.Carrier, input.OriginCountry, input.DestinationCountry)
	if err != nil {
		return nil, EstimateDeliveryOutput{}, err
	}

	return nil, EstimateDeliveryOutput{
		Carrier:            input.Carrier,
		ServiceLevel:       profile.ServiceLevel,
		MinDays:            profile.MinDays,
		MaxDays:            profile.MaxDays,
		OriginCountry:      origin,
		DestinationCountry: destination,
	}, nil
}

func applySurcharge(_ context.Context, _ *mcp.CallToolRequest, input ApplySurchargeInput) (*mcp.CallToolResult, ApplySurchargeOutput, error) {
	profile, weightKG, err := resolveCarrierWeight(input.Carrier, input.WeightKG)
	if err != nil {
		return nil, ApplySurchargeOutput{}, err
	}

	items := make([]SurchargeLine, 0, 3)
	if weightKG > 2 {
		items = append(items, SurchargeLine{Code: "weight_handling", Label: "Weight handling", AmountEUR: profile.WeightHandlingEUR})
	}
	if input.IsRemoteArea {
		items = append(items, SurchargeLine{Code: "remote_area", Label: "Remote area", AmountEUR: profile.RemoteAreaEUR})
	}
	if input.IsFragile {
		items = append(items, SurchargeLine{Code: "fragile", Label: "Fragile handling", AmountEUR: profile.FragileEUR})
	}

	total := 0.0
	for _, item := range items {
		total += item.AmountEUR
	}

	return nil, ApplySurchargeOutput{
		Carrier:           input.Carrier,
		WeightKG:          weightKG,
		TotalSurchargeEUR: roundEUR(total),
		AppliedSurcharges: items,
	}, nil
}

func quoteSummary(_ context.Context, _ *mcp.CallToolRequest, input QuoteSummaryInput) (*mcp.CallToolResult, QuoteSummaryOutput, error) {
	if _, ok := shippingProfiles[input.Carrier]; !ok {
		return nil, QuoteSummaryOutput{}, fmt.Errorf("unsupported carrier %q", input.Carrier)
	}
	if input.MinDays <= 0 || input.MaxDays <= 0 || input.MinDays > input.MaxDays {
		return nil, QuoteSummaryOutput{}, fmt.Errorf("invalid delivery window %d-%d", input.MinDays, input.MaxDays)
	}
	total := roundEUR(input.BasePriceEUR + input.SurchargeEUR)
	return nil, QuoteSummaryOutput{
		Carrier:        input.Carrier,
		Currency:       "EUR",
		TotalPriceEUR:  total,
		BasePriceEUR:   roundEUR(input.BasePriceEUR),
		SurchargeEUR:   roundEUR(input.SurchargeEUR),
		MinDays:        input.MinDays,
		MaxDays:        input.MaxDays,
		DeliveryWindow: fmt.Sprintf("%d-%d business days", input.MinDays, input.MaxDays),
	}, nil
}

func normalizedShippingRoute(originCountry, destinationCountry string) (string, string, error) {
	origin := normalizeCountryCode(originCountry)
	destination := normalizeCountryCode(destinationCountry)
	if origin == "" || destination == "" {
		return "", "", fmt.Errorf("origin_country and destination_country are required")
	}
	if origin == destination {
		return "", "", fmt.Errorf("origin_country and destination_country must differ")
	}
	return origin, destination, nil
}

func resolveCarrierRoute(carrier, originCountry, destinationCountry string) (string, string, carrierQuoteProfile, error) {
	origin, destination, err := normalizedShippingRoute(originCountry, destinationCountry)
	if err != nil {
		return "", "", carrierQuoteProfile{}, err
	}
	profile, ok := shippingProfiles[carrier]
	if !ok {
		return "", "", carrierQuoteProfile{}, fmt.Errorf("unsupported carrier %q", carrier)
	}
	return origin, destination, profile, nil
}

func resolveCarrierWeight(carrier string, weightKG float64) (carrierQuoteProfile, float64, error) {
	profile, ok := shippingProfiles[carrier]
	if !ok {
		return carrierQuoteProfile{}, 0, fmt.Errorf("unsupported carrier %q", carrier)
	}
	if weightKG <= 0 {
		return carrierQuoteProfile{}, 0, fmt.Errorf("weight_kg must be greater than zero")
	}
	return profile, roundWeight(weightKG), nil
}

func resolveShippingQuote(carrier, originCountry, destinationCountry string, weightKG float64) (string, string, carrierQuoteProfile, float64, error) {
	origin, destination, profile, err := resolveCarrierRoute(carrier, originCountry, destinationCountry)
	if err != nil {
		return "", "", carrierQuoteProfile{}, 0, err
	}
	_, normalizedWeight, err := resolveCarrierWeight(carrier, weightKG)
	if err != nil {
		return "", "", carrierQuoteProfile{}, 0, err
	}
	return origin, destination, profile, normalizedWeight, nil
}

func normalizeCountryCode(value string) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) != 2 {
		return ""
	}
	letters := []byte(value)
	for i := range letters {
		if letters[i] >= 'a' && letters[i] <= 'z' {
			letters[i] -= 'a' - 'A'
		}
		if letters[i] < 'A' || letters[i] > 'Z' {
			return ""
		}
	}
	return string(letters)
}

func roundWeight(weightKG float64) float64 {
	return math.Round(weightKG*100) / 100
}

func roundEUR(value float64) float64 {
	return math.Round(value*100) / 100
}
