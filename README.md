# 🔍 Clean IP Scanner

<div dir="rtl">

ابزار پیدا کردن IP‌های تمیز CDN‌های مختلف برای Termux (اندروید ARM64)

---

## ✨ ویژگی‌ها

- اسکن هزاران IP از رنج‌های CDN‌های مختلف — نه فقط Cloudflare
- **پشتیبانی از تمام CDN‌های بزرگ دنیا:**
  - **Cloudflare** — پیش‌فرض ابزار
  - **Akamai** — یکی از بزرگ‌ترین CDN‌های جهان
  - **Fastly** — CDN محبوب برای سرویس‌های بزرگ
  - **Amazon CloudFront** — CDN آمازون
  - **Microsoft Azure CDN**
  - **Google Cloud CDN**
  - **BunnyCDN**
  - **KeyCDN**
  - **Limelight Networks**
  - و هر CDN دیگری که رنج IP‌های آن در دسترس باشد
- **دو روش اسکن پیشرفته:**
  - **حالت Normal:** تست TCPing (۴ بار) + تست سرعت دانلود
  - **حالت Xray:** اسکن واقعی از طریق هسته‌ی Xray با کانفیگ شخصی شما
- **تعداد worker قابل تنظیم در حالت Xray:** عددی بین ۱ تا ۱۶ را بر اساس قدرت دستگاه خود انتخاب کنید
- **ذخیره و ادامه‌ی اسکن:** پیشرفت اسکن هر ۵۰۰ IP یک‌بار ذخیره می‌شود. اگر اگر اسکن به هر دلیلی یا با Ctrl+C توسط خود کاربر قطع شود، دفعه‌ی بعد می‌توانید از همان‌جا ادامه دهید
- **پشتیبانی از دو فرمت کانفیگ برای حالت Xray:**
  - **URL** (فایل `config/xray_config.txt`): لینک مستقیم کانفیگ
  - **JSON** (فایل `config/xray_config.json`): کانفیگ کامل Xray
- پشتیبانی از پروتکل‌های **VLESS، VMess، Trojan و Shadowsocks**
- محاسبه‌ی **نرخ Packet Loss** و **میانگین تأخیر** برای هر IP
- قابلیت توقف اسکن در هر لحظه با **Ctrl+C** و نمایش نتایج یافت‌شده تا آن لحظه
- نمایش **مدت زمان اسکن** در پایان
- ذخیره‌ی خودکار نتایج در فایل `clean_ips.txt` و لیست ساده در `clean_ips_list.txt`

---

## 📥 نصب

### روش اول: دانلود مستقیم فایل آماده (پیشنهادی برای اکثر کاربران)

این روش سریع‌ترین و ساده‌ترین روش است. فقط یک دستور در Termux وارد کنید:

```bash
pkg update && pkg upgrade -y && pkg install -y wget unzip && wget https://github.com/4n0nymou3/Clean-IP-Scanner/releases/latest/download/clean-ip-scanner-arm64.zip && unzip clean-ip-scanner-arm64.zip && chmod +x clean-ip-scanner
```

پس از اتمام، ابزار را اجرا کنید:

```bash
./clean-ip-scanner
```

**مزایا:**
- بسیار سریع (حدود ۳۰ ثانیه)
- بدون نیاز به نصب Go
- فایل آماده و کامپایل‌شده

**نکته:** فایل‌های کانفیگ و نتایج در همان پوشه‌ای که دستور را اجرا کردید ذخیره می‌شوند (معمولاً پوشه‌ی home یعنی `~`).

---

### روش دوم: ساخت از سورس کد

این روش برای کاربرانی است که می‌خواهند ابزار را مستقیماً از سورس کد در دستگاه خودشان بسازند. یک دستور ساده:

```bash
curl -sL https://raw.githubusercontent.com/4n0nymou3/Clean-IP-Scanner/main/install.sh | bash
```

پس از اتمام، از هرجایی در Termux اجرا کنید:

```bash
clean-ip-scanner
```

**مزایا:**
- ۱۰۰٪ سازگار با دستگاه شما
- ساخت مستقیم در Termux
- نصب خودکار در PATH (از هر پوشه‌ای قابل اجراست)
- دریافت خودکار هسته‌ی Xray

**معایب:**
- زمان بیشتر (تا ۱۰ دقیقه)
- نیاز به نصب Golang (به‌صورت خودکار انجام می‌گیرد)

**محل نصب و فایل‌ها:**
```
~/Clean-IP-Scanner/          ← پوشه‌ی اصلی برنامه
~/Clean-IP-Scanner/config/   ← پوشه‌ی فایل‌های کانفیگ
~/Clean-IP-Scanner/xray/     ← هسته‌ی Xray
```

---

## ▶️ اجرا و استفاده

### اجرای ابزار

**روش اول (دانلود مستقیم):**
```bash
./clean-ip-scanner
```

**روش دوم (ساخت از سورس):**
```bash
clean-ip-scanner
```

### انتخاب حالت اسکن

پس از اجرا، اگر اسکن ناتمامی وجود داشته باشد ابتدا این پیام نمایش داده می‌شود:

```
+-------------------------------------------------+
|         UNFINISHED SCAN DETECTED                |
+-------------------------------------------------+

  Previous scan mode : Xray (8 workers)
  Progress           : 1200 / 5956 IPs (20%)
  Responsive IPs     : 47 found so far
  Saved at           : 2025-01-15 22:30:14

Enter R to resume previous scan or N to start a new scan:
```

- **R** → ادامه‌ی اسکن قبلی از همان‌جایی که متوقف شده بود
- **N** → شروع اسکن جدید (اسکن قبلی پاک می‌شود)

اگر اسکن ناتمامی وجود نداشته باشد، مستقیماً منوی زیر نمایش داده می‌شود:

```
Select scan mode:
  [1] Normal scan (TCP ping + speed test)
  [2] Xray scan (uses Xray core with your config)
Enter 1 or 2:
```

- **گزینه ۱ — Normal:** اسکن معمولی. بدون نیاز به هیچ تنظیمی.
- **گزینه ۲ — Xray:** اسکن با هسته‌ی واقعی Xray. نیازمند تعریف کانفیگ شخصی است.

### انتخاب تعداد worker (فقط در حالت Xray)

بعد از انتخاب گزینه ۲، این منو نمایش داده می‌شود:

```
Select number of parallel workers for Xray scan:
  4  - Recommended for weak/older devices
  8  - Recommended for most devices (default)
  16 - Recommended for powerful devices
  Valid range: 1 to 16
Enter worker count (press Enter for default 8):
```

عدد دلخواه را وارد کنید یا Enter بزنید تا پیش‌فرض ۸ استفاده شود. اگر عددی خارج از بازه ۱ تا ۱۶ وارد کنید، خطا نمایش داده می‌شود.

---

## ⚙️ روند کار ابزار

### حالت Normal (گزینه ۱)

**مرحله ۱ — تست تأخیر (Latency):**
- تمام IP‌های موجود در فایل `config/ip_ranges.txt` تست می‌شوند
- هر IP دقیقاً ۴ بار از طریق TCP روی پورت ۴۴۳ پینگ می‌شود
- نرخ Packet Loss و میانگین تأخیر برای هر IP محاسبه می‌شود
- نتایج بر اساس کمترین Packet Loss و سپس کمترین تأخیر مرتب می‌شوند

**مرحله ۲ — تست سرعت دانلود:**
- از بین بهترین IP‌های مرحله‌ی اول، ۱۰ IP برتر تست دانلود می‌شوند
- تست از سرور رسمی Cloudflare انجام می‌شود (حجم تست: ۵۰ مگابایت)

### حالت Xray (گزینه ۲)

**مرحله ۱ — تست تأخیر با هسته‌ی Xray:**
- برای هر IP، یک کانفیگ موقت ساخته می‌شود که IP اسکن‌شده جایگزین آدرس سرور در کانفیگ شما می‌شود
- هسته‌ی Xray با آن کانفیگ اجرا می‌شود و از طریق SOCKS داخلی، یک درخواست واقعی ارسال می‌شود
- اسکن با تعداد worker‌هایی که شما انتخاب کردید (۱ تا ۱۶) به‌صورت همزمان انجام می‌شود

**مرحله ۲ — تست سرعت دانلود با هسته‌ی Xray:**
- از بین بهترین IP‌های مرحله‌ی اول، ۱۰ IP برتر انتخاب شده و سرعت دانلود واقعی از طریق همان فرآیند Xray اندازه‌گیری می‌شود

### ذخیره‌ی خودکار پیشرفت

در حالت TCP، پیشرفت اسکن هر ۲۰۰۰ IP یک‌بار و در حالت Xray، پیشرفت اسکن هر ۵۰۰ IP یک‌بار در فایل `scan_checkpoint.json` ذخیره می‌شود. اگر اسکن به هر دلیلی قطع شود (Ctrl+C، خاموش شدن دستگاه، بسته شدن Termux و...)، دفعه‌ی بعد که ابزار را اجرا کنید می‌توانید از همان‌جا ادامه دهید.

### نتیجه‌ی نهایی (هر دو حالت)

IP‌ها بر اساس بالاترین سرعت دانلود مرتب و نمایش داده می‌شوند. فایل‌های زیر نیز ذخیره می‌گردند:
- `clean_ips.txt` — نتایج کامل با جزئیات (تأخیر، packet loss، سرعت)
- `clean_ips_list.txt` — لیست ساده‌ی IP‌ها

---

## 🛑 توقف اسکن

در هر لحظه می‌توانید با فشار دادن **Ctrl+C** اسکن را متوقف کنید. ابزار تمام IP‌های سالم یافت‌شده تا آن لحظه را نمایش و ذخیره می‌کند. پیشرفت اسکن هم ذخیره می‌ماند تا دفعه‌ی بعد بتوانید ادامه دهید.

---

## 📊 نمونه خروجی

```
=================================================
              CLEAN IP SCANNER
          Find the fastest clean IPs
=================================================
...:..::.::: Designed by: Anonymous :::.::..:...

Version: 3.1.1

Optimized for Iran network conditions
Press Ctrl+C at any time to stop and see results found so far.

Select scan mode:
  [1] Normal scan (TCP ping + speed test)
  [2] Xray scan (uses Xray core with your config)
Enter 1 or 2: 2

Running Xray environment self-test...
Xray self-test passed successfully!

Select number of parallel workers for Xray scan:
  4  - Recommended for weak/older devices
  8  - Recommended for most devices (default)
  16 - Recommended for powerful devices
  Valid range: 1 to 16
Enter worker count (press Enter for default 8): 8

Start latency test (Xray mode - 8 workers, timeout 10s per IP)
5956 / 5956 [--↗--] Available: 312   8m14s

Latency test completed (Xray): 312 responsive IPs found

Start download speed test (Xray mode, Number: 10, Queue: 10)
10 / 10 [--↘--]   2m05s

Speed test completed (Xray): 10 clean IPs found

===========================================================================
                      CLEAN IPs FOUND
===========================================================================

Rank   IP Address           Sent   Received   Loss       Avg Delay      Download Speed
---------------------------------------------------------------------------
1.     188.114.97.163       1      1          0.00       198ms          1.47 MB/s
2.     190.93.246.213       1      1          0.00       214ms          1.21 MB/s
3.     190.93.244.169       1      1          0.00       231ms          1.05 MB/s

Results saved to clean_ips.txt
Simple IP list saved to clean_ips_list.txt

========================================
      Scan completed successfully!
========================================

  Scan Duration : 00:10:19
```

---

## 📋 تغییر لیست IP‌های اسکن (پشتیبانی از تمام CDN‌ها)

> **قابلیت مهم:** این ابزار محدود به Cloudflare نیست. با تغییر فایل `ip_ranges.txt` می‌توانید IP‌های هر CDN دیگری را اسکن کنید.

ابزار به‌صورت پیش‌فرض از رنج‌های رسمی **Cloudflare** استفاده می‌کند. اما شما می‌توانید رنج IP‌های هر CDN زیر را جایگزین کنید یا به لیست اضافه کنید:

| CDN | توضیح |
|-----|-------|
| **Cloudflare** | پیش‌فرض ابزار — رنج‌های رسمی از cloudflare.com/ips |
| **Akamai** | یکی از بزرگ‌ترین CDN‌های جهان |
| **Fastly** | CDN محبوب سرویس‌های بزرگ |
| **Amazon CloudFront** | CDN آمازون AWS |
| **Microsoft Azure CDN** | CDN مایکروسافت |
| **Google Cloud CDN** | CDN گوگل |
| **BunnyCDN** | CDN اروپایی با قیمت مناسب |
| **KeyCDN** | CDN سوئیسی |
| **Limelight Networks** | CDN برای استریمینگ |

> **نکته‌ی بسیار مهم برای حالت Xray:** اگر IP‌های یک CDN خاص را در `ip_ranges.txt` قرار دادید، کانفیگ Xray شما **حتماً باید روی همان CDN** تنظیم شده باشد. یعنی اگر IP‌های Akamai را اسکن می‌کنید، سرور پروکسی شما باید روی Akamai باشد — نه Cloudflare یا CDN دیگری. در غیر این صورت هیچ IP سالمی پیدا نخواهد شد.

### پیدا کردن فایل ip_ranges

```bash
# روش اول (دانلود مستقیم):
nano config/ip_ranges.txt

# روش دوم (ساخت از سورس):
nano ~/Clean-IP-Scanner/config/ip_ranges.txt
```

### فرمت فایل

هر خط می‌تواند شامل یک IP منفرد یا یک رنج CIDR باشد:

```
103.21.244.0/22
103.22.200.0/22
104.16.0.0/13
188.114.96.0/20
190.93.240.0/20
197.234.240.0/22
198.41.128.0/17
```

**نکته:** ابزار رنج‌های CIDR را به‌طور خودکار گسترش می‌دهد، IP‌های تکراری را حذف می‌کند، و قبل از شروع اسکن ترتیب تمام IP‌ها را به‌صورت کاملاً تصادفی درمی‌آورد.

---

## ⚙️ تنظیم کانفیگ برای حالت Xray (راهنمای کامل)

> **توضیح مهم:** در حالت Xray، شما یک کانفیگ شخصی تعریف می‌کنید. ابزار IP‌های مختلف را در آن کانفیگ قرار می‌دهد و تست می‌کند که کدام IP با کانفیگ شما کار می‌کند.

> **هم‌خوانی کانفیگ با CDN انتخابی:** کانفیگ Xray شما باید با CDN‌ای که IP‌های آن را در `ip_ranges.txt` قرار داده‌اید **هم‌خوانی داشته باشد**. برای مثال:
> - اگر رنج IP‌های **Cloudflare** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Cloudflare تنظیم شده باشد
> - اگر رنج IP‌های **Akamai** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Akamai تنظیم شده باشد
> - اگر رنج IP‌های **Fastly** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Fastly تنظیم شده باشد
>
> **CDN کانفیگ و CDN فایل ip_ranges.txt باید یکی باشند، در غیر این صورت هیچ IP سالمی پیدا نخواهد شد.**

برای استفاده از حالت Xray باید **یکی** از دو فایل زیر را ویرایش کنید.

---

### روش اول: فرمت URL (ساده‌تر — پیشنهادی)

```bash
nano config/xray_config.txt
```

لینک کانفیگ خود را در فایل قرار دهید:

```
vless://9b8928b1-5394-4433-bf94-6116fd5656b3@example.com:443?type=ws&security=tls&host=example.com&path=%2Fproxy&sni=example.com#MyConfig
```

> خطوطی که با `#` شروع می‌شوند به‌عنوان توضیح نادیده گرفته می‌شوند.

> **یادآوری:** مطمئن شوید این کانفیگ متعلق به همان CDN‌ای است که IP‌های آن را در `ip_ranges.txt` قرار داده‌اید.

**ذخیره و خروج از nano:** `Ctrl+O` → Enter → `Ctrl+X`

**پروتکل‌های پشتیبانی‌شده:** `vless://` · `vmess://` · `trojan://` · `ss://`

---

### روش دوم: فرمت JSON

```bash
nano config/xray_config.json
```

**نمونه کانفیگ JSON:**

```json
{
  "log": { "loglevel": "warning" },
  "inbounds": [
    {
      "port": 1080,
      "protocol": "socks",
      "settings": { "udp": false },
      "listen": "127.0.0.1"
    }
  ],
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "your-server-ip-or-domain",
            "port": 443,
            "users": [
              {
                "id": "your-uuid-here",
                "encryption": "none"
              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "ws",
        "security": "tls",
        "tlsSettings": {
          "serverName": "your-domain.com",
          "allowInsecure": false
        },
        "wsSettings": {
          "path": "/proxy",
          "headers": { "Host": "your-domain.com" }
        }
      }
    }
  ]
}
```

> **نکته:** اگر هر دو فایل پر باشند، فایل `xray_config.txt` (فرمت URL) اولویت دارد.

---

## 📁 فایل‌های مهم

### ساختار پوشه‌ها

**روش اول (دانلود مستقیم):**
```
./clean-ip-scanner            ← فایل اجرایی ابزار
./config/
    ip_ranges.txt             ← لیست رنج‌های IP برای اسکن (قابل تغییر برای هر CDN)
    xray_config.txt           ← کانفیگ Xray (فرمت URL)
    xray_config.json          ← کانفیگ Xray (فرمت JSON)
./xray/
    xray                      ← هسته‌ی Xray
./clean_ips.txt               ← نتایج کامل آخرین اسکن
./clean_ips_list.txt          ← لیست ساده‌ی IP‌ها
./scan_checkpoint.json        ← فایل ذخیره‌ی پیشرفت (خودکار)
```

**روش دوم (ساخت از سورس):**
```
~/Clean-IP-Scanner/
    clean-ip-scanner
    config/
        ip_ranges.txt
        xray_config.txt
        xray_config.json
    xray/
        xray
    clean_ips.txt
    clean_ips_list.txt
    scan_checkpoint.json
```

---

## 🔄 به‌روزرسانی

### روش اول (دانلود مستقیم):

```bash
rm -f clean-ip-scanner clean-ip-scanner-arm64.zip
wget https://github.com/4n0nymou3/Clean-IP-Scanner/releases/latest/download/clean-ip-scanner-arm64.zip
unzip clean-ip-scanner-arm64.zip
chmod +x clean-ip-scanner
```

> **نکته:** به‌روزرسانی فایل‌های `config/` را تغییر نمی‌دهد. کانفیگ‌های شما دست نخورده می‌مانند.

### روش دوم (ساخت از سورس):

```bash
cd ~/Clean-IP-Scanner
git pull
CGO_ENABLED=0 go build -ldflags="-s -w" -o clean-ip-scanner
```

---

## 🗑️ حذف ابزار

### روش اول (دانلود مستقیم):

```bash
rm -f clean-ip-scanner clean-ip-scanner-arm64.zip clean_ips.txt clean_ips_list.txt scan_checkpoint.json
rm -rf config/ xray/
```

### روش دوم (ساخت از سورس):

```bash
rm -rf ~/Clean-IP-Scanner
rm -f /data/data/com.termux/files/usr/bin/clean-ip-scanner
```

---

## ❓ سوالات متداول

**آیا این ابزار فقط برای Cloudflare است؟**

خیر. ابزار به‌صورت پیش‌فرض رنج IP‌های Cloudflare را اسکن می‌کند، اما با تغییر فایل `config/ip_ranges.txt` می‌توانید IP‌های هر CDN دیگری مانند Akamai، Fastly، Amazon CloudFront، Microsoft Azure CDN، Google Cloud CDN و غیره را اسکن کنید.

**تعداد worker چه عددی وارد کنم؟**

- دستگاه‌های قدیمی یا ضعیف: ۴
- اکثر دستگاه‌های معمولی: ۸ (پیش‌فرض)
- دستگاه‌های قوی: ۱۶
اگر مطمئن نیستید، Enter بزنید تا عدد پیش‌فرض ۸ استفاده شود.

**اگر اسکن قطع شود چه اتفاقی می‌افتد؟**

در حالت TCP، پیشرفت اسکن هر ۲۰۰۰ IP یک‌بار و در حالت Xray، پیشرفت اسکن هر ۵۰۰ IP یک‌بار در فایل `scan_checkpoint.json` ذخیره می‌شود. اگر اسکن به هر دلیلی قطع شود (Ctrl+C، خاموش شدن دستگاه، بسته شدن Termux و...)، دفعه‌ی بعد که ابزار را اجرا کنید، اسکن ناتمام را تشخیص داده و می‌پرسد آیا می‌خواهید ادامه دهید یا از نو شروع کنید.

**چرا IP پیدا نمی‌کند؟**

- در ساعات کم‌ترافیک دوباره امتحان کنید
- مطمئن شوید هیچ VPN فعالی ندارید
- از حالت Xray با یک کانفیگ معتبر استفاده کنید
- در حالت Xray، مطمئن شوید کانفیگ شما متعلق به همان CDN‌ای است که IP‌های آن را اسکن می‌کنید

**چقدر طول می‌کشد؟**

- **حالت Normal:** مرحله‌ی تأخیر ۲ تا ۳ دقیقه، مرحله‌ی سرعت ۱ تا ۲ دقیقه
- **حالت Xray:** بستگی به تعداد worker و سرعت دستگاه دارد. با ۸ worker حدود ۸ تا ۱۵ دقیقه

---

## 🔧 عیب‌یابی

**خطا: `Permission denied`**
```bash
chmod +x clean-ip-scanner
```

**خطا: `wget not found` یا `curl not found`**
```bash
pkg install wget curl unzip
```

**خطا: `Xray binary not found`**

دوباره از مراحل نصب شروع کنید.

**خطا: `No Xray config found`**

یکی از این دو فایل را ویرایش کنید:
- `config/xray_config.txt` برای فرمت URL
- `config/xray_config.json` برای فرمت JSON

**خطا: `unsupported protocol`**

پروتکل‌های معتبر: `vless://`، `vmess://`، `trojan://`، `ss://`

**خطا: `no SOCKS inbound found`**

کانفیگ JSON شما فاقد بخش inbound از نوع SOCKS است. بر اساس نمونه‌ی بالا اصلاح کنید.

---

## 💡 نکات مهم

- اسکن را **بدون VPN فعال** انجام دهید
- **کانفیگ Xray شما باید با CDN مورد نظر هم‌خوانی داشته باشد**
- در حالت Xray فقط **یکی** از دو فایل کانفیگ را پر کنید (فایل txt اولویت دارد)
- در هر لحظه با **Ctrl+C** می‌توانید اسکن را متوقف کنید — پیشرفت ذخیره می‌ماند
- فایل `scan_checkpoint.json` را دستی پاک نکنید مگر اینکه بخواهید اسکن قبلی را کامل فراموش کنید

---

## 📜 مجوز

این پروژه تحت مجوز MIT منتشر شده است — استفاده آزاد.

---

## 👤 سازنده

طراحی و توسعه توسط: **Anonymous**

---

## ⭐ حمایت از پروژه

اگر این ابزار برای شما مفید بود:
- یک **Star ⭐** به repository بدهید
- آن را با دوستانتان به اشتراک بگذارید

</div>