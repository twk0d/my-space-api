import { formatDuration, formatTrackLabel, progressPercent } from './spotify.service';

describe('spotify helpers', () => {
  it('formats durations and progress', () => {
    expect(formatDuration(185000)).toBe('3:05');
    expect(formatDuration(0)).toBe('');
    expect(progressPercent(500, 1000)).toBe(50);
    expect(progressPercent(1500, 1000)).toBe(100);
  });

  it('formats track labels', () => {
    expect(formatTrackLabel({ name: 'Song', artists: ['One', 'Two'] })).toBe('Song - One, Two');
    expect(formatTrackLabel({ name: 'Song', artists: [] })).toBe('Song');
  });
});
