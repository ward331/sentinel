#!/usr/bin/env python3
"""
SENTINEL Alert System — Send security/system alerts via Telegram + Gmail.
Usage:
  python3 sentinel-alerts.py --level critical --title "Security Alert" --body "Details here"
  python3 sentinel-alerts.py --level warning --title "SSH Login" --body "Login from 192.168.1.1"
  echo '{"level":"critical","title":"Test","body":"Test alert"}' | python3 sentinel-alerts.py --stdin
"""
import os
import sys
import json
import smtplib
import argparse
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from urllib.request import Request, urlopen
from urllib.parse import quote
from datetime import datetime

# ─── Config ───────────────────────────────────────────────────────

BOT_TOKEN = os.environ.get('TELEGRAM_BOT_TOKEN', '')
CHAT_ID = os.environ.get('TELEGRAM_CHAT_ID', '8177356632')
GMAIL_USER = 'edssynology@gmail.com'
GMAIL_APP_PASSWORD = ''
GMAIL_TO = 'edssynology@gmail.com'
SMTP_PASS_FILE = os.path.expanduser('~/.config/mit16410/smtp_pass.txt')

# Load app password
try:
    with open(SMTP_PASS_FILE) as f:
        for line in f:
            if line.startswith('SMTP_PASS='):
                GMAIL_APP_PASSWORD = line.strip().split('=', 1)[1]
                break
except Exception:
    pass

# Load bot token from systemd env or fallback
if not BOT_TOKEN:
    try:
        svc_path = os.path.expanduser('~/.config/systemd/user/sentinel-telegram-4086.service')
        with open(svc_path) as f:
            for line in f:
                line = line.strip()
                if line.startswith('Environment=TELEGRAM_BOT_TOKEN='):
                    BOT_TOKEN = line.split('=', 2)[2]
                    break
    except Exception:
        pass

# ─── Telegram Alert ──────────────────────────────────────────────

def send_telegram(title, body, level='info'):
    if not BOT_TOKEN:
        print('[Alert] No bot token — skipping Telegram', file=sys.stderr)
        return False

    emoji_map = {'critical': '🚨', 'warning': '⚠️', 'info': 'ℹ️'}
    emoji = emoji_map.get(level, 'ℹ️')

    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    text = f"{emoji} <b>SENTINEL SECURITY ALERT</b>\n\n<b>{title}</b>\n<i>{timestamp}</i>\n\n{body}"

    try:
        url = f'https://api.telegram.org/bot{BOT_TOKEN}/sendMessage'
        payload = json.dumps({
            'chat_id': CHAT_ID,
            'text': text,
            'parse_mode': 'HTML',
        }).encode('utf-8')
        req = Request(url, data=payload, headers={'Content-Type': 'application/json'})
        resp = urlopen(req, timeout=10)
        result = json.loads(resp.read())
        if result.get('ok'):
            print(f'[Alert] Telegram sent: {title}')
            return True
        else:
            print(f'[Alert] Telegram failed: {result}', file=sys.stderr)
            return False
    except Exception as e:
        print(f'[Alert] Telegram error: {e}', file=sys.stderr)
        return False

# ─── Gmail Alert ─────────────────────────────────────────────────

def send_gmail(title, body, level='info'):
    if not GMAIL_APP_PASSWORD:
        print('[Alert] No Gmail app password — skipping email', file=sys.stderr)
        return False

    emoji_map = {'critical': '🚨', 'warning': '⚠️', 'info': 'ℹ️'}
    emoji = emoji_map.get(level, 'ℹ️')
    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

    msg = MIMEMultipart('alternative')
    msg['Subject'] = f'{emoji} SENTINEL Alert: {title}'
    msg['From'] = f'SENTINEL AI <{GMAIL_USER}>'
    msg['To'] = GMAIL_TO

    # Plain text
    plain = f"SENTINEL SECURITY ALERT\n\n{title}\n{timestamp}\n\n{body}"

    # HTML
    level_color = {'critical': '#dc2626', 'warning': '#f59e0b', 'info': '#3b82f6'}.get(level, '#3b82f6')
    html = f"""<html><body style="font-family: monospace; background: #111; color: #e0e0e0; padding: 20px;">
<div style="border-left: 4px solid {level_color}; padding: 12px; background: #1a1a2e;">
<h2 style="color: {level_color}; margin: 0;">{emoji} SENTINEL SECURITY ALERT</h2>
<p style="color: #888; margin: 4px 0;">{timestamp}</p>
<h3 style="color: #fff; margin: 8px 0;">{title}</h3>
<pre style="color: #ccc; white-space: pre-wrap; margin: 8px 0;">{body}</pre>
</div>
<p style="color: #555; font-size: 11px; margin-top: 12px;">🛰 SENTINEL AI Security System</p>
</body></html>"""

    msg.attach(MIMEText(plain, 'plain'))
    msg.attach(MIMEText(html, 'html'))

    try:
        with smtplib.SMTP('smtp.gmail.com', 587, timeout=10) as server:
            server.starttls()
            server.login(GMAIL_USER, GMAIL_APP_PASSWORD)
            server.sendmail(GMAIL_USER, GMAIL_TO, msg.as_string())
        print(f'[Alert] Gmail sent to {GMAIL_TO}: {title}')
        return True
    except Exception as e:
        print(f'[Alert] Gmail error: {e}', file=sys.stderr)
        return False

# ─── Main ────────────────────────────────────────────────────────

def send_alert(title, body, level='info'):
    """Send alert via both Telegram and Gmail."""
    tg = send_telegram(title, body, level)
    gm = send_gmail(title, body, level)
    return tg or gm

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Send SENTINEL security alert')
    parser.add_argument('--level', choices=['critical', 'warning', 'info'], default='info')
    parser.add_argument('--title', default='Alert')
    parser.add_argument('--body', default='')
    parser.add_argument('--stdin', action='store_true', help='Read JSON from stdin')
    args = parser.parse_args()

    if args.stdin:
        data = json.loads(sys.stdin.read())
        send_alert(data.get('title', 'Alert'), data.get('body', ''), data.get('level', 'info'))
    else:
        send_alert(args.title, args.body, args.level)
