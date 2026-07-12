

# Xray CLI Dashboard 🚀

![Go Version](https://img.shields.io/badge/Go-1.20%2B-blue.svg)
![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

[ 🇮🇷 نسخه فارسی در پایین صفحه (Persian version below) ]

A fast, interactive, and powerful Terminal User Interface (TUI) client for Xray-core. Manage your VLESS subscriptions, test server latencies in the background, and connect seamlessly using Manual, System Proxy (GNOME/KDE), or TUN modes directly from your terminal.


## ✨ Features

- **Interactive Dashboard:** Built with Bubbletea for a smooth, lag-free terminal experience.
- **Subscription Management:** Add, update, edit, and delete Subscriptions or Local nodes easily.
- **Smart Viewport:** Scroll through hundreds of servers without breaking your terminal UI.
- **Built-in Latency Tester:** Concurrent TCP ping testing for individual nodes, groups, or all servers.
- **Multiple Connection Modes:**
  - `Manual`: Generates config and binds to your preferred SOCKS/HTTP port.
  - `System Proxy`: Automatically configures your GNOME/KDE desktop proxy settings.
  - `TUN Mode`: Routes entire OS traffic through Xray (Requires root/sudo).
- **QR Code Generator:** View configuration QR codes directly in the terminal.

---

## ⚙️ Prerequisites & Installation

To use this tool, you need two things: **Xray-core** and the **Xray CLI** itself.

### 1. Install Xray-core (Mandatory)
This CLI is a frontend wrapper; it requires the official `xray` binary to be installed and available in your system's `PATH`.

**Fastest way (Official Script for Linux):**
```bash
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install

```

**Manual way:**

1. Download the latest release from the https://github.com/XTLS/Xray-core/releases .
2. Unzip it and move the `xray` binary to `/usr/local/bin`:

```bash
unzip Xray-linux-64.zip
sudo mv xray /usr/local/bin/
sudo chmod +x /usr/local/bin/xray

```

### 2. Install Xray CLI

Ensure you have Go installed, then clone and build the project:

```bash
git clone https://github.com/Mhmdrz-rasekh/xray-cli.git
cd xray-cli
go mod tidy
go build -ldflags="-s -w" -o xray-cli main.go
sudo mv xray-cli /usr/local/bin/

```

## ⌨️ Shortcuts & Usage

Run the dashboard by typing `xray-cli` in your terminal.

| Key | Action | Key | Action |
| --- | --- | --- | --- |
| `↑` / `↓` | Navigate the list | `A` | Add Subscription |
| `M` | Connect (Manual Port) | `L` | Add Local Config |
| `S` | Connect (System Proxy) | `U` | Update Subscriptions |
| `T` | Connect (TUN Mode) | `E` | Edit Selected Node |
| `D` | Disconnect | `V` | View QR Code |
| `P` | Ping Selected Node | `x` | Delete Selected Node |
| `G` | Ping Entire Group | `Shift+X` | Delete Entire Sub/Group |
| `C` | Ping All Nodes | `Q` / `ESC` | Quit / Go Back |

---

---

# داشبورد تعاملی Xray CLI 🚀 (نسخه فارسی)

یک کلاینت سریع، تعاملی و قدرتمند تحت ترمینال (TUI) برای هسته Xray. با این ابزار می‌توانید ساب‌اسکریپشن‌های VLESS خود را مدیریت کنید، از سرورها پینگ بگیرید و به صورت مستقیم از طریق ترمینال در حالت‌های مختلف (پروکسی سیستم، حالت TUN یا دستی) به اینترنت آزاد متصل شوید.

## ✨ قابلیت‌ها

* **رابط کاربری تعاملی (TUI):** طراحی شده با کتابخانه قدرتمند Bubbletea برای تجربه‌ای روان در ترمینال.
* **مدیریت اشتراک‌ها:** افزودن، آپدیت، ویرایش و حذف لینک‌های ساب‌اسکریپشن یا کانفیگ‌های لوکال به سادگی.
* **اسکرول هوشمند:** امکان پیمایش بین صدها سرور بدون به هم ریختن ظاهر ترمینال.
* **پینگ‌گیر داخلی:** تست تاخیر (Ping) بر پایه TCP برای یک سرور، یک گروه یا تمام سرورها در پس‌زمینه.
* **حالت‌های اتصال متنوع:**
* `Manual` (دستی): تعیین پورت دلخواه SOCKS/HTTP.
* `System Proxy` (پروکسی سیستم): تنظیم خودکار پروکسی روی لینوکس (GNOME و KDE).
* `TUN Mode` (حالت تونل): تونل کردن کل ترافیک سیستم‌عامل (نیازمند دسترسی root).


* **تولید بارکد (QR Code):** نمایش بارکد کانفیگ‌ها مستقیماً در محیط ترمینال.

---

## ⚙️ پیش‌نیازها و نصب

برای استفاده از این ابزار به دو چیز نیاز دارید: **هسته Xray** و **برنامه Xray CLI**.

### ۱. نصب هسته Xray (الزامی)

این برنامه در واقع یک داشبورد برای کنترل Xray است، بنابراین باید فایل اجرایی `xray` روی سیستم شما نصب و در مسیر `PATH` قرار داشته باشد.

**روش سریع (اسکریپت رسمی):**

```bash
bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
```

**روش دستی:**
۱. آخرین نسخه را از [مخزن رسمی Xray-core](https://github.com/XTLS/Xray-core/releases) دانلود کنید.
۲. آن را از حالت فشرده خارج کرده و فایل `xray` را به مسیر برنامه‌های سیستم منتقل کنید:

```bash
unzip Xray-linux-64.zip
sudo mv xray /usr/local/bin/
sudo chmod +x /usr/local/bin/xray

```

### ۲. نصب Xray CLI

مطمئن شوید که زبان Go روی سیستم شما نصب است، سپس دستورات زیر را اجرا کنید:

```bash
git clone https://github.com/Mhmdrz-rasekh/xray-cli.git
cd xray-cli
go mod tidy
go build -ldflags="-s -w" -o xray-cli main.go
sudo mv xray-cli /usr/local/bin/

```

*(فراموش نکنید که `YOUR_USERNAME` را با یوزرنیم گیت‌هاب خود جایگزین کنید).*

## ⌨️ راهنمای کلیدها

برای اجرای برنامه کافیست کلمه `xray-cli` را در ترمینال تایپ کنید.

| کلید | عملکرد | کلید | عملکرد |
| --- | --- | --- | --- |
| `↑` / `↓` | حرکت در لیست | `A` | افزودن ساب‌اسکریپشن |
| `M` | اتصال (انتخاب پورت دستی) | `L` | افزودن کانفیگ لوکال |
| `S` | اتصال (پروکسی سیستم) | `U` | آپدیت تمام ساب‌ها |
| `T` | اتصال (حالت تونل کل سیستم) | `E` | ویرایش سرور انتخاب شده |
| `D` | قطع اتصال (Disconnect) | `V` | نمایش بارکد (QR Code) |
| `P` | پینگ گرفتن از یک سرور | `x` (کوچک) | حذف یک سرور |
| `G` | پینگ گرفتن از کل گروه | `X` (بزرگ) | حذف کل یک ساب‌اسکریپشن |
| `C` | پینگ گرفتن از همه سرورها | `Q` / `ESC` | خروج / بازگشت |

---

*Developed with ❤️ by MohammadReza Rasekh*

```

**دو نکته کوچک برای نهایی کردن کار:**
1. در بخش انگلیسی، یادتان نرود که یک عکس از محیط داشبوردتان بگیرید و لینک آن را در خط ۵ (جایی که نوشته‌ام `[Dashboard Screenshot]`) قرار دهید.
2. در بخش دستورات `git clone`، عبارت `YOUR_USERNAME` را پیدا کنید و نام کاربری خودتان (`Mhmdrz-rasekh`) را به جای آن بنویسید.

به دنیای توسعه‌دهندگان متن‌باز (Open-Source) خوش آمدید! 🚀

```
