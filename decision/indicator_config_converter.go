package decision

import "nofx/market"

// convertToMarketConfig 将 decision 包的 IndicatorConfig 转换为 market 包的 IndicatorConfig
func convertToMarketConfig(config *IndicatorConfig) *market.IndicatorConfig {
	if config == nil {
		return nil
	}

	return &market.IndicatorConfig{
		UseEMA20:      config.UseEMA20,
		UseMACDValues: config.UseMACDValues,
		UseRSI7:       config.UseRSI7,
		UseRSI14:      config.UseRSI14,
		UseRSI20:      config.UseRSI20,
		UseRSI25:      config.UseRSI25,
		UseRSI100:     config.UseRSI100,
		UseATR14:      config.UseATR14,
		UseATR20:      config.UseATR20,
		UseHeikinAshi: config.UseHeikinAshi,
		UseLongEMA:    config.UseLongEMA,
		UseLongATR:    config.UseLongATR,
		UseLongMACD:   config.UseLongMACD,
		UseLongRSI14:  config.UseLongRSI14,
	}
}
