package market

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FundingRateCache 资金费率缓存结构
// Binance Funding Rate 每 8 小时才更新一次，使用 1 小时缓存可显著减少 API 调用
type FundingRateCache struct {
	Rate      float64
	UpdatedAt time.Time
}

var (
	fundingRateMap sync.Map // map[string]*FundingRateCache
	frCacheTTL     = 1 * time.Hour
)

// Get 获取指定代币的市场数据
func Get(symbol string, shortTF string, longTF string) (*Data, error) {
	var klinesShort, klinesLong []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取短周期K线数据
	klinesShort, err = WSMonitorCli.GetCurrentKlines(symbol, shortTF)
	if err != nil {
		return nil, fmt.Errorf("获取%s K线失败: %v", shortTF, err)
	}

	// Data staleness detection: Prevent DOGEUSDT-style price freeze issues
	if isStaleData(klinesShort, symbol) {
		log.Printf("⚠️  WARNING: %s detected stale data (consecutive price freeze), skipping symbol", symbol)
		return nil, fmt.Errorf("%s data is stale, possible cache failure", symbol)
	}

	// 获取长周期K线数据
	klinesLong, err = WSMonitorCli.GetCurrentKlines(symbol, longTF)
	if err != nil {
		return nil, fmt.Errorf("获取%s K线失败: %v", longTF, err)
	}

	// 检查数据是否为空
	if len(klinesShort) == 0 {
		return nil, fmt.Errorf("%s K线数据为空", shortTF)
	}
	if len(klinesLong) == 0 {
		return nil, fmt.Errorf("%s K线数据为空", longTF)
	}

	// 计算当前指标 (基于短周期最新数据)
	currentPrice := klinesShort[len(klinesShort)-1].Close
	currentEMA20 := calculateEMA(klinesShort, 20)
	currentMACD := calculateMACD(klinesShort)
	currentRSI7 := calculateRSI(klinesShort, 7)

	// 计算价格变化百分比
	// 短周期价格变化 (例如：20个短周期K线前的价格)
	priceChange1h := 0.0
	if len(klinesShort) >= 21 { // 至少需要21根K线 (当前 + 20根前)
		price1hAgo := klinesShort[len(klinesShort)-21].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 长周期价格变化 (例如：1个长周期K线前的价格)
	priceChange4h := 0.0
	if len(klinesLong) >= 2 {
		price4hAgo := klinesLong[len(klinesLong)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据 (基于短周期)
	intradayData := calculateIntradaySeries(klinesShort)

	// 计算长期数据 (基于长周期)
	longerTermData := calculateLongerTermData(klinesLong)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		LongerTermContext: longerTermData,
		ShortTimeframe:    shortTF,
		LongTimeframe:     longTF,
	}, nil
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:    make([]float64, 0, 10),
		EMA20Values:  make([]float64, 0, 10),
		MACDValues:   make([]float64, 0, 10),
		RSI7Values:   make([]float64, 0, 10),
		RSI14Values:  make([]float64, 0, 10),
		RSI20Values:  make([]float64, 0, 10),
		RSI25Values:  make([]float64, 0, 10),
		RSI100Values: make([]float64, 0, 10),
		Volume:       make([]float64, 0, 10),
		HAOpen:       make([]float64, 0, 10),
		HAClose:      make([]float64, 0, 10),
		HAHigh:       make([]float64, 0, 10),
		HALow:        make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
		if i >= 20 {
			rsi20 := calculateRSI(klines[:i+1], 20)
			data.RSI20Values = append(data.RSI20Values, rsi20)
		}
		if i >= 25 {
			rsi25 := calculateRSI(klines[:i+1], 25)
			data.RSI25Values = append(data.RSI25Values, rsi25)
		}
		if i >= 100 {
			rsi100 := calculateRSI(klines[:i+1], 100)
			data.RSI100Values = append(data.RSI100Values, rsi100)
		}
	}

	// 计算 Heikin Ashi 数据
	haData := calculateHeikinAshi(klines)
	// 获取最近10个 Heikin Ashi 数据点
	haStart := len(haData) - 10
	if haStart < 0 {
		haStart = 0
	}
	for i := haStart; i < len(haData); i++ {
		data.HAOpen = append(data.HAOpen, haData[i].Open)
		data.HAClose = append(data.HAClose, haData[i].Close)
		data.HAHigh = append(data.HAHigh, haData[i].High)
		data.HALow = append(data.HALow, haData[i].Low)
	}

	// 计算 ATR
	data.ATR14 = calculateATR(klines, 14)
	data.ATR20 = calculateATR(klines, 20)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率（优化：使用 1 小时缓存）
func getFundingRate(symbol string) (float64, error) {
	// 检查缓存（有效期 1 小时）
	// Funding Rate 每 8 小时才更新，1 小时缓存非常合理
	if cached, ok := fundingRateMap.Load(symbol); ok {
		cache := cached.(*FundingRateCache)
		if time.Since(cache.UpdatedAt) < frCacheTTL {
			// 缓存命中，直接返回
			return cache.Rate, nil
		}
	}

	// 缓存过期或不存在，调用 API
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)

	// 更新缓存
	fundingRateMap.Store(symbol, &FundingRateCache{
		Rate:      rate,
		UpdatedAt: time.Now(),
	})

	return rate, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	// 只输出价格，不输出旧指标（EMA20, MACD, RSI7）以避免与新策略混淆
	sb.WriteString(fmt.Sprintf("current_price = %s\n\n", priceStr))

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil {
		// 使用动态精度格式化 OI 数据
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

		if len(data.IntradaySeries.RSI25Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (25‑Period, TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI25Values)))
		}

		if len(data.IntradaySeries.RSI100Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (100‑Period, TradingView Strategy): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI100Values)))
		}

		if len(data.IntradaySeries.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
		}

		sb.WriteString(fmt.Sprintf("%s ATR (20‑period, TradingView Strategy): %.3f\n\n", data.ShortTimeframe, data.IntradaySeries.ATR20))

		// Heikin Ashi 数据
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

	if data.LongerTermContext != nil {
		sb.WriteString(fmt.Sprintf("Longer‑term context (%s timeframe):\n\n", data.LongTimeframe))

		// 长周期数据仅作为背景参考，不作为主要决策依据
		// 旧指标已注释，避免与新策略混淆
		// sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
		// 	data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))
		// sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f vs. 14‑Period ATR: %.3f\n\n",
		// 	data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		// MACD 和 RSI14 不在新策略中使用
		// if len(data.LongerTermContext.MACDValues) > 0 {
		// 	sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		// }
		// if len(data.LongerTermContext.RSI14Values) > 0 {
		// 	sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		// }
	}

	return sb.String()
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// isStaleData detects stale data (consecutive price freeze)
// Fix DOGEUSDT-style issue: consecutive N periods with completely unchanged prices indicate data source anomaly
func isStaleData(klines []Kline, symbol string) bool {
	if len(klines) < 5 {
		return false // Insufficient data to determine
	}

	// Detection threshold: 5 consecutive 3-minute periods with unchanged price (15 minutes without fluctuation)
	const stalePriceThreshold = 5
	const priceTolerancePct = 0.0001 // 0.01% fluctuation tolerance (avoid false positives)

	// Take the last stalePriceThreshold K-lines
	recentKlines := klines[len(klines)-stalePriceThreshold:]
	firstPrice := recentKlines[0].Close

	// Check if all prices are within tolerance
	for i := 1; i < len(recentKlines); i++ {
		priceDiff := math.Abs(recentKlines[i].Close-firstPrice) / firstPrice
		if priceDiff > priceTolerancePct {
			return false // Price fluctuation exists, data is normal
		}
	}

	// Additional check: MACD and volume
	// If price is unchanged but MACD/volume shows normal fluctuation, it might be a real market situation (extremely low volatility)
	// Check if volume is also 0 (data completely frozen)
	allVolumeZero := true
	for _, k := range recentKlines {
		if k.Volume > 0 {
			allVolumeZero = false
			break
		}
	}

	if allVolumeZero {
		log.Printf("⚠️  %s stale data confirmed: price freeze + zero volume", symbol)
		return true
	}

	// Price frozen but has volume: might be extremely low volatility market, allow but log warning
	log.Printf("⚠️  %s detected extreme price stability (no fluctuation for %d consecutive periods), but volume is normal", symbol, stalePriceThreshold)
	return false
}

// calculateHeikinAshi 计算 Heikin Ashi 平滑K线
func calculateHeikinAshi(klines []Kline) []Kline {
	if len(klines) == 0 {
		return []Kline{}
	}

	haKlines := make([]Kline, len(klines))
	
	// 第一根 Heikin Ashi K线
	haKlines[0] = Kline{
		Open:   (klines[0].Open + klines[0].Close) / 2,
		Close:  (klines[0].Open + klines[0].High + klines[0].Low + klines[0].Close) / 4,
		High:   klines[0].High,
		Low:    klines[0].Low,
		Volume: klines[0].Volume,
	}

	// 后续 Heikin Ashi K线
	for i := 1; i < len(klines); i++ {
		// HA Close = (Open + High + Low + Close) / 4
		haClose := (klines[i].Open + klines[i].High + klines[i].Low + klines[i].Close) / 4
		
		// HA Open = (Previous HA Open + Previous HA Close) / 2
		haOpen := (haKlines[i-1].Open + haKlines[i-1].Close) / 2
		
		// HA High = Max(High, HA Open, HA Close)
		haHigh := math.Max(klines[i].High, math.Max(haOpen, haClose))
		
		// HA Low = Min(Low, HA Open, HA Close)
		haLow := math.Min(klines[i].Low, math.Min(haOpen, haClose))
		
		haKlines[i] = Kline{
			Open:   haOpen,
			Close:  haClose,
			High:   haHigh,
			Low:    haLow,
			Volume: klines[i].Volume,
		}
	}

	return haKlines
}
