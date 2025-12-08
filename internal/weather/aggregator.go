package weather

// AggregateCurrentWeather combines multiple CurrentWeather results into one.
//
// For now it returns the first successful entry. Later this function can be
// extended to compute averages for temperature, humidity, wind speed and other
// numeric fields, as well as to merge metadata (sources, confidence, etc.).
func AggregateCurrentWeather(results []CurrentWeather) CurrentWeather {
	if len(results) == 0 {
		return CurrentWeather{}
	}

	// TODO: implement real aggregation logic (averages, merge sources, etc.).
	return results[0]
}

// AggregateForecast combines multiple Forecast results into one.
//
// For now it returns the first successful entry. Later this function can be
// extended to merge time series, deduplicate timestamps, and average numeric
// values across providers.
func AggregateForecast(results []Forecast) Forecast {
	if len(results) == 0 {
		return Forecast{}
	}

	// TODO: implement real aggregation logic when multiple providers are live.
	return results[0]
}
