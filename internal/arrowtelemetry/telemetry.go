package arrowtelemetry

import (
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/egidinas/gossamer/internal/contracts"
)

const (
	SchemaName    = "gossamer.telemetry.arrow.v2"
	TransportMIME = "application/vnd.apache.arrow.stream"
)

var dictionaryString = &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}

var TelemetrySchema = arrow.NewSchema([]arrow.Field{
	{Name: "timestamp_ns", Type: arrow.PrimitiveTypes.Int64},
	{Name: "sensor", Type: dictionaryString},
	{Name: "value", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
	{Name: "unit", Type: dictionaryString},
	{Name: "campaign_id", Type: dictionaryString},
	{Name: "source", Type: dictionaryString},
	{Name: "series_role", Type: dictionaryString},
	{Name: "signal_kind", Type: dictionaryString},
	{Name: "source_family", Type: dictionaryString},
	{Name: "quality", Type: dictionaryString},
	{Name: "state", Type: dictionaryString, Nullable: true},
}, nil)

type SignalMeta struct {
	Unit         string
	Source       string
	SeriesRole   string
	SignalKind   string
	SourceFamily string
}

func MetadataFromGraph(model contracts.GraphModel) map[string]SignalMeta {
	meta := map[string]SignalMeta{}
	for _, lane := range model.Lanes {
		for _, series := range lane.Series {
			meta[series.ID] = SignalMeta{
				Unit:       series.Units,
				Source:     series.Source,
				SeriesRole: series.Role,
				SignalKind: "numeric",
			}
		}
	}
	if model.GraphWall != nil {
		for _, section := range model.GraphWall.Sections {
			for _, card := range section.Cards {
				for _, signal := range card.Signals {
					item := meta[signal.ID]
					if signal.Unit != "" {
						item.Unit = signal.Unit
					}
					if signal.Source != "" {
						item.Source = signal.Source
					}
					if signal.Role != "" {
						item.SeriesRole = signal.Role
					}
					if signal.Kind != "" {
						item.SignalKind = signal.Kind
					}
					if signal.SourceFamily != "" {
						item.SourceFamily = signal.SourceFamily
					}
					meta[signal.ID] = item
				}
			}
		}
	}
	return meta
}

func WriteCampaign(path, campaignID string, samples []contracts.TelemetrySample, meta map[string]SignalMeta) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	builder := array.NewRecordBuilder(memory.DefaultAllocator, TelemetrySchema)
	defer builder.Release()

	ts := builder.Field(0).(*array.Int64Builder)
	sensor := builder.Field(1).(*array.BinaryDictionaryBuilder)
	value := builder.Field(2).(*array.Float64Builder)
	unit := builder.Field(3).(*array.BinaryDictionaryBuilder)
	campaign := builder.Field(4).(*array.BinaryDictionaryBuilder)
	source := builder.Field(5).(*array.BinaryDictionaryBuilder)
	role := builder.Field(6).(*array.BinaryDictionaryBuilder)
	kind := builder.Field(7).(*array.BinaryDictionaryBuilder)
	sourceFamily := builder.Field(8).(*array.BinaryDictionaryBuilder)
	quality := builder.Field(9).(*array.BinaryDictionaryBuilder)
	state := builder.Field(10).(*array.BinaryDictionaryBuilder)

	for sampleIndex, sample := range samples {
		parsed, err := time.Parse(time.RFC3339, sample.Timestamp)
		if err != nil {
			return fmt.Errorf("sample %d timestamp: %w", sampleIndex, err)
		}
		timestampNS := parsed.UnixNano()
		signalIDs := sortedFloatKeys(sample.Signals)
		for _, signalID := range signalIDs {
			item := meta[signalID]
			ts.Append(timestampNS)
			if err := appendDict(sensor, signalID); err != nil {
				return err
			}
			value.Append(roundTelemetryValue(signalID, item.Unit, sample.Signals[signalID]))
			if err := appendDict(unit, item.Unit); err != nil {
				return err
			}
			if err := appendDict(campaign, campaignID); err != nil {
				return err
			}
			if err := appendDict(source, item.Source); err != nil {
				return err
			}
			if err := appendDict(role, item.SeriesRole); err != nil {
				return err
			}
			if err := appendDefault(kind, item.SignalKind, "numeric"); err != nil {
				return err
			}
			if err := appendDict(sourceFamily, item.SourceFamily); err != nil {
				return err
			}
			if err := appendDict(quality, sample.Quality); err != nil {
				return err
			}
			state.AppendNull()
		}
		stateIDs := sortedStringKeys(sample.States)
		for _, signalID := range stateIDs {
			item := meta[signalID]
			ts.Append(timestampNS)
			if err := appendDict(sensor, signalID); err != nil {
				return err
			}
			value.AppendNull()
			if err := appendDefault(unit, item.Unit, "state"); err != nil {
				return err
			}
			if err := appendDict(campaign, campaignID); err != nil {
				return err
			}
			if err := appendDict(source, item.Source); err != nil {
				return err
			}
			if err := appendDict(role, item.SeriesRole); err != nil {
				return err
			}
			if err := appendDefault(kind, item.SignalKind, "state"); err != nil {
				return err
			}
			if err := appendDict(sourceFamily, item.SourceFamily); err != nil {
				return err
			}
			if err := appendDict(quality, sample.Quality); err != nil {
				return err
			}
			if err := appendDict(state, sample.States[signalID]); err != nil {
				return err
			}
		}
	}

	record := builder.NewRecord()
	defer record.Release()

	writer := ipc.NewWriter(f, ipc.WithSchema(TelemetrySchema))
	if err := writer.Write(record); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return writeGzipSidecar(path)
}

func writeGzipSidecar(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	gz, err := gzip.NewWriterLevel(dst, gzip.BestCompression)
	if err != nil {
		_ = dst.Close()
		return err
	}
	if _, err := io.Copy(gz, src); err != nil {
		_ = gz.Close()
		_ = dst.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}

func roundTelemetryValue(signalID, unit string, value float64) float64 {
	if value == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return value
	}
	id := strings.ToLower(signalID)
	unit = strings.ToLower(unit)
	switch {
	case strings.HasSuffix(id, "_count") || strings.HasSuffix(id, "_counter"):
		return math.Round(value)
	case unit == "state":
		return value
	case unit == "degc" || unit == "w" || unit == "v" || unit == "a" || unit == "db" || unit == "%" || strings.Contains(id, "_pct"):
		return roundDecimal(value, 3)
	case unit == "ms" || strings.Contains(id, "latency"):
		return roundDecimal(value, 3)
	case strings.Contains(unit, "mbar") || strings.Contains(unit, "pa"):
		return roundSignificant(value, 7)
	default:
		return roundSignificant(value, 7)
	}
}

func roundDecimal(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}

func roundSignificant(value float64, digits int) float64 {
	if value == 0 {
		return 0
	}
	magnitude := math.Floor(math.Log10(math.Abs(value)))
	scale := math.Pow10(digits - 1 - int(magnitude))
	return math.Round(value*scale) / scale
}

func appendDefault(builder *array.BinaryDictionaryBuilder, value, fallback string) error {
	if value == "" {
		return appendDict(builder, fallback)
	}
	return appendDict(builder, value)
}

func appendDict(builder *array.BinaryDictionaryBuilder, value string) error {
	if value == "" {
		value = "unknown"
	}
	return builder.AppendString(value)
}

func sortedFloatKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
