package store

import (
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestCanReviewAssetTransitionAllowsApproveFromRejected(t *testing.T) {
	if !canReviewAssetTransition(domain.AssetRejected, "approve") {
		t.Fatal("expected rejected asset to be selectable again")
	}
	if !canReviewAssetTransition(domain.AssetApproved, "reject") {
		t.Fatal("expected selected asset to remain rejectable")
	}
	if canReviewAssetTransition(domain.AssetPublished, "approve") {
		t.Fatal("published asset should not transition back to approve through review flow")
	}
}
