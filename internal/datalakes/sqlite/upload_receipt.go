// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sqlite

import "context"

// UploadReceipt records the upload session a scan's results went up in, so
// the caller can ask the upstream about that upload afterwards. SessionID
// stays empty when nothing was uploaded.
type UploadReceipt struct {
	SessionID string
}

type uploadReceiptCtxKey struct{}

// WithUploadReceipt asks the upload to fill in receipt.
func WithUploadReceipt(ctx context.Context, receipt *UploadReceipt) context.Context {
	if receipt == nil {
		return ctx
	}
	return context.WithValue(ctx, uploadReceiptCtxKey{}, receipt)
}

func uploadReceiptFromCtx(ctx context.Context) *UploadReceipt {
	if v, ok := ctx.Value(uploadReceiptCtxKey{}).(*UploadReceipt); ok {
		return v
	}
	return nil
}
