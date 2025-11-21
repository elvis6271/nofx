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

// parseIndicatorConfig 从提示词内容中解析指标配置
// 支持两种格式：
// 1. YAML格式（在文件开头）：
//    ---indicators
//    rsi25: true
//    rsi100: true
//    ---
// 2. 注释格式（兼容旧提示词）：
//    # @indicators: rsi25,rsi100,atr20,heikin_ashi
func parseIndicatorConfig(content string) *IndicatorConfig {
	config := &IndicatorConfig{}
	
	// 默认配置：如果没有指定，则使用所有指标（向后兼容）
	defaultAll := true
	
	lines := strings.Split(content, "\n")
	inIndicatorBlock := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 检查 YAML 格式的指标配置块
		if line == "---indicators" {
			inIndicatorBlock = true
			defaultAll = false
			continue
		}
		if line == "---" && inIndicatorBlock {
			break
		}
		
		// 解析 YAML 格式的配置
		if inIndicatorBlock {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				enabled := value == "true" || value == "yes" || value == "1"
				
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
				case "heikin_ashi":
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
			continue
		}
		
		// 检查注释格式的配置
		if strings.HasPrefix(line, "#") && strings.Contains(line, "@indicators:") {
			defaultAll = false
			// 提取指标列表
			parts := strings.SplitN(line, "@indicators:", 2)
			if len(parts) == 2 {
				indicators := strings.Split(parts[1], ",")
				for _, ind := range indicators {
					ind = strings.TrimSpace(ind)
					switch ind {
					case "ema20":
						config.UseEMA20 = true
					case "macd":
						config.UseMACDValues = true
					case "rsi7":
						config.UseRSI7 = true
					case "rsi14":
						config.UseRSI14 = true
					case "rsi20":
						config.UseRSI20 = true
					case "rsi25":
						config.UseRSI25 = true
					case "rsi100":
						config.UseRSI100 = true
					case "atr14":
						config.UseATR14 = true
					case "atr20":
						config.UseATR20 = true
					case "heikin_ashi":
						config.UseHeikinAshi = true
					case "long_ema":
						config.UseLongEMA = true
					case "long_atr":
						config.UseLongATR = true
					case "long_macd":
						config.UseLongMACD = true
					case "long_rsi14":
						config.UseLongRSI14 = true
					}
				}
			}
			break
		}
		
		// 如果已经过了前几行还没找到配置，就停止查找
		if len(line) > 0 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			break
		}
	}
	
	// 如果没有找到配置，使用默认配置（所有指标都启用，向后兼容）
	if defaultAll {
		config.UseEMA20 = true
		config.UseMACDValues = true
		config.UseRSI7 = true
		config.UseRSI14 = true
		config.UseRSI20 = true
		config.UseRSI25 = true
		config.UseRSI100 = true
		config.UseATR14 = true
		config.UseATR20 = true
		config.UseHeikinAshi = true
		config.UseLongEMA = true
		config.UseLongATR = true
		config.UseLongMACD = true
		config.UseLongRSI14 = true
	}
	
	return config
}
