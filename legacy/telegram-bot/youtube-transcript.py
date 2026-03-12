#!/usr/bin/env python3
"""Fetch YouTube video transcript. Returns plain text transcript to stdout.
Usage: python3 youtube-transcript.py VIDEO_URL_OR_ID
"""
import sys
import re
import json

def extract_video_id(url_or_id):
    """Extract video ID from various YouTube URL formats."""
    # Already a bare ID
    if re.match(r'^[A-Za-z0-9_-]{11}$', url_or_id):
        return url_or_id
    # Standard URLs
    patterns = [
        r'(?:youtube\.com/watch\?.*v=|youtu\.be/|youtube\.com/embed/|youtube\.com/v/|youtube\.com/shorts/)([A-Za-z0-9_-]{11})',
    ]
    for pat in patterns:
        m = re.search(pat, url_or_id)
        if m:
            return m.group(1)
    return None

def get_transcript(video_id):
    """Try youtube-transcript-api first, fall back to yt-dlp."""
    # Method 1: youtube-transcript-api (fast, no download)
    try:
        from youtube_transcript_api import YouTubeTranscriptApi
        ytt_api = YouTubeTranscriptApi()
        transcript = ytt_api.fetch(video_id)
        lines = []
        for snippet in transcript:
            lines.append(snippet.text)
        if lines:
            return '\n'.join(lines)
    except Exception as e:
        err1 = str(e)

    # Method 2: yt-dlp subtitle extraction (handles more cases)
    try:
        import subprocess
        import tempfile
        import os
        with tempfile.TemporaryDirectory() as tmpdir:
            out_template = os.path.join(tmpdir, 'sub')
            result = subprocess.run([
                'yt-dlp', '--skip-download',
                '--write-auto-sub', '--write-sub',
                '--sub-lang', 'en',
                '--sub-format', 'vtt',
                '-o', out_template,
                f'https://www.youtube.com/watch?v={video_id}'
            ], capture_output=True, text=True, timeout=30)

            # Find the subtitle file
            for f in os.listdir(tmpdir):
                if f.endswith('.vtt'):
                    vtt_path = os.path.join(tmpdir, f)
                    with open(vtt_path, 'r') as fh:
                        vtt_text = fh.read()
                    # Strip VTT formatting
                    lines = []
                    for line in vtt_text.split('\n'):
                        line = line.strip()
                        if not line:
                            continue
                        if line.startswith('WEBVTT') or line.startswith('Kind:') or line.startswith('Language:'):
                            continue
                        if re.match(r'^\d{2}:\d{2}', line):
                            continue
                        if re.match(r'^\d+$', line):
                            continue
                        # Strip VTT tags
                        line = re.sub(r'<[^>]+>', '', line)
                        if line and line not in lines[-1:]:
                            lines.append(line)
                    if lines:
                        return '\n'.join(lines)
    except Exception as e:
        err2 = str(e)

    return None

def get_video_title(video_id):
    """Get video title via yt-dlp."""
    try:
        import subprocess
        result = subprocess.run([
            'yt-dlp', '--skip-download', '--print', 'title',
            f'https://www.youtube.com/watch?v={video_id}'
        ], capture_output=True, text=True, timeout=15)
        if result.returncode == 0 and result.stdout.strip():
            return result.stdout.strip()
    except:
        pass
    return None

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: youtube-transcript.py VIDEO_URL_OR_ID", file=sys.stderr)
        sys.exit(1)

    video_id = extract_video_id(sys.argv[1])
    if not video_id:
        print(f"Could not extract video ID from: {sys.argv[1]}", file=sys.stderr)
        sys.exit(1)

    title = get_video_title(video_id)
    transcript = get_transcript(video_id)

    if transcript:
        output = {
            'video_id': video_id,
            'title': title or 'Unknown',
            'transcript': transcript,
            'char_count': len(transcript),
        }
        print(json.dumps(output))
    else:
        print(json.dumps({'error': f'No transcript available for {video_id}', 'video_id': video_id}))
        sys.exit(1)
