'use client';

import { useState, useCallback } from 'react';
import { useDropzone, type FileRejection } from 'react-dropzone';
import { FiUpload } from 'react-icons/fi';
import {
  MAX_UPLOAD_BYTES,
  MAX_UPLOAD_MB,
  validateUploadFile,
} from '@/lib/uploadValidation';

interface FileUploadProps {
  onFileUpload: (file: File) => void;
  isUploading: boolean;
}

function rejectionMessage(rejections: FileRejection[]): string {
  const first = rejections[0]?.errors[0];
  if (first?.message) return first.message;
  return 'File not accepted.';
}

export default function FileUpload({ onFileUpload, isUploading }: FileUploadProps) {
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      if (acceptedFiles.length === 0) return;

      const file = acceptedFiles[0];
      const validationError = validateUploadFile(file);
      if (validationError) {
        setError(validationError);
        return;
      }

      setError(null);
      onFileUpload(file);
    },
    [onFileUpload]
  );

  const onDropRejected = useCallback((rejections: FileRejection[]) => {
    setDragActive(false);
    setError(rejectionMessage(rejections));
  }, []);

  const { getRootProps, getInputProps } = useDropzone({
    onDrop,
    multiple: false,
    maxSize: MAX_UPLOAD_BYTES,
    validator: (file) => {
      const message = validateUploadFile(file);
      if (message) {
        return { code: 'invalid-file', message };
      }
      return null;
    },
    onDragEnter: () => setDragActive(true),
    onDragLeave: () => setDragActive(false),
    onDropAccepted: () => setDragActive(false),
    onDropRejected,
  });

  return (
    <div className="space-y-2">
      <div
        {...getRootProps()}
        className={`
        w-full p-8 border-2 border-dashed rounded-lg text-center cursor-pointer transition-all
        ${dragActive
            ? 'border-blue-500 bg-blue-50'
            : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'
          }
        ${isUploading ? 'opacity-50 pointer-events-none' : ''}
      `}
      >
        <input {...getInputProps()} />
        <div className="flex flex-col items-center justify-center space-y-3">
          <div className="p-3 bg-blue-100 rounded-full">
            <FiUpload className="w-6 h-6 text-blue-500" />
          </div>
          <p className="text-lg font-medium">Drag & drop a file here, or click to select</p>
          <p className="text-sm text-gray-500">
            Max {MAX_UPLOAD_MB} MB · all file types supported
          </p>
        </div>
      </div>
      {error && <p className="text-sm text-red-600">{error}</p>}
    </div>
  );
}
