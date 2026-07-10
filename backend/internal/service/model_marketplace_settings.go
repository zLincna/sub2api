package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	modelMarketplaceMaxModels = 200
	modelMarketplaceMaxBytes  = 256 * 1024
)

// ModelMarketplaceConfig is display-only configuration for the user model
// marketplace. Array order is the display order within each platform.
type ModelMarketplaceConfig struct {
	Models []ModelMarketplaceModel `json:"models"`
}

type ModelMarketplaceModel struct {
	ID                 string   `json:"id"`
	Platform           string   `json:"platform"`
	Description        string   `json:"description"`
	ChannelName        string   `json:"channel_name"`
	ChannelDescription string   `json:"channel_description"`
	GroupName          string   `json:"group_name"`
	RateMultiplier     float64  `json:"rate_multiplier"`
	BillingMode        string   `json:"billing_mode"`
	InputPrice         *float64 `json:"input_price_per_million"`
	OutputPrice        *float64 `json:"output_price_per_million"`
	CacheWritePrice    *float64 `json:"cache_write_price_per_million"`
	CacheReadPrice     *float64 `json:"cache_read_price_per_million"`
	ImageOutputPrice   *float64 `json:"image_output_price_per_request"`
	PerRequestPrice    *float64 `json:"per_request_price"`
	Enabled            bool     `json:"enabled"`
}

func marketplacePrice(value float64) *float64 {
	return &value
}

func defaultModelMarketplaceConfig() ModelMarketplaceConfig {
	openAIChannel := "OpenAI Pro 20x"
	openAIGroup := "OpenAI Pro 20x 号池"
	claudeChannel := "Claude Code 低倍率"
	claudeGroup := "Claude Code 低倍率"
	return ModelMarketplaceConfig{Models: []ModelMarketplaceModel{
		{ID: "gpt-5.6-sol", Platform: "openai", Description: "支持超长上下文、视觉输入、工具调用与高强度推理任务。", ChannelName: openAIChannel, ChannelDescription: "稳定 20x Pro 号池，适合 Codex、IDE 与高强度代码任务。", GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(5), OutputPrice: marketplacePrice(30), CacheReadPrice: marketplacePrice(0.5), Enabled: true},
		{ID: "gpt-5.6-terra", Platform: "openai", Description: "支持超长上下文、视觉输入、工具调用与复杂工程任务。", ChannelName: openAIChannel, ChannelDescription: "稳定 20x Pro 号池，适合 Codex、IDE 与高强度代码任务。", GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(5), OutputPrice: marketplacePrice(30), CacheReadPrice: marketplacePrice(0.5), Enabled: true},
		{ID: "gpt-5.6-luna", Platform: "openai", Description: "支持超长上下文、视觉输入、工具调用与深度推理。", ChannelName: openAIChannel, ChannelDescription: "稳定 20x Pro 号池，适合 Codex、IDE 与高强度代码任务。", GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(5), OutputPrice: marketplacePrice(30), CacheReadPrice: marketplacePrice(0.5), Enabled: true},
		{ID: "gpt-5-codex", Platform: "openai", Description: "适合 Codex、IDE 与高强度代码任务。", ChannelName: openAIChannel, ChannelDescription: "稳定 20x Pro 号池。", GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(1.25), OutputPrice: marketplacePrice(10), CacheWritePrice: marketplacePrice(1.25), CacheReadPrice: marketplacePrice(0.125), Enabled: true},
		{ID: "gpt-5", Platform: "openai", Description: "通用旗舰模型，适合复杂推理、代码与长上下文任务。", ChannelName: openAIChannel, GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(1.25), OutputPrice: marketplacePrice(10), CacheWritePrice: marketplacePrice(1.25), CacheReadPrice: marketplacePrice(0.125), Enabled: true},
		{ID: "gpt-5-mini", Platform: "openai", Description: "高性价比快速模型，适合日常问答与轻量编码。", ChannelName: openAIChannel, GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(0.25), OutputPrice: marketplacePrice(2), CacheWritePrice: marketplacePrice(0.25), CacheReadPrice: marketplacePrice(0.025), Enabled: true},
		{ID: "gpt-4.1", Platform: "openai", Description: "稳定通用模型，兼容大量 OpenAI 生态客户端。", ChannelName: openAIChannel, GroupName: openAIGroup, RateMultiplier: 0.1, BillingMode: "token", InputPrice: marketplacePrice(2), OutputPrice: marketplacePrice(8), CacheWritePrice: marketplacePrice(0.5), CacheReadPrice: marketplacePrice(0.5), Enabled: true},
		{ID: "claude-sonnet-4-5", Platform: "anthropic", Description: "Claude Code 主力模型，适合代码生成、重构与复杂项目执行。", ChannelName: claudeChannel, GroupName: claudeGroup, RateMultiplier: 0.02, BillingMode: "token", InputPrice: marketplacePrice(3), OutputPrice: marketplacePrice(15), CacheWritePrice: marketplacePrice(3.75), CacheReadPrice: marketplacePrice(0.3), Enabled: true},
		{ID: "claude-opus-4-1", Platform: "anthropic", Description: "顶级推理与复杂代码任务模型。", ChannelName: "Claude Code 顶级模型", GroupName: "Claude Code 顶级模型", RateMultiplier: 0.04, BillingMode: "token", InputPrice: marketplacePrice(15), OutputPrice: marketplacePrice(75), CacheWritePrice: marketplacePrice(18.75), CacheReadPrice: marketplacePrice(1.5), Enabled: true},
		{ID: "claude-sonnet-4", Platform: "anthropic", Description: "成熟稳定的 Claude Code 模型，适合日常开发与长任务。", ChannelName: claudeChannel, GroupName: claudeGroup, RateMultiplier: 0.02, BillingMode: "token", InputPrice: marketplacePrice(3), OutputPrice: marketplacePrice(15), CacheWritePrice: marketplacePrice(3.75), CacheReadPrice: marketplacePrice(0.3), Enabled: true},
		{ID: "claude-haiku-4-5", Platform: "anthropic", Description: "快速低成本模型，适合轻量代码、摘要和批量处理。", ChannelName: claudeChannel, GroupName: claudeGroup, RateMultiplier: 0.02, BillingMode: "token", InputPrice: marketplacePrice(1), OutputPrice: marketplacePrice(5), CacheWritePrice: marketplacePrice(1.25), CacheReadPrice: marketplacePrice(0.1), Enabled: true},
	}}
}

func (s *SettingService) GetModelMarketplaceConfig(ctx context.Context) (ModelMarketplaceConfig, error) {
	if s == nil || s.settingRepo == nil {
		return ModelMarketplaceConfig{}, fmt.Errorf("model marketplace setting repository is not configured")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelMarketplaceConfig)
	if errors.Is(err, ErrSettingNotFound) || strings.TrimSpace(raw) == "" {
		return defaultModelMarketplaceConfig(), nil
	}
	if err != nil {
		return ModelMarketplaceConfig{}, fmt.Errorf("get model marketplace config: %w", err)
	}
	var config ModelMarketplaceConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return ModelMarketplaceConfig{}, fmt.Errorf("decode model marketplace config: %w", err)
	}
	return normalizeModelMarketplaceConfig(config)
}

func (s *SettingService) UpdateModelMarketplaceConfig(ctx context.Context, config ModelMarketplaceConfig) (ModelMarketplaceConfig, error) {
	normalized, err := normalizeModelMarketplaceConfig(config)
	if err != nil {
		return ModelMarketplaceConfig{}, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return ModelMarketplaceConfig{}, fmt.Errorf("encode model marketplace config: %w", err)
	}
	if len(raw) > modelMarketplaceMaxBytes {
		return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_CONFIG_TOO_LARGE", "model marketplace config is too large")
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelMarketplaceConfig, string(raw)); err != nil {
		return ModelMarketplaceConfig{}, fmt.Errorf("save model marketplace config: %w", err)
	}
	return normalized, nil
}

func normalizeModelMarketplaceConfig(config ModelMarketplaceConfig) (ModelMarketplaceConfig, error) {
	if len(config.Models) > modelMarketplaceMaxModels {
		return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_TOO_MANY_MODELS", "model marketplace supports at most 200 models")
	}
	seen := make(map[string]struct{}, len(config.Models))
	result := ModelMarketplaceConfig{Models: make([]ModelMarketplaceModel, 0, len(config.Models))}
	for index, model := range config.Models {
		model.ID = strings.TrimSpace(model.ID)
		model.Platform = strings.ToLower(strings.TrimSpace(model.Platform))
		model.Description = strings.TrimSpace(model.Description)
		model.ChannelName = strings.TrimSpace(model.ChannelName)
		model.ChannelDescription = strings.TrimSpace(model.ChannelDescription)
		model.GroupName = strings.TrimSpace(model.GroupName)
		model.BillingMode = strings.ToLower(strings.TrimSpace(model.BillingMode))
		if model.ID == "" || len(model.ID) > 160 {
			return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_INVALID_MODEL_ID", fmt.Sprintf("model %d has an invalid model ID", index+1))
		}
		if model.Platform == "" || len(model.Platform) > 60 {
			return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_INVALID_PLATFORM", fmt.Sprintf("model %s has an invalid platform", model.ID))
		}
		key := model.Platform + "\x00" + strings.ToLower(model.ID)
		if _, exists := seen[key]; exists {
			return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_DUPLICATE_MODEL", fmt.Sprintf("duplicate model: %s", model.ID))
		}
		seen[key] = struct{}{}
		if model.BillingMode == "" {
			model.BillingMode = "token"
		}
		if model.BillingMode != "token" && model.BillingMode != "per_request" && model.BillingMode != "image" {
			return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_INVALID_BILLING_MODE", fmt.Sprintf("model %s has an invalid billing mode", model.ID))
		}
		if model.RateMultiplier < 0 || model.RateMultiplier > 1000 {
			return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_INVALID_RATE", fmt.Sprintf("model %s has an invalid rate multiplier", model.ID))
		}
		for _, price := range []*float64{model.InputPrice, model.OutputPrice, model.CacheWritePrice, model.CacheReadPrice, model.ImageOutputPrice, model.PerRequestPrice} {
			if price != nil && (*price < 0 || *price > 1_000_000) {
				return ModelMarketplaceConfig{}, infraerrors.BadRequest("MODEL_MARKETPLACE_INVALID_PRICE", fmt.Sprintf("model %s has an invalid price", model.ID))
			}
		}
		result.Models = append(result.Models, model)
	}
	return result, nil
}
