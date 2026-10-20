# 介面譯成繁體中文

研究日期：2026-10-21。對象是這個工具自己的全螢幕介面，不是 Ghostty 應用程式的選單，也不是 README。

## 結論

可以譯，而且文字模式的格子寬度已經算進東亞全形字。預設的圖片介面現在還畫不出中文：內嵌的 Google Sans 沒有漢字，缺字會被換成 `?`。換行也只在空白處斷開，一整句沒有空格的中文會成一整行。

建議順序：

1. 讓圖片介面能畫出譯文會用到的字，再談翻譯。
2. 換行改成依格子或字形寬度斷開，不能只靠空白。
3. 用一層查表翻譯。英文是原文，缺譯回英文，不回簡體。指令名稱、設定檔的鍵與值、主題名稱維持英文。
4. 用語對齊 Ghostty 的 `zh_TW` 目錄。語言選擇記在這個工具自己的介面狀態，並提供 `-lang`。

英文路徑必須保持現在的格子寬度。測試把每一行鎖在固定寬度上。

## 介面上有兩套畫法

在 Ghostty 與 Kitty 裡，介面是用 Kitty graphics protocol 畫成圖。其他終端、以及 tmux、screen 裡，改畫終端格子。README 的 Display 一節這樣寫，程式在 `internal/tui/screen.go` 的 `layout`：`pixelOn()` 時圖自己排，否則用格子欄寬。

使用者看得到的英文散在這些地方，沒有字串目錄：

| 位置 | 內容 |
| --- | --- |
| `internal/tui/sections.go` | 七個區與小節標題：Themes、Fonts、Window、Input、Session、Extensions、Tool |
| `internal/tui/commands.go` 的 `buildCommands` | 約 22 條指令的 `desc`；`name` 是使用者打進去的字 |
| 同檔 `buildSettings` | 每個設定的 `label`、`detail`、`explain`、`note` |
| `internal/tui/dialogs.go` 的 `helpKeyGroups` | 說明視窗的按鍵說明 |
| `internal/tui/pxview.go` | 圖片介面另寫的標題，例如 “Keys and commands” |
| 歡迎、更新、確認對話框 | `dialogs.go` 與其呼叫端 |

`command.name`（`theme`、`font`、`opacity`）同時是提示列要打的字，也是寫進操作流程的識別名。`setting.key`（`background-opacity`、`macos-titlebar-style`）是 Ghostty 設定檔的鍵，寫進檔案的值（`native`、`on`、`0.85`）也是。這些不能譯。

## 圖片介面畫不出漢字

`internal/kimg/fonts.go` 只內嵌 Google Sans Code 與 Google Sans Flex，並註明圖片因此不跟著終端字型變。`Covers` 用字形索引判斷某個 face 有沒有這個字。

`internal/pix/pix.go` 的 `runs` 在畫字與量寬度時處理缺字：碼位大於 127、目前 face 沒有這個字，就改試等寬 face；還是沒有，就查 `substitutes`（⌘、⌥ 這類符號）；再沒有就畫成 `?`。漢字會走最後一條。

主題卡片走的是 `internal/kimg/canvas.go` 的 `Text`，直接 `DrawString`，沒有這層替換。使用者若把主題命名成中文，卡片上會是缺字框，說明文字則是問號。兩條路徑都要補。

完整的 Noto CJK 或思源黑體一個字重就有數 MB，和「單一執行檔、沒有額外依賴」的安裝方式不合。做法是：

- 內嵌一份 SIL OFL 的繁體子集，只收譯文實際用到的字。Noto Sans CJK TC 或思源黑體 TC 都可以，授權與現有 Google Sans 的 OFL 一致。
- 子集以外的字（使用者打的主題名、搜尋字）再向系統要字形：macOS 的蘋方-繁，Linux 上已安裝的 Noto Sans CJK TC。系統沒有這個字時，維持現在的 `?`，不要悄悄換成簡體字形。

子集要在譯文穩定之後再切。先切一次、譯文又加字，那些新字仍會變成問號。

## 寬度已經算對，換行還沒有

文字模式的留白、截斷、面板都用顯示寬度，不是位元組數：

- `pad`、`truncate`、`fill` 用 `lipgloss.Width` 與 `ansi.Truncate`（`internal/tui/styles.go`）。
- 依賴鏈上已有 `github.com/mattn/go-runewidth`、`github.com/clipperhouse/displaywidth`、`github.com/rivo/uniseg`（`go.mod`）。全形漢字算 2 格。

所以「視窗」是 4 格，「Window」是 6 格，對齊不會把後面的字吃掉。風險在固定欄寬被英文長度寫死：

- 說明視窗左欄把按鍵名補到 12 格，整框寬 104、左欄 41（`dialogs.go` 的 `viewHelp`）。
- 主題牆的側欄在終端寬度至少 96 時是 18 格；設定頁側欄在至少 104 時是 26 格；主欄夾在 34 到 54 格（`screen.go` 的 `layout`）。
- 狀態列的短字（choose、change、open）換成中文通常更短。說明句有時更長，超寬會被 `truncate` 切成省略號。

圖片介面的欄寬是畫完再量（`Canvas.Width`），字形補上之後會自己重排。文字模式的欄寬是格數，譯文變長只會被截斷，不會把版面撐破。`internal/tui/tui_test.go` 把英文畫面鎖在 105 或 130 格，譯文不能改到英文那條路徑的輸出。

換行是另一件事。`wrapPlain`（`panels.go`）和 `Canvas.Wrap`（`pix.go`）都用 `strings.Fields`，只在空白斷行。中文句子通常沒有空白，會被當成一個詞。`Canvas.Fit` 已經依 rune 二分截斷，截斷沒問題，換行有問題。`Wrap` 要能在一個「詞」超過行寬時依寬度切開。`uniseg` 已在依賴裡，可用來避免把一個字族序列切成兩半。

## 譯什麼、留什麼

譯：區名、小節名、設定的 `label` 與 `detail`、`explain`、`note`、指令的 `desc`、說明視窗、歡迎與更新對話框、空狀態（“No favourites yet”）。

留英文：

- 打進去的指令名與語法裡的鍵：`theme`、`fav`、`opacity`、`background-opacity`。
- 寫進設定檔的值：`native`、`transparent`、`on`、`off`、數字、十六進位色碼。
- 主題名稱、字型家族名、檔案路徑、產品名 Ghostty。
- 按鍵名：Enter、Tab、Esc、F1、Delete。說明可以譯，鍵帽不譯。

說明裡可以寫「輸入 theme，再按 Enter」。使用者搜尋時，中文應找得到譯文，英文指令名仍然找得到。`menuOptions` 現在把 `title(cmd.name)` 當顯示名，譯文標籤要另放，不能取代 `name`。

原文用英式拼法（colour、favourite）。譯意思，不要把 favourite 先改成美式再譯。Ghostty `zh_TW` 把 favourite 這一類說成「星號」的動作時，這個工具的介面現在用 Star / favourite；對齊時用「加上星號」或「加入最愛」擇一，全介面同一個詞。

## 用語

Ghostty 主線已有 `po/zh_TW.po`（Language: zh_TW，Last-Translator 含 Alang Hsu，POT-Creation-Date: 2026-10-14）。那份目錄翻的是 GTK 介面與命令面板，不是這支工具，但重疊的詞應沿用，避免同一個 Ghostty 視窗裡兩套說法：

| 英文 | `zh_TW.po` 的譯法 | 出處 |
| --- | --- | --- |
| Window | 視窗 | msgid “Window” |
| Tab | 分頁 | msgid “Tab” |
| Split（動作） | 分割 | msgid “Split” |
| split（那一格） | 窗格 | msgid “Close Split” → 「關閉窗格」 |
| Config | 設定 | msgid “Config” |
| Reload Configuration | 重新載入設定 | msgid “Reload Configuration” |
| Font / font size | 字型 | msgid “Increase Font Size” → 「放大字型」 |
| clipboard | 剪貼簿 | msgid “Copy to Clipboard” |
| Quit | 結束 | msgid “Quit” |
| Cancel | 取消 | msgid “Cancel” |
| Close | 關閉 | msgid “Close” |
| Background opacity | 背景不透明度 | msgid “Toggle Background Opacity” |
| scrollback | 捲動緩衝區 | msgid “Clear the screen and scrollback.” |

`zh_TW.po` 裡沒有 Theme、Blur、Ligatures、Shader 這幾個設定名。建議：主題、模糊、連字、著色器。著色器若顯得過專，介面上可寫「Shader」，說明裡再解釋；一旦選定就固定。

語氣跟 Ghostty 譯者指南：對等、好懂、不逐字硬譯、中文不用英文的標題大小寫、省略號用 `…` 不用三個句點。台灣用語：字型、設定、預設、程式、資訊。不用「字体」「默認」「程序」「信息」。

簡體目錄 `zh_CN.po` 存在，而且 Ghostty 在 `src/os/i18n.zig` 的 `fixZhLocale` 把 `zh-Hans*` 對到 `zh_CN`、`zh-Hant*` 對到 `zh_TW`（港澳再細分）。GNU gettext 也要求想看繁體就設 `zh_TW`，不要拿 `zh_CN` 當繁體的後備。這個工具同樣不要在缺繁體時顯示簡體。

Ghostty 譯者指南仍寫明：本地化系統在 macOS 上還沒接上，所以那裡還看不到這些譯文。2026-01-05 的討論 [#10172](https://github.com/ghostty-org/ghostty/discussions/10172) 裡，維護者說 macOS 本地化仍在進行，當時打算放在 1.3，視 macOS 維護者的時間而定。因此不能假設使用者的 Ghostty 選單已經是中文，這支工具的介面要自己譯。

## 語言怎麼選

終端裡的 `LANG` 常常是 shell 的地區設定，不一定是系統偏好的語言。macOS 上兩者經常不同。自動跟 `LANG` 會讓一部分人突然看到中文、另一部分人怎麼設系統語言都還是英文。

建議：

- 預設維持英文，跟現在的行為一致。
- 在既有的 Interface 區加一個語言選項，寫進這個工具自己的狀態，不寫進 Ghostty 設定檔。
- `-lang en` 與 `-lang zh-Hant` 覆蓋該選項，方便測試。
- 辨認 `zh-Hant`、`zh_TW`、`zh_HK`、`zh_MO`。`zh_CN` 與 `zh-Hans` 維持英文，直到另有簡體目錄。

實作用一張 `map[string]string`，鍵是英文原文，查不到就回原文。`golang.org/x/text` 已在 `go.mod` 的間接依賴裡，需要語言標籤比對時再升成直接依賴即可。不必上 gettext：字串是 Go 字面值，發行物是單一靜態執行檔，沒有 C 工具鏈。

## 不在這次範圍

- README、安裝腳本、GitHub 發行說明。
- 252 個策展主題的英文名稱。
- Ghostty 自己的選單。那是上游 `po/zh_TW.po` 的工作。
- 簡體中文目錄。

## 來源

- 本倉庫：`README.md`；`internal/kimg/fonts.go`；`internal/pix/pix.go` 的 `runs`、`Fit`、`Wrap`；`internal/kimg/canvas.go` 的 `Text`；`internal/tui/styles.go`、`panels.go`、`screen.go`、`sections.go`、`commands.go`、`dialogs.go`；`go.mod`。
- Ghostty 譯者指南：<https://github.com/ghostty-org/ghostty/blob/main/po/README_TRANSLATORS.md>（含「macOS 上尚未實作本地化」與風格指南）。
- Ghostty `po/zh_TW.po`（main，擷取於 2026-10-21）：上表各 msgid。
- Ghostty `src/os/i18n.zig` 的 `fixZhLocale`：`zh-Hant` 對 `zh_TW`，`zh-Hans` 對 `zh_CN`。
- GNU gettext 手冊，LANGUAGE 變數一節對中文使用者的說明：繁體用 `zh_TW`，簡體用 `zh_CN`，兩者不互為後備。<https://www.gnu.org/software/gettext/manual/html_node/The-LANGUAGE-variable.html>
- Ghostty 討論 [#10172](https://github.com/ghostty-org/ghostty/discussions/10172)（2026-01-05）：macOS 介面語言當時仍不可切。
