# 🔍 CF Clean IP Scanner

ابزار پیدا کردن IP‌های تمیز Cloudflare برای Termux (Android ARM64)

<div dir="rtl">

## ویژگی‌ها

- تست دقیق هزاران IP از رنج‌های رسمی Cloudflare  
- **دو روش اسکن پیشرفته:**  
  - **Normal:** تست TCPing × 4 بار + تست سرعت دانلود  
  - **Xray:** اسکن از طریق هسته‌ی واقعی Xray با کانفیگ شخصی شما  
- **پشتیبانی از دو فرمت کانفیگ برای حالت Xray:**  
  - **JSON** (فایل `config/xray_config.json`): کانفیگ کامل Xray  
  - **URL** (فایل `config/xray_config.txt`): لینک مستقیم کانفیگ  
- پشتیبانی از پروتکل‌های **VLESS، VMess، Trojan و Shadowsocks**  
- محاسبه **نرخ Packet Loss** و **میانگین تأخیر** برای هر IP  
- قابلیت متوقف کردن اسکن در هر لحظه با **Ctrl+C** و نمایش نتایج یافت‌شده  
- نمایش **مدت زمان اسکن** در پایان  
- ذخیره خودکار نتایج در فایل `clean_ips.txt` و لیست ساده در `clean_ips_list.txt`  

---

## 📥 نصب

### روش اول: دانلود مستقیم (پیشنهادی)

یک دستور ساده در Termux:

```bash
pkg update && pkg upgrade && pkg install wget unzip && wget https://github.com/4n0nymou3/CF-Clean-IP-Scanner/releases/download/v2.2.0/cf-scanner-arm64-v2.2.0.zip && unzip cf-scanner-arm64.zip && chmod +x cf-scanner
```

سپس اجرا کنید:

```bash
./cf-scanner
```

مزایا:

- سریع (حدود ۳۰ ثانیه)
- بدون نیاز به نصب Go
- فایل آماده و کامپایل‌شده

---

روش دوم: Build از سورس

یک دستور ساده:

```bash
curl -sL https://raw.githubusercontent.com/4n0nymou3/CF-Clean-IP-Scanner/main/install.sh | bash
```

سپس از هرجا اجرا کنید:

```bash
cf-scanner
```

مزایا:

- ۱۰۰٪ سازگار با دستگاه شما
- Build مستقیم در Termux
- نصب خودکار در PATH
- همراه با دریافت خودکار هسته Xray

معایب:

- زمان بیشتر (تا ۱۰ دقیقه)
- نیاز به نصب Golang (به‌صورت خودکار انجام می‌گیرد)

---

▶️ استفاده

پس از اجرای دستور، منوی زیر نمایش داده می‌شود:

```
Select scan mode:
  [1] Normal scan (TCP ping + speed test)
  [2] Xray scan (uses Xray core with your config)
Enter 1 or 2:
```

- گزینه ۱: اسکن معمولی (مناسب برای استفاده سریع و بدون نیاز به تنظیمات اضافی)
- گزینه ۲: اسکن با هسته Xray (نیازمند تنظیم کانفیگ شخصی است — توضیحات زیر را ببینید)

پس از انتخاب، اسکن به‌طور خودکار آغاز می‌شود. در هر لحظه می‌توانید با Ctrl+C متوقف کنید.

---

⚙️ روند کار ابزار

حالت Normal (گزینه ۱)

مرحله ۱: تست Latency

- تمام IP‌های موجود در فایل config/ip_ranges.txt (پیش‌فرض حدود ۶۰۰۰ رنج مؤثر) تست می‌شوند.
- هر IP دقیقاً ۴ بار TCPing روی پورت ۴۴۳ انجام می‌شود.
- برای هر IP محاسبه می‌شود:
  نرخ Packet Loss (درصد موفقیت پینگ‌ها)
  میانگین تأخیر (میانگین زمان پاسخ پینگ‌های موفق)
- نتایج بر اساس کمترین Packet Loss و سپس کمترین تأخیر مرتب می‌شوند.

مرحله ۲: تست سرعت دانلود

- از بین بهترین IP‌های مرحله اول، ۱۰ IP اول تست دانلود می‌شوند.
- تست از سرور رسمی Cloudflare انجام می‌شود.
- به محض یافتن ۱۰ IP سالم، اسکن متوقف می‌شود.

حالت Xray (گزینه ۲)

مرحله ۱: تست Latency با Xray

- برای هر IP، یک کانفیگ موقت با جایگزینی آدرس IP در outbound اصلی ساخته می‌شود.
- هسته Xray با آن کانفیگ اجرا شده و از طریق SOCKS داخلی، درخواست به https://cp.cloudflare.com/generate_204 ارسال می‌شود.
- این فرآیند دقیقاً مانند عملکرد یک کلاینت واقعی (مثل v2rayNG) است.
- هر IP ۳ بار تست می‌شود و میانگین تأخیر و درصد موفقیت محاسبه می‌گردد.
- تست‌ها به صورت همزمان با ۸ کارگر انجام می‌شود تا سرعت اسکن افزایش یابد.

مرحله ۲: تست سرعت دانلود با Xray

- از بین بهترین IP‌های مرحله اول، ۱۰ IP اول انتخاب شده و سرعت دانلود واقعی از طریق همان فرآیند Xray اندازه‌گیری می‌شود.
- حجم تست دانلود حدود ۵۰ مگابایت است.

نتیجه نهایی در هر دو حالت

IP‌ها بر اساس بالاترین سرعت دانلود مرتب و نمایش داده می‌شوند. همچنین فایل‌های زیر ذخیره می‌گردند:

- clean_ips.txt – نتایج کامل با جزئیات
- clean_ips_list.txt – لیست ساده IP‌ها (۱۰ IP برتر و همه IPهای پاسخ‌دهنده)

---

🛑 توقف اسکن در هر لحظه

در هر لحظه‌ای که خواستید می‌توانید با فشار دادن Ctrl+C اسکن را متوقف کنید. ابزار تمام IP‌های سالم یافت‌شده تا آن لحظه را برای شما نمایش و ذخیره می‌کند.

---

📊 مثال خروجی

```
========================================
   CLOUDFLARE CLEAN IP SCANNER
   Find the fastest Cloudflare IPs
========================================
...:..::.::: Designed by: Anonymous :::.::..:...

Version: 2.2.0

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
...

Results saved to clean_ips.txt
Simple IP list saved to clean_ips_list.txt

========================================
      Scan completed successfully!
========================================

  Scan Duration : 00:03:28
```

در حالت Xray، خروجی مشابه است اما در عنوان مراحل عبارت (Xray mode) نمایش داده می‌شود.

---

📁 فایل‌های خروجی

روش اول (دانلود مستقیم):

- فایل نتایج: clean_ips.txt و clean_ips_list.txt در همان پوشه‌ای که اجرا کردید.

روش دوم (Build از سورس):

- برنامه:
```
~/CF-Clean-IP-Scanner/
```
- فایل نتایج:
```
~/CF-Clean-IP-Scanner/clean_ips.txt
~/CF-Clean-IP-Scanner/clean_ips_list.txt
```

مشاهده نتایج:

```bash
cat clean_ips.txt
```

---

⚙️ تنظیمات پیشرفته (حالت Xray)

برای استفاده از اسکن Xray، **یکی** از دو فایل زیر را ویرایش کنید:

### روش اول: فرمت URL — فایل `config/xray_config.txt`

این روش برای کسانی است که لینک کانفیگ (مثلاً از v2rayNG یا اشتراک‌ها) دارند.

فایل `config/xray_config.txt` را باز کرده و لینک کانفیگ خود را در آن قرار دهید:

```
vless://9b8928b1-5394-4433-bf94-6116fd5656b3@example.com:443?type=ws&security=tls&host=example.com&path=%2F&sni=example.com#MyConfig
```

پروتکل‌های پشتیبانی‌شده: `vless://` · `vmess://` · `trojan://` · `ss://`

خطوطی که با `#` شروع می‌شوند نادیده گرفته می‌شوند (توضیحات).

### روش دوم: فرمت JSON — فایل `config/xray_config.json`

این روش برای کسانی است که کانفیگ کامل JSON دارند (مثل فایل‌های export شده از v2rayNG).

فایل `config/xray_config.json` را با کانفیگ معتبر خود جایگزین کنید. نمونه:

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
            "address": "IP_PLACEHOLDER",
            "port": 443,
            "users": [
              { "id": "your-uuid-here", "encryption": "none", "flow": "xtls-rprx-vision" }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
          "serverName": "your-domain.com",
          "allowInsecure": false
        }
      }
    }
  ]
}
```

نکته: اگر هر دو فایل پر باشند، فایل `xray_config.txt` (فرمت URL) اولویت دارد.

---

❓ سوالات متداول

چرا IP پیدا نمی‌کند؟

در شرایط فیلترینگ بسیار شدید یا اینترنت ناپایدار، ممکن است IP سالم پیدا نشود. راه‌حل‌ها:

- در ساعات کم‌ترافیک (شب) دوباره امتحان کنید
- مطمئن شوید VPN روشن نیست (اسکن باید بدون VPN انجام شود)
- از حالت Xray با یک کانفیگ معتبر استفاده کنید (دقت بالاتری دارد)

ملاک انتخاب IP‌ها چیست؟

1. اول: کمترین Packet Loss (IP‌هایی که بیشترین پینگ‌ها را پاسخ داده‌اند)
2. دوم: کمترین تأخیر (میانگین زمان پاسخ)
3. سوم: بالاترین سرعت دانلود (در نتیجه نهایی)

چقدر طول می‌کشد؟

- حالت Normal (برای ۶۰۰۰ IP مؤثر):
  مرحله Latency: حدود ۲ تا ۳ دقیقه
  مرحله Speed: حدود ۱ تا ۲ دقیقه
- حالت Xray (با ۸ کارگر همزمان و ۳ بار تکرار):
  سرعت اسکن حدود ۸ تا ۱۰ IP در ثانیه (بسته به تأخیر شبکه)
  برای ۱۰۰۰۰ IP حدود ۲۰ دقیقه

تست سرعت صفر نشان می‌دهد؟

دستور زیر را در Termux اجرا کنید:

```bash
pkg install ca-certificates
```

در شرایط فیلترینگ شدید در ایران، ممکن است آدرس تست سرعت مستقیماً در دسترس نباشد. در این حالت از حالت Xray استفاده کنید.

---

🔧 عیب‌یابی

خطا: Permission denied

```bash
chmod +x cf-scanner
```

خطا: wget not found / unzip not found / curl not found

```bash
pkg install wget unzip curl
```

خطا: Xray binary not found

در روش دوم نصب (Build از سورس)، Xray به‌طور خودکار دانلود می‌شود. در روش اول باید فایل را جداگانه دریافت کنید یا از روش دوم استفاده کنید.

خطا: No Xray config found

هیچ‌کدام از فایل‌های کانفیگ پر نشده‌اند. یکی از این دو فایل را ویرایش کنید:
- برای URL: `config/xray_config.txt`
- برای JSON: `config/xray_config.json`

خطا: unsupported URL scheme

پروتکل URL شما پشتیبانی نمی‌شود. پروتکل‌های معتبر: `vless://`، `vmess://`، `trojan://`، `ss://`

خطا: no SOCKS inbound found

کانفیگ JSON شما فاقد inbound SOCKS است. کانفیگ خود را بر اساس نمونه اصلاح کنید.

برنامه کرش می‌کند

Termux را ریستارت کنید:

```bash
exit
```

سپس دوباره Termux را باز کنید.

---

🔄 به‌روزرسانی

روش اول (دانلود مستقیم):

```bash
rm -f cf-scanner cf-scanner-arm64.zip
wget https://github.com/4n0nymou3/CF-Clean-IP-Scanner/releases/latest/download/cf-scanner-arm64.zip
unzip cf-scanner-arm64.zip
chmod +x cf-scanner
```

روش دوم (Build از سورس):

```bash
cd ~/CF-Clean-IP-Scanner
git pull
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner
```

---

🗑️ حذف

روش اول:

```bash
rm -f cf-scanner cf-scanner-arm64.zip clean_ips.txt clean_ips_list.txt
```

روش دوم:

```bash
rm -rf ~/CF-Clean-IP-Scanner
rm /data/data/com.termux/files/usr/bin/cf-scanner
```

---

💡 نکات مهم

- تست را با VPN خاموش انجام دهید تا IP‌های واقعی ایران پیدا شوند.
- فایل clean_ips.txt را برای استفاده بعدی نگه دارید.
- اگر نتیجه خوب نگرفتید، در زمان دیگری دوباره تست کنید.
- در هر لحظه می‌توانید با Ctrl+C اسکن را متوقف کنید.
- حداقل ۵۰ مگابایت فضای خالی در Termux داشته باشید.
- برای حالت Xray فقط یکی از دو فایل کانفیگ را پر کنید (txt برای URL، json برای JSON کامل).
- اگر از رنج‌های بسیار وسیع استفاده می‌کنید، صبور باشید یا تعداد IP‌ها را محدود کنید.

---

مجوز

این پروژه تحت مجوز MIT منتشر شده است — استفاده آزاد.

---

سازنده

طراحی و توسعه توسط: **Anonymous**

---

حمایت از پروژه

اگر این ابزار برای شما مفید بود:

- یک Star ⭐ به repository بدهید.
- آن را با دوستانتان به اشتراک بگذارید.

</div>