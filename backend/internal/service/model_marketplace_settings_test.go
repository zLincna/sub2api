//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelMarketplaceConfig_DefaultOrderKeepsLatestOpenAIModelsFirst(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, nil)

	config, err := svc.GetModelMarketplaceConfig(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(config.Models), 3)
	require.Equal(t, "gpt-5.6-sol", config.Models[0].ID)
	require.Equal(t, "gpt-5.6-terra", config.Models[1].ID)
	require.Equal(t, "gpt-5.6-luna", config.Models[2].ID)
}

func TestModelMarketplaceConfig_UpdateRoundTripPreservesOrder(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, nil)
	input := ModelMarketplaceConfig{Models: []ModelMarketplaceModel{
		{ID: " model-new ", Platform: " OpenAI ", BillingMode: "token", Enabled: true},
		{ID: "model-old", Platform: "openai", BillingMode: "token", Enabled: true},
	}}

	saved, err := svc.UpdateModelMarketplaceConfig(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "model-new", saved.Models[0].ID)
	require.Equal(t, "openai", saved.Models[0].Platform)

	loaded, err := svc.GetModelMarketplaceConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"model-new", "model-old"}, []string{loaded.Models[0].ID, loaded.Models[1].ID})
}

func TestModelMarketplaceConfig_RejectsDuplicateModel(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, nil)
	_, err := svc.UpdateModelMarketplaceConfig(context.Background(), ModelMarketplaceConfig{Models: []ModelMarketplaceModel{
		{ID: "gpt-test", Platform: "openai", BillingMode: "token", Enabled: true},
		{ID: "GPT-TEST", Platform: "openai", BillingMode: "token", Enabled: true},
	}})
	require.Error(t, err)
}
