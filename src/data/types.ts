export interface Currently {
  listening: string;
  isOnline: boolean;
  track?: MusicTrack;
}

export enum SignatureStatus {
  Pending = 'pending',
  Approved = 'approved',
  Rejected = 'rejected',
}

export interface Signature {
  id: string;
  name: string;
  message: string;
  status: SignatureStatus;
  createdAt: Date;
  approvedAt?: Date;
  rejectedAt?: Date;
}

export interface MusicTrack {
  name: string;
  artists: string[];
  album?: string;
  durationMs?: number;
  durationLabel?: string;
  progressMs?: number;
  progressLabel?: string;
  progressPercent?: number;
  imageUrl?: string;
  externalUrl?: string;
  isPlaying?: boolean;
}

export interface Database {
  currently: Currently;
  signatures: Signature[];
  visitorCount: number;
}
