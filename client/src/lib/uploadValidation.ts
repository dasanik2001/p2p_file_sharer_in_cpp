/** Max upload size enforced in the UI (must match product policy). */
export const MAX_UPLOAD_MB = 100;
export const MAX_UPLOAD_BYTES = MAX_UPLOAD_MB * 1024 * 1024;

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

/** Returns a user-facing error message, or null if the file is allowed. */
export function validateUploadFile(file: File): string | null {
  if (file.size > MAX_UPLOAD_BYTES) {
    return `File must be ${MAX_UPLOAD_MB} MB or smaller (selected: ${formatFileSize(file.size)}).`;
  }
  return null;
}
