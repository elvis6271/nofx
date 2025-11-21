package decision

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PromptTemplate 系统提示词模板
type PromptTemplate struct {
	Name              string            // 模板名称（文件名，不含扩展名）
	Content           string            // 模板内容
	IndicatorConfig   *IndicatorConfig  // 指标配置（从模板头部解析）
}

// IndicatorConfig 指标配置
type IndicatorConfig struct {
	// 短周期指标
	UseEMA20    bool
	UseMACDValues bool
	UseRSI7     bool
	UseRSI14    bool
	UseRSI20    bool
	UseRSI25    bool
	UseRSI100   bool
	UseATR14    bool
	UseATR20    bool
	UseHeikinAshi bool
	
	// 长周期指标
	UseLongEMA    bool
	UseLongATR    bool
	UseLongMACD   bool
	UseLongRSI14  bool
}

// PromptManager 提示词管理器
type PromptManager struct {
	templates map[string]*PromptTemplate
	mu        sync.RWMutex
}

var (
	// globalPromptManager 全局提示词管理器
	globalPromptManager *PromptManager
	// promptsDir 提示词文件夹路径
	promptsDir = "prompts"
)

// init 包初始化时加载所有提示词模板
func init() {
	globalPromptManager = NewPromptManager()
	if err := globalPromptManager.LoadTemplates(promptsDir); err != nil {
		log.Printf("⚠️  加载提示词模板失败: %v", err)
	} else {
		log.Printf("✓ 已加载 %d 个系统提示词模板", len(globalPromptManager.templates))
	}
}

// NewPromptManager 创建提示词管理器
func NewPromptManager() *PromptManager {
	return &PromptManager{
		templates: make(map[string]*PromptTemplate),
	}
}

// LoadTemplates 从指定目录加载所有提示词模板
func (pm *PromptManager) LoadTemplates(dir string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("提示词目录不存在: %s", dir)
	}

	// 扫描目录中的所有 .txt 文件
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return fmt.Errorf("扫描提示词目录失败: %w", err)
	}

	if len(files) == 0 {
		log.Printf("⚠️  提示词目录 %s 中没有找到 .txt 文件", dir)
		return nil
	}

	// 加载每个模板文件
	for _, file := range files {
		// 读取文件内容
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("⚠️  读取提示词文件失败 %s: %v", file, err)
			continue
		}

		// 提取文件名（不含扩展名）作为模板名称
		fileName := filepath.Base(file)
		templateName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		// 解析指标配置
		indicatorConfig := parseIndicatorConfig(string(content))

		// 存储模板
		pm.templates[templateName] = &PromptTemplate{
			Name:            templateName,
			Content:         string(content),
			IndicatorConfig: indicatorConfig,
		}

		log.Printf("  📄 加载提示词模板: %s (%s)", templateName, fileName)
	}

	return nil
}

// GetTemplate 获取指定名称的提示词模板
func (pm *PromptManager) GetTemplate(name string) (*PromptTemplate, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	template, exists := pm.templates[name]
	if !exists {
		return nil, fmt.Errorf("提示词模板不存在: %s", name)
	}

	return template, nil
}

// GetAllTemplateNames 获取所有模板名称列表
func (pm *PromptManager) GetAllTemplateNames() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.templates))
	for name := range pm.templates {
		names = append(names, name)
	}

	return names
}

// GetAllTemplates 获取所有模板
func (pm *PromptManager) GetAllTemplates() []*PromptTemplate {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	templates := make([]*PromptTemplate, 0, len(pm.templates))
	for _, template := range pm.templates {
		templates = append(templates, template)
	}

	return templates
}

// ReloadTemplates 重新加载所有模板
func (pm *PromptManager) ReloadTemplates(dir string) error {
	pm.mu.Lock()
	pm.templates = make(map[string]*PromptTemplate)
	pm.mu.Unlock()

	return pm.LoadTemplates(dir)
}

// === 全局函数（供外部调用）===

// GetPromptTemplate 获取指定名称的提示词模板（全局函数）
func GetPromptTemplate(name string) (*PromptTemplate, error) {
	return globalPromptManager.GetTemplate(name)
}

// GetAllPromptTemplateNames 获取所有模板名称（全局函数）
func GetAllPromptTemplateNames() []string {
	return globalPromptManager.GetAllTemplateNames()
}

// GetAllPromptTemplates 获取所有模板（全局函数）
func GetAllPromptTemplates() []*PromptTemplate {
	return globalPromptManager.GetAllTemplates()
}

// ReloadPromptTemplates 重新加载所有模板（全局函数）
func ReloadPromptTemplates() error {
	return globalPromptManager.ReloadTemplates(promptsDir)
}

// parseIndicatorConfig 从提示词内容中自动检测使用的指标
// 支持三种模式：
// 1. 显式配置（YAML格式）：
//    ---indicators
//    rsi25: true
//    ---
// 2. 显式配置（注释格式）：
//    # @indicators: rsi25,rsi100,atr20
// 3. 自动检测（默认）：分析提示词内容，检测提到的指标关键词
func parseIndicatorConfig(content string) *IndicatorConfig {
	// 先尝试解析显式配置
	explicitConfig := parseExplicitConfig(content)
	if explicitConfig != nil {
		return explicitConfig
	}
	
	// 没有显式配置，使用自动检测
	return autoDetectIndicators(content)
}

// parseExplicitConfig 解析显式指标配置
func parseExplicitConfig(content string) *IndicatorConfig {
	config := &IndicatorConfig{}
	hasExplicitConfig := false
	
	lines := strings.Split(content, "\n")
	inIndicatorBlock := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 检查 YAML 格式
		if line == "---indicators" {
			inIndicatorBlock = true
			hasExplicitConfig = true
			continue
		}
		if line == "---" && inIndicatorBlock {
			break
		}
		
		if inIndicatorBlock {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				enabled := value == "true" || value == "yes" || value == "1"
				setIndicatorFlag(config, key, enabled)
			}
			continue
		}
		
		// 检查注释格式
		if strings.HasPrefix(line, "#") && strings.Contains(line, "@indicators:") {
			hasExplicitConfig = true
			parts := strings.SplitN(line, "@indicators:", 2)
			if len(parts) == 2 {
				indicators := strings.Split(parts[1], ",")
				for _, ind := range indicators {
					setIndicatorFlag(config, strings.TrimSpace(ind), true)
				}
			}
			break
		}
		
		// 只检查前面几行
		if len(line) > 0 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			break
		}
	}
	
	if hasExplicitConfig {
		return config
	}
	return nil
}

// autoDetectIndicators 自动检测提示词中使用的指标
func autoDetectIndicators(content string) *IndicatorConfig {
	config := &IndicatorConfig{}
	contentLower := strings.ToLower(content)
	
	// 检测短周期指标
	// EMA20
	if strings.Contains(contentLower, "ema") && 
	   (strings.Contains(contentLower, "ema20") || strings.Contains(contentLower, "ema(20") || 
	    strings.Contains(contentLower, "20-period ema") || strings.Contains(contentLower, "20周期ema")) {
		config.UseEMA20 = true
	}
	
	// MACD
	if strings.Contains(contentLower, "macd") {
		config.UseMACDValues = true
	}
	
	// RSI 各周期
	if strings.Contains(contentLower, "rsi") {
		if strings.Contains(contentLower, "rsi7") || strings.Contains(contentLower, "rsi(7") || 
		   strings.Contains(contentLower, "7-period rsi") || strings.Contains(contentLower, "7周期rsi") {
			config.UseRSI7 = true
		}
		if strings.Contains(contentLower, "rsi14") || strings.Contains(contentLower, "rsi(14") || 
		   strings.Contains(contentLower, "14-period rsi") || strings.Contains(contentLower, "14周期rsi") {
			config.UseRSI14 = true
		}
		if strings.Contains(contentLower, "rsi20") || strings.Contains(contentLower, "rsi(20") || 
		   strings.Contains(contentLower, "20-period rsi") || strings.Contains(contentLower, "20周期rsi") {
			config.UseRSI20 = true
		}
		if strings.Contains(contentLower, "rsi25") || strings.Contains(contentLower, "rsi(25") || 
		   strings.Contains(contentLower, "25-period rsi") || strings.Contains(contentLower, "25周期rsi") {
			config.UseRSI25 = true
		}
		if strings.Contains(contentLower, "rsi100") || strings.Contains(contentLower, "rsi(100") || 
		   strings.Contains(contentLower, "100-period rsi") || strings.Contains(contentLower, "100周期rsi") {
			config.UseRSI100 = true
		}
	}
	
	// ATR
	if strings.Contains(contentLower, "atr") {
		if strings.Contains(contentLower, "atr14") || strings.Contains(contentLower, "atr(14") || 
		   strings.Contains(contentLower, "14-period atr") || strings.Contains(contentLower, "14周期atr") {
			config.UseATR14 = true
		}
		if strings.Contains(contentLower, "atr20") || strings.Contains(contentLower, "atr(20") || 
		   strings.Contains(contentLower, "20-period atr") || strings.Contains(contentLower, "20周期atr") {
			config.UseATR20 = true
		}
	}
	
	// Heikin Ashi
	if strings.Contains(contentLower, "heikin") || strings.Contains(contentLower, "平滑k线") {
		config.UseHeikinAshi = true
	}
	
	// 检测长周期指标
	if strings.Contains(contentLower, "长周期") || strings.Contains(contentLower, "longer-term") || 
	   strings.Contains(contentLower, "4h") || strings.Contains(contentLower, "1h") {
		// 如果提到长周期上下文，启用长周期指标
		if strings.Contains(contentLower, "ema") {
			config.UseLongEMA = true
		}
		if strings.Contains(contentLower, "atr") && !config.UseATR14 && !config.UseATR20 {
			config.UseLongATR = true
		}
		if strings.Contains(contentLower, "macd") && !config.UseMACDValues {
			config.UseLongMACD = true
		}
		if strings.Contains(contentLower, "rsi") && !config.UseRSI7 && !config.UseRSI14 && 
		   !config.UseRSI20 && !config.UseRSI25 && !config.UseRSI100 {
			config.UseLongRSI14 = true
		}
	}
	
	// 如果没有检测到任何指标，使用所有指标（向后兼容旧提示词）
	if !hasAnyIndicator(config) {
		return getDefaultAllConfig()
	}
	
	return config
}

// setIndicatorFlag 设置指标标志
func setIndicatorFlag(config *IndicatorConfig, key string, enabled bool) {
	switch key {
	case "ema20":
		config.UseEMA20 = enabled
	case "macd":
		config.UseMACDValues = enabled
	case "rsi7":
		config.UseRSI7 = enabled
	case "rsi14":
		config.UseRSI14 = enabled
	case "rsi20":
		config.UseRSI20 = enabled
	case "rsi25":
		config.UseRSI25 = enabled
	case "rsi100":
		config.UseRSI100 = enabled
	case "atr14":
		config.UseATR14 = enabled
	case "atr20":
		config.UseATR20 = enabled
	case "heikin_ashi", "heikin-ashi", "heikinashi":
		config.UseHeikinAshi = enabled
	case "long_ema":
		config.UseLongEMA = enabled
	case "long_atr":
		config.UseLongATR = enabled
	case "long_macd":
		config.UseLongMACD = enabled
	case "long_rsi14":
		config.UseLongRSI14 = enabled
	}
}

// hasAnyIndicator 检查是否有任何指标被启用
func hasAnyIndicator(config *IndicatorConfig) bool {
	return config.UseEMA20 || config.UseMACDValues || config.UseRSI7 || config.UseRSI14 ||
		config.UseRSI20 || config.UseRSI25 || config.UseRSI100 || config.UseATR14 ||
		config.UseATR20 || config.UseHeikinAshi || config.UseLongEMA || config.UseLongATR ||
		config.UseLongMACD || config.UseLongRSI14
}

// getDefaultAllConfig 返回默认配置（所有指标）
func getDefaultAllConfig() *IndicatorConfig {
	return &IndicatorConfig{
		UseEMA20:      true,
		UseMACDValues: true,
		UseRSI7:       true,
		UseRSI14:      true,
		UseRSI20:      true,
		UseRSI25:      true,
		UseRSI100:     true,
		UseATR14:      true,
		UseATR20:      true,
		UseHeikinAshi: true,
		UseLongEMA:    true,
		UseLongATR:    true,
		UseLongMACD:   true,
		UseLongRSI14:  true,
	}
}
