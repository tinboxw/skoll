package pluginmgr

import (
	"fmt"
	"strings"
	"time"
)

func (s *Service) UpgradePackageTransactional(transactionID, name, targetVersion, packageURL, packageHash, signature string, dependencies []Dependency, hooks []string, now time.Time) (UpgradeTransactionResult, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		transactionID = fmt.Sprintf("tx-%d", now.UTC().UnixNano())
	}
	name = strings.TrimSpace(name)
	targetVersion = strings.TrimSpace(targetVersion)
	packageURL = strings.TrimSpace(packageURL)
	packageHash = strings.TrimSpace(packageHash)
	signature = strings.TrimSpace(signature)

	checkpoints := make([]UpgradeCheckpoint, 0, 8)
	appendCheckpoint := func(step, status, detail string) {
		checkpoints = append(checkpoints, UpgradeCheckpoint{Step: step, Status: status, Detail: detail, At: now.UTC()})
	}

	appendCheckpoint("validate_input", "started", "")
	if name == "" || targetVersion == "" || packageURL == "" || packageHash == "" {
		appendCheckpoint("validate_input", "failed", "name,target_version,package_url,package_hash are required")
		result := UpgradeTransactionResult{Name: name, TransactionID: transactionID, TargetVersion: targetVersion, Checkpoints: checkpoints, Reason: "invalid input"}
		s.recordUpgradeProvenance(result, now)
		return result, nil
	}
	appendCheckpoint("validate_input", "succeeded", "")

	appendCheckpoint("load_current", "started", "")
	current, err := s.Get(name)
	if err != nil {
		appendCheckpoint("load_current", "failed", err.Error())
		result := UpgradeTransactionResult{Name: name, TransactionID: transactionID, TargetVersion: targetVersion, Checkpoints: checkpoints, Reason: err.Error()}
		s.recordUpgradeProvenance(result, now)
		return result, nil
	}
	appendCheckpoint("load_current", "succeeded", current.Version)

	appendCheckpoint("verify_signature", "started", "")
	if signature != "" && !verifySignature(packageHash, signature) {
		appendCheckpoint("verify_signature", "failed", "signature verification failed")
		result := UpgradeTransactionResult{Name: name, TransactionID: transactionID, PreviousVersion: current.Version, TargetVersion: targetVersion, Checkpoints: checkpoints, Reason: ErrPluginSignatureInvalid.Error()}
		s.recordUpgradeProvenance(result, now)
		return result, nil
	}
	appendCheckpoint("verify_signature", "succeeded", "")

	appendCheckpoint("dependency_precheck", "started", "")
	if err := s.precheckDependencies(dependencies); err != nil {
		appendCheckpoint("dependency_precheck", "failed", err.Error())
		result := UpgradeTransactionResult{Name: name, TransactionID: transactionID, PreviousVersion: current.Version, TargetVersion: targetVersion, Checkpoints: checkpoints, Reason: err.Error()}
		s.recordUpgradeProvenance(result, now)
		return result, nil
	}
	appendCheckpoint("dependency_precheck", "succeeded", "")

	appendCheckpoint("apply_upgrade", "started", "")
	_, err = s.InstallPackageVerified(name, targetVersion, packageURL, packageHash, signature, dependencies, hooks)
	if err != nil {
		s.mu.Lock()
		s.items[name] = current
		s.mu.Unlock()
		appendCheckpoint("apply_upgrade", "failed", err.Error())
		appendCheckpoint("rollback", "succeeded", current.Version)
		result := UpgradeTransactionResult{
			Name:            name,
			TransactionID:   transactionID,
			PreviousVersion: current.Version,
			TargetVersion:   targetVersion,
			Succeeded:       false,
			RolledBack:      true,
			Checkpoints:     checkpoints,
			Reason:          err.Error(),
		}
		s.recordUpgradeProvenance(result, now)
		return result, nil
	}
	appendCheckpoint("apply_upgrade", "succeeded", targetVersion)

	result := UpgradeTransactionResult{
		Name:            name,
		TransactionID:   transactionID,
		PreviousVersion: current.Version,
		TargetVersion:   targetVersion,
		Succeeded:       true,
		RolledBack:      false,
		Checkpoints:     checkpoints,
	}
	s.recordUpgradeProvenance(result, now)
	return result, nil
}

func (s *Service) ListUpgradeProvenance(limit int) []UpgradeProvenanceRecord {
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.provenance) == 0 {
		return nil
	}
	if limit > len(s.provenance) {
		limit = len(s.provenance)
	}
	out := make([]UpgradeProvenanceRecord, 0, limit)
	for i := len(s.provenance) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.provenance[i])
	}
	return out
}

func (s *Service) recordUpgradeProvenance(result UpgradeTransactionResult, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provenance = append(s.provenance, UpgradeProvenanceRecord{
		TransactionID:   result.TransactionID,
		Name:            result.Name,
		PreviousVersion: result.PreviousVersion,
		TargetVersion:   result.TargetVersion,
		Succeeded:       result.Succeeded,
		RolledBack:      result.RolledBack,
		Reason:          result.Reason,
		RecordedAt:      now.UTC(),
	})
}
