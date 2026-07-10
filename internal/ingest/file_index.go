package ingest

import (
	"context"
	"fmt"
	"os"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/sessionjsonl"
)

type preparedSourceFile struct {
	record    sourceFileRecord
	id        int64
	knownHash string
}

type fileIndexResult struct {
	Indexed  int
	Skipped  int
	Failed   int
	Sessions int
	Warnings []string
}

type skipDecision struct {
	Skip      bool
	KnownHash string
}

func (r *indexRun) indexFile(ctx context.Context, source model.Source, path string, force bool) fileIndexResult {
	prepared, skipped, err := r.prepareFile(ctx, source.ID, path, force)
	if err != nil {
		return failedFileIndex(path, err)
	}
	if skipped {
		return fileIndexResult{Skipped: 1}
	}
	return r.indexPreparedFile(ctx, source, prepared)
}

func (r *indexRun) prepareFile(ctx context.Context, sourceID int64, path string, force bool) (preparedSourceFile, bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return preparedSourceFile{}, false, err
	}
	record := sourceFileRecord{
		SourceID:   sourceID,
		Path:       path,
		SizeBytes:  stat.Size(),
		ModifiedAt: stat.ModTime().UTC(),
	}
	existing, hasExisting := r.existingFiles[path]
	decision, err := r.skipDecision(ctx, record, existing, hasExisting, force)
	if err != nil {
		return preparedSourceFile{}, false, err
	}
	if decision.Skip {
		return preparedSourceFile{}, true, nil
	}
	record.Hash = decision.KnownHash
	record.Status = "scanning"
	sourceFileID, err := r.indexer.upsertSourceFile(ctx, record)
	if err != nil {
		return preparedSourceFile{}, false, err
	}
	return preparedSourceFile{record: record, id: sourceFileID, knownHash: decision.KnownHash}, false, nil
}

func (r *indexRun) skipDecision(ctx context.Context, record sourceFileRecord, existing existingFile, hasExisting, force bool) (skipDecision, error) {
	if !hasExisting || force {
		return skipDecision{}, nil
	}
	decision := metadataSkipDecision(record, existing)
	if !isSkippableCandidate(record, existing) {
		return decision, nil
	}
	if existing.ModifiedAt.Equal(record.ModifiedAt) {
		decision.Skip = true
		return decision, nil
	}
	return r.contentHashSkipDecision(ctx, record, existing)
}

func metadataSkipDecision(record sourceFileRecord, existing existingFile) skipDecision {
	if existing.ModifiedAt.Equal(record.ModifiedAt) && fileSizeMatches(record, existing) {
		return skipDecision{KnownHash: existing.ContentHash}
	}
	return skipDecision{}
}

func isSkippableCandidate(record sourceFileRecord, existing existingFile) bool {
	return fileSizeMatches(record, existing) && canSkipExistingFile(existing)
}

func (r *indexRun) contentHashSkipDecision(ctx context.Context, record sourceFileRecord, existing existingFile) (skipDecision, error) {
	hash, err := sessionjsonl.HashFile(record.Path)
	if err != nil {
		return skipDecision{}, err
	}
	decision := skipDecision{KnownHash: hash}
	if existing.ContentHash == "" || existing.ContentHash != hash {
		return decision, nil
	}
	record.Hash = hash
	record.Status = existing.ScanStatus
	record.Message = existing.Message
	if _, err := r.indexer.upsertSourceFile(ctx, record); err != nil {
		return skipDecision{}, err
	}
	return skipDecision{Skip: true, KnownHash: hash}, nil
}

func fileSizeMatches(record sourceFileRecord, existing existingFile) bool {
	return existing.SizeBytes == record.SizeBytes
}

func canSkipExistingFile(existing existingFile) bool {
	return existing.ParserVersion >= sourceFileParserVersion && existing.HasAuditRun && isCompleteScanStatus(existing.ScanStatus)
}

func isCompleteScanStatus(status string) bool {
	return status == "indexed" || status == "warning"
}

func (r *indexRun) indexPreparedFile(ctx context.Context, source model.Source, file preparedSourceFile) fileIndexResult {
	parsed, hash, err := r.parseFile(file.record.Path, source.ID, file.id, file.knownHash)
	if err != nil {
		hash = fallbackFileHash(file.record.Path, hash)
		return r.finishFailedFile(ctx, file, hash, err)
	}
	if err := r.indexer.replaceParsedSession(ctx, source, file.id, parsed, r.calculator); err != nil {
		return r.finishFailedFile(ctx, file, hash, err)
	}
	status, message := indexedFileStatus(parsed)
	if err := r.indexer.finishSourceFile(ctx, file.id, hash, status, message); err != nil {
		return failedFileIndex(file.record.Path, err)
	}
	return fileIndexResult{Indexed: 1, Sessions: 1, Warnings: parsed.Warnings}
}

func (r *indexRun) finishFailedFile(ctx context.Context, file preparedSourceFile, hash string, indexErr error) fileIndexResult {
	_ = r.indexer.finishSourceFile(ctx, file.id, hash, "error", indexErr.Error())
	return failedFileIndex(file.record.Path, indexErr)
}

func (r *indexRun) parseFile(path string, sourceID, sourceFileID int64, knownHash string) (model.ParsedSession, string, error) {
	if knownHash != "" {
		parsed, err := sessionjsonl.ParseFile(path, sourceID, sourceFileID)
		return parsed, knownHash, err
	}
	return sessionjsonl.ParseFileWithHash(path, sourceID, sourceFileID)
}

func failedFileIndex(path string, err error) fileIndexResult {
	return fileIndexResult{Failed: 1, Warnings: []string{fmt.Sprintf("%s: %v", path, err)}}
}

func fallbackFileHash(path, hash string) string {
	if hash != "" {
		return hash
	}
	fallback, err := sessionjsonl.HashFile(path)
	if err != nil {
		return ""
	}
	return fallback
}

func indexedFileStatus(parsed model.ParsedSession) (string, string) {
	if parsed.Session.ParseStatus == "warning" {
		return "warning", joinWarnings(parsed.Warnings)
	}
	return "indexed", ""
}
