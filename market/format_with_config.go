package market

import (
	"fmt"
	"strings"
)

// IndicatorConfig 指标配置（从 decision 包复制，避免循环依赖）
type IndicatorConfig struct {
	// 短周期指标
	UseEMA20      bool
	UseMACDValues bool
	UseRSI7       bool
	UseRSI14      bool
	UseRSI20      bool
	UseRSI25      bool
	UseRSI100     bool
	UseATR14      bool
	UseATR20      bool
	UseHeikinAshi bool

	// 长周期指标
	UseLongEMA   bool
	UseLongATR   bool
	UseLongMACD  bool
	UseLongRSI14 bool
}

// FormatWithConfig 根据指标配置格式化输出市场数据
func FormatWithConfig(data *Data, config *IndicatorConfig) string {
	// 如果没有配置，使用默认的 Format 函数（所有指标）
	if config == nil {
		return Format(data)
	}

	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)

	// 输出基础指标（根据配置）
	if config.UseEMA20 || config.UseMACDValues || config.UseRSI7 {
		sb.WriteString(fmt.Sprintf("current_price = %s", priceStr))
		if config.UseEMA20 {
			sb.WriteString(fmt.Sprintf(", current_ema20 = %.3f", data.CurrentEMA20))
		}
		if config.UseMACDValues {
			sb.WriteString(fmt.Sprintf(", current_macd = %.3f", data.CurrentMACD))
		}
		if config.UseRSI7 {
			sb.WriteString(fmt.Sprintf(", current_rsi (7 period) = %.3f", data.CurrentRSI7))
		}
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("current_price = %s\n\n", priceStr))
	}

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil {
		oiLatestStr := formatPriceWithDynamicPrecision(data.OpenInterest.Latest)
		oiAverageStr := formatPriceWithDynamicPrecision(data.OpenInterest.Average)
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %s Average: %s\n\n",
			oiLatestStr, oiAverageStr))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		sb.WriteString(fmt.Sprintf("Intraday series (%s intervals, oldest → latest):\n\n", data.ShortTimeframe))

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if config.UseEMA20 && len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if config.UseMACDValues && len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if config.UseRSI7 && len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if config.UseRSI14 && len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}

		if config.UseRSI20 && len(data.IntradaySeries.RSI20Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (20‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI20Values)))
		}

		if config.UseRSI25 && len(data.IntradaySeries.RSI25Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (25‑Period, TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI25Values)))
		}

		if config.UseRSI100 && len(data.IntradaySeries.RSI100Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (100‑Period, TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI100Values)))
		}

		if len(data.IntradaySeries.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
		}

		if config.UseATR14 {
			sb.WriteString(fmt.Sprintf("%s ATR (14‑period): %.3f\n\n", data.ShortTimeframe, data.IntradaySeries.ATR14))
		}
		if config.UseATR20 {
			sb.WriteString(fmt.Sprintf("%s ATR (20‑period, TradingView Strategy): %.3f\n\n", data.ShortTimeframe, data.IntradaySeries.ATR20))
		}

		// Heikin Ashi 数据
		if config.UseHeikinAshi {
			if len(data.IntradaySeries.HAOpen) > 0 {
				sb.WriteString(fmt.Sprintf("Heikin Ashi Open (TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.HAOpen)))
			}
			if len(data.IntradaySeries.HAClose) > 0 {
				sb.WriteString(fmt.Sprintf("Heikin Ashi Close (TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.HAClose)))
			}
			if len(data.IntradaySeries.HAHigh) > 0 {
				sb.WriteString(fmt.Sprintf("Heikin Ashi High (TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.HAHigh)))
			}
			if len(data.IntradaySeries.HALow) > 0 {
				sb.WriteString(fmt.Sprintf("Heikin Ashi Low (TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.HALow)))
			}
		}
	}

	if data.LongerTermContext != nil {
		sb.WriteString(fmt.Sprintf("Longer‑term context (%s timeframe):\n\n", data.LongTimeframe))

		if config.UseLongEMA {
			sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
				data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))
		}

		if config.UseLongATR {
			sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f vs. 14‑Period ATR: %.3f\n\n",
				data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))
		}

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		if config.UseLongMACD && len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if config.UseLongRSI14 && len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}
	}

	return sb.String()
}
