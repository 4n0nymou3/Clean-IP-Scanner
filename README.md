# 🔍 Clean IP Scanner

<div dir="rtl">

ابزار پیدا کردن IP‌های تمیز CDN‌های مختلف برای Termux (اندروید ARM64)

---

## ✨ ویژگی‌ها

- اسکن هزاران IP از رنج‌های CDN‌های مختلف
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

پس از اجرا، منوی زیر نمایش داده می‌شود:

```
Select scan mode:
  [1] Normal scan (TCP ping + speed test)
  [2] Xray scan (uses Xray core with your config)
Enter 1 or 2:
```

عدد `1` یا `2` را تایپ کرده و Enter بزنید.

- **گزینه ۱ — Normal:** اسکن معمولی. بدون نیاز به هیچ تنظیمی. مناسب برای پیدا کردن سریع IP‌های سالم از هر CDN.
- **گزینه ۲ — Xray:** اسکن با هسته‌ی واقعی Xray. نیازمند تعریف کانفیگ شخصی است. IP‌هایی که پیدا می‌کند ۱۰۰٪ با کانفیگ شما سازگار هستند.

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
- این فرآیند دقیقاً مانند عملکرد یک کلاینت واقعی مبتنی بر Xray است
- اسکن با ۸ کارگر (worker) همزمان انجام می‌شود

**مرحله ۲ — تست سرعت دانلود با هسته‌ی Xray:**
- از بین بهترین IP‌های مرحله‌ی اول، ۱۰ IP برتر انتخاب شده و سرعت دانلود واقعی از طریق همان فرآیند Xray اندازه‌گیری می‌شود

### نتیجه‌ی نهایی (هر دو حالت)

IP‌ها بر اساس بالاترین سرعت دانلود مرتب و نمایش داده می‌شوند. فایل‌های زیر نیز ذخیره می‌گردند:
- `clean_ips.txt` — نتایج کامل با جزئیات (تأخیر، packet loss، سرعت)
- `clean_ips_list.txt` — لیست ساده‌ی IP‌ها

---

## 🛑 توقف اسکن

در هر لحظه می‌توانید با فشار دادن **Ctrl+C** اسکن را متوقف کنید. ابزار تمام IP‌های سالم یافت‌شده تا آن لحظه را نمایش و ذخیره می‌کند.

---

## 📊 نمونه خروجی

```
=================================================
              CLEAN IP SCANNER
          Find the fastest clean IPs
=================================================
...:..::.::: Designed by: Anonymous :::.::..:...

Version: 3.0.0

Optimized for Iran network conditions
Press Ctrl+C at any time to stop and see results found so far.

Select scan mode:
  [1] Normal scan (TCP ping + speed test)
  [2] Xray scan (uses Xray core with your config)
Enter 1 or 2: 1

Start latency test (Mode: TCP, Port: 443, Range: 0 ~ 9999 ms, Packet Loss: 1.00)
5956 / 5956 [--↗--] Available: 2184   2m3s

Latency test completed: 2184 responsive IPs found

Start download speed test (Minimum speed: 0.00 MB/s, Number: 10, Queue: 10)
10 / 10 [--↘--]   1m22s

Speed test completed: 10 clean IPs found

===========================================================================
                      CLEAN IPs FOUND
===========================================================================

Rank   IP Address           Sent   Received   Loss       Avg Delay      Download Speed
---------------------------------------------------------------------------
1.     188.114.97.163       4      4          0.00       241ms          1.32 MB/s
2.     190.93.246.213       4      4          0.00       212ms          1.05 MB/s
3.     190.93.244.169       4      4          0.00       201ms          1.04 MB/s

Results saved to clean_ips.txt
Simple IP list saved to clean_ips_list.txt

========================================
      Scan completed successfully!
========================================

  Scan Duration : 00:03:28
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

### پاک کردن لیست و جایگزینی با لیست جدید

**در nano:**
1. فایل را باز کنید: `nano config/ip_ranges.txt`
2. برای پاک کردن همه: `Ctrl+K` را پشت سر هم فشار دهید تا فایل خالی شود
3. رنج‌های جدید را تایپ یا paste کنید (هر رنج در یک خط)
4. ذخیره: `Ctrl+O` → Enter
5. خروج: `Ctrl+X`

**نکته:** ابزار رنج‌های CIDR را به‌طور خودکار گسترش می‌دهد و IP‌های تکراری را حذف می‌کند.

---

## ⚙️ تنظیم کانفیگ برای حالت Xray (راهنمای کامل)

> **توضیح مهم:** در حالت Xray، شما یک کانفیگ شخصی تعریف می‌کنید. ابزار IP‌های مختلف را در آن کانفیگ قرار می‌دهد و تست می‌کند که کدام IP با کانفیگ شما کار می‌کند. بنابراین کانفیگی که تعریف می‌کنید باید از نظر UUID، سرور، و تنظیمات معتبر باشد — تنها چیزی که ممکن است مسدود شده باشد IP آن است.

> **هم‌خوانی کانفیگ با CDN انتخابی:** کانفیگ Xray شما باید با CDN‌ای که IP‌های آن را در `ip_ranges.txt` قرار داده‌اید **هم‌خوانی داشته باشد**. برای مثال:
> - اگر رنج IP‌های **Cloudflare** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Cloudflare تنظیم شده باشد
> - اگر رنج IP‌های **Akamai** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Akamai تنظیم شده باشد
> - اگر رنج IP‌های **Fastly** را اسکن می‌کنید ← کانفیگ شما باید روی سرور Fastly تنظیم شده باشد
>
> **CDN کانفیگ و CDN فایل ip_ranges.txt باید یکی باشند، در غیر این صورت هیچ IP سالمی پیدا نخواهد شد.**

برای استفاده از حالت Xray باید **یکی** از دو فایل زیر را ویرایش کنید.

---

### روش اول: فرمت URL (ساده‌تر — پیشنهادی)

**این روش برای کسانی است که لینک کانفیگ (مثلاً از اشتراک یا دوستان) دارند.**

#### پیدا کردن فایل کانفیگ

ابتدا باید به پوشه‌ی `config` بروید:

**روش اول (دانلود مستقیم):**
```bash
ls
cd config
ls
```

**روش دوم (ساخت از سورس):**
```bash
cd ~/Clean-IP-Scanner/config
ls
```

#### ویرایش فایل با nano

```bash
nano xray_config.txt
```

پنجره‌ی ویرایشگر nano باز می‌شود. محتوای پیش‌فرض فایل را می‌بینید.

**پاک کردن کل محتوا و نوشتن کانفیگ جدید:**

1. کلیدهای `Ctrl+K` را چند بار فشار دهید تا تمام خطوط پاک شوند
2. یا از میانبر `Ctrl+A` (انتخاب همه) و سپس `Delete` استفاده کنید

**نوشتن کانفیگ:**

لینک کانفیگ خود را در فایل قرار دهید. مثال:

```
vless://9b8928b1-5394-4433-bf94-6116fd5656b3@example.com:443?type=ws&security=tls&host=example.com&path=%2Fproxy&sni=example.com#MyConfig
```

> خطوطی که با `#` شروع می‌شوند به‌عنوان توضیح نادیده گرفته می‌شوند.

> **یادآوری:** مطمئن شوید این کانفیگ متعلق به همان CDN‌ای است که IP‌های آن را در `ip_ranges.txt` قرار داده‌اید.

**ذخیره و خروج از nano:**
- `Ctrl+O` → فشار Enter (ذخیره)
- `Ctrl+X` (خروج)

**پروتکل‌های پشتیبانی‌شده:** `vless://` · `vmess://` · `trojan://` · `ss://`

---

### روش دوم: فرمت JSON

**این روش برای کسانی است که فایل کانفیگ کامل JSON دارند.**

#### ویرایش فایل JSON

```bash
# روش اول (دانلود مستقیم):
nano config/xray_config.json

# روش دوم (ساخت از سورس):
nano ~/Clean-IP-Scanner/config/xray_config.json
```

**پاک کردن کل محتوا:**

در nano، برای پاک کردن سریع تمام محتوا:
1. `Ctrl+K` را نگه دارید تا تمام خطوط پاک شوند
2. سپس کانفیگ JSON خود را paste کنید

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
          "headers": {
            "Host": "your-domain.com"
          }
        }
      }
    }
  ]
}
```

**ذخیره و خروج:** `Ctrl+O` → Enter → `Ctrl+X`

> **نکته:** اگر هر دو فایل پر باشند، فایل `xray_config.txt` (فرمت URL) اولویت دارد.

> **یادآوری:** مطمئن شوید این کانفیگ JSON متعلق به همان CDN‌ای است که IP‌های آن را در `ip_ranges.txt` قرار داده‌اید.

---

## 📁 فایل‌های مهم

### ساختار پوشه‌ها

**روش اول (دانلود مستقیم) — از پوشه‌ای که دانلود کردید:**
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
```

**روش دوم (ساخت از سورس):**
```
~/Clean-IP-Scanner/
    clean-ip-scanner          ← فایل اجرایی
    config/
        ip_ranges.txt
        xray_config.txt
        xray_config.json
    xray/
        xray
    clean_ips.txt
    clean_ips_list.txt
```

### مشاهده‌ی نتایج

```bash
# نتایج کامل:
cat clean_ips.txt

# فقط لیست IP‌ها:
cat clean_ips_list.txt
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
rm -f clean-ip-scanner clean-ip-scanner-arm64.zip clean_ips.txt clean_ips_list.txt
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

خیر. ابزار به‌صورت پیش‌فرض رنج IP‌های Cloudflare را اسکن می‌کند، اما با تغییر فایل `config/ip_ranges.txt` می‌توانید IP‌های هر CDN دیگری مانند Akamai، Fastly، Amazon CloudFront، Microsoft Azure CDN، Google Cloud CDN و غیره را اسکن کنید. تنها کافی است رنج‌های CIDR آن CDN را در فایل قرار دهید.

**چرا IP پیدا نمی‌کند؟**

در شرایط فیلترینگ شدید، این کاملاً طبیعی است. راه‌حل‌ها:
- در ساعات کم‌ترافیک (مثلاً شب) دوباره امتحان کنید
- مطمئن شوید هیچ VPN فعالی ندارید (اسکن باید بدون VPN انجام شود)
- از حالت Xray با یک کانفیگ معتبر استفاده کنید
- در حالت Xray، مطمئن شوید کانفیگ شما متعلق به همان CDN‌ای است که IP‌های آن را اسکن می‌کنید

**ملاک انتخاب IP‌ها چیست؟**

1. کمترین Packet Loss (IP‌هایی که بیشترین پینگ‌ها را پاسخ داده‌اند)
2. کمترین تأخیر (میانگین زمان پاسخ)
3. بالاترین سرعت دانلود (در نتیجه‌ی نهایی)

**چقدر طول می‌کشد؟**

- **حالت Normal:**
  - مرحله‌ی تأخیر: ۲ تا ۳ دقیقه
  - مرحله‌ی سرعت: ۱ تا ۲ دقیقه
- **حالت Xray** (با ۸ worker همزمان):
  - سرعت اسکن به میزان قابل توجهی بالاتر از نسخه‌های قبلی است
  - می‌توانید هر زمان با Ctrl+C متوقف کنید

**تست سرعت صفر نشان می‌دهد؟**

```bash
pkg install ca-certificates
```

در شرایط فیلترینگ شدید، ممکن است آدرس تست سرعت مستقیماً در دسترس نباشد. در این حالت از حالت Xray استفاده کنید.

---

## 🔧 عیب‌یابی

**خطا: `Permission denied`**
```bash
chmod +x clean-ip-scanner
```

**خطا: `wget not found` یا `unzip not found` یا `curl not found`**
```bash
pkg install wget unzip curl
```

**خطا: `Xray binary not found`**

در روش دوم نصب، Xray به‌طور خودکار دانلود می‌شود. در روش اول، فایل zip باید شامل Xray باشد. دوباره از مراحل نصب شروع کنید.

**خطا: `No Xray config found`**

هیچ‌کدام از فایل‌های کانفیگ پر نشده‌اند. یکی از این دو فایل را ویرایش کنید:
- برای URL: `config/xray_config.txt`
- برای JSON: `config/xray_config.json`

**خطا: `unsupported protocol`**

پروتکل URL شما پشتیبانی نمی‌شود. پروتکل‌های معتبر: `vless://`، `vmess://`، `trojan://`، `ss://`

**خطا: `no SOCKS inbound found`**

کانفیگ JSON شما فاقد بخش inbound از نوع SOCKS است. کانفیگ خود را بر اساس نمونه‌ی بالا اصلاح کنید.

**ابزار کرش می‌کند یا پاسخ نمی‌دهد**

Termux را ببندید و دوباره باز کنید:
```bash
exit
```
سپس Termux را دوباره اجرا کنید و ابزار را مجدداً اجرا کنید.

---

## 💡 نکات مهم

- اسکن را **بدون VPN فعال** انجام دهید تا نتایج دقیق‌تری بگیرید
- فایل `clean_ips.txt` را برای استفاده‌ی بعدی نگه دارید
- در حالت Xray فقط **یکی** از دو فایل کانفیگ را پر کنید (اگر هر دو پر باشند، فایل txt اولویت دارد)
- **کانفیگ Xray شما باید با CDN مورد نظر هم‌خوانی داشته باشد** — اگر IP‌های Akamai را اسکن می‌کنید، کانفیگتان باید روی سرور Akamai باشد
- حداقل ۵۰ مگابایت فضای خالی در Termux داشته باشید
- اگر نتیجه‌ی خوبی نگرفتید، در زمان دیگری دوباره امتحان کنید — شرایط شبکه متغیر است
- در هر لحظه با **Ctrl+C** می‌توانید اسکن را متوقف کنید و نتایج تا آن لحظه ذخیره می‌شوند

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