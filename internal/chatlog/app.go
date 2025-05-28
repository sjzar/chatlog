package chatlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/sjzar/chatlog/internal/chatlog/ctx"
	"github.com/sjzar/chatlog/internal/ui/footer"
	"github.com/sjzar/chatlog/internal/ui/form"
	"github.com/sjzar/chatlog/internal/ui/help"
	"github.com/sjzar/chatlog/internal/ui/infobar"
	"github.com/sjzar/chatlog/internal/ui/menu"
	"github.com/sjzar/chatlog/internal/wechat"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	RefreshInterval = 1000 * time.Millisecond
)

type App struct {
	*tview.Application

	ctx         *ctx.Context
	m           *Manager
	stopRefresh chan struct{}

	// page
	mainPages *tview.Pages
	infoBar   *infobar.InfoBar
	tabPages  *tview.Pages
	footer    *footer.Footer

	// tab
	menu      *menu.Menu
	help      *help.Help
	activeTab int
	tabCount  int
}

func NewApp(ctx *ctx.Context, m *Manager) *App {
	app := &App{
		ctx:         ctx,
		m:           m,
		Application: tview.NewApplication(),
		mainPages:   tview.NewPages(),
		infoBar:     infobar.New(),
		tabPages:    tview.NewPages(),
		footer:      footer.New(),
		menu:        menu.New("主菜单"),
		help:        help.New(),
	}

	app.initMenu()

	app.updateMenuItemsState()

	return app
}

func (a *App) Run() error {

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.infoBar, infobar.InfoBarViewHeight, 0, false).
		AddItem(a.tabPages, 0, 1, true).
		AddItem(a.footer, 1, 1, false)

	a.mainPages.AddPage("main", flex, true, true)

	a.tabPages.
		AddPage("0", a.menu, true, true).
		AddPage("1", a.help, true, false)
	a.tabCount = 2

	a.SetInputCapture(a.inputCapture)

	go a.refresh()

	if err := a.SetRoot(a.mainPages, true).EnableMouse(false).Run(); err != nil {
		return err
	}

	return nil
}

func (a *App) Stop() {
	// 添加一个通道用于停止刷新 goroutine
	if a.stopRefresh != nil {
		close(a.stopRefresh)
	}
	a.Application.Stop()
}

func (a *App) updateMenuItemsState() {
	// 查找并更新自动解密菜单项
	for _, item := range a.menu.GetItems() {
		// 更新自动解密菜单项
		if item.Index == 5 {
			if a.ctx.AutoDecrypt {
				item.Name = "停止自动解密"
				item.Description = "停止监控数据目录更新，不再自动解密新增数据"
			} else {
				item.Name = "开启自动解密"
				item.Description = "监控数据目录更新，自动解密新增数据"
			}
		}

		// 更新HTTP服务菜单项
		if item.Index == 4 {
			if a.ctx.HTTPEnabled {
				item.Name = "停止 HTTP 服务"
				item.Description = "停止本地 HTTP & MCP 服务器"
			} else {
				item.Name = "启动 HTTP 服务"
				item.Description = "启动本地 HTTP & MCP 服务器"
			}
		}
	}
}

func (a *App) switchTab(step int) {
	index := (a.activeTab + step) % a.tabCount
	if index < 0 {
		index = a.tabCount - 1
	}
	a.activeTab = index
	a.tabPages.SwitchToPage(fmt.Sprint(a.activeTab))
}

func (a *App) refresh() {
	tick := time.NewTicker(RefreshInterval)
	defer tick.Stop()

	for {
		select {
		case <-a.stopRefresh:
			return
		case <-tick.C:
			if a.ctx.AutoDecrypt || a.ctx.HTTPEnabled {
				a.m.RefreshSession()
			}
			a.infoBar.UpdateAccount(a.ctx.Account)
			a.infoBar.UpdateBasicInfo(a.ctx.PID, a.ctx.FullVersion, a.ctx.ExePath)
			a.infoBar.UpdateStatus(a.ctx.Status)
			a.infoBar.UpdateDataKey(a.ctx.DataKey)
			a.infoBar.UpdatePlatform(a.ctx.Platform)
			a.infoBar.UpdateDataUsageDir(a.ctx.DataUsage, a.ctx.DataDir)
			a.infoBar.UpdateWorkUsageDir(a.ctx.WorkUsage, a.ctx.WorkDir)
			if a.ctx.LastSession.Unix() > 1000000000 {
				a.infoBar.UpdateSession(a.ctx.LastSession.Format("2006-01-02 15:04:05"))
			}
			if a.ctx.HTTPEnabled {
				a.infoBar.UpdateHTTPServer(fmt.Sprintf("[green][已启动][white] [%s]", a.ctx.HTTPAddr))
			} else {
				a.infoBar.UpdateHTTPServer("[未启动]")
			}
			if a.ctx.AutoDecrypt {
				a.infoBar.UpdateAutoDecrypt("[green][已开启][white]")
			} else {
				a.infoBar.UpdateAutoDecrypt("[未开启]")
			}

			a.Draw()
		}
	}
}

func (a *App) inputCapture(event *tcell.EventKey) *tcell.EventKey {

	// 如果当前页面不是主页面，ESC 键返回主页面
	if a.mainPages.HasPage("submenu") && event.Key() == tcell.KeyEscape {
		a.mainPages.RemovePage("submenu")
		a.mainPages.SwitchToPage("main")
		return nil
	}

	if a.tabPages.HasFocus() {
		switch event.Key() {
		case tcell.KeyLeft:
			a.switchTab(-1)
			return nil
		case tcell.KeyRight:
			a.switchTab(1)
			return nil
		}
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		a.Stop()
	}

	return event
}

func (a *App) initMenu() {
	getDataKey := &menu.Item{
		Index:       2,
		Name:        "获取数据密钥",
		Description: "从进程获取数据密钥",
		Selected: func(i *menu.Item) {
			modal := tview.NewModal()
			if runtime.GOOS == "darwin" {
				modal.SetText("获取数据密钥中...\n预计需要 20 秒左右的时间，期间微信会卡住，请耐心等待")
			} else {
				modal.SetText("获取数据密钥中...")
			}
			a.mainPages.AddPage("modal", modal, true, true)
			a.SetFocus(modal)

			go func() {
				err := a.m.GetDataKey()

				// 在主线程中更新UI
				a.QueueUpdateDraw(func() {
					if err != nil {
						// 解密失败
						modal.SetText("获取数据密钥失败: " + err.Error())
					} else {
						// 解密成功
						modal.SetText("获取数据密钥成功")
					}

					// 添加确认按钮
					modal.AddButtons([]string{"OK"})
					modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						a.mainPages.RemovePage("modal")
					})
					a.SetFocus(modal)
				})
			}()
		},
	}

	decryptData := &menu.Item{
		Index:       3,
		Name:        "解密数据",
		Description: "解密数据文件",
		Selected: func(i *menu.Item) {
			// 创建一个没有按钮的模态框，显示"解密中..."
			modal := tview.NewModal().
				SetText("解密中...")

			a.mainPages.AddPage("modal", modal, true, true)
			a.SetFocus(modal)

			// 在后台执行解密操作
			go func() {
				// 执行解密
				err := a.m.DecryptDBFiles()

				// 在主线程中更新UI
				a.QueueUpdateDraw(func() {
					if err != nil {
						// 解密失败
						modal.SetText("解密失败: " + err.Error())
					} else {
						// 解密成功
						modal.SetText("解密数据成功")
					}

					// 添加确认按钮
					modal.AddButtons([]string{"OK"})
					modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						a.mainPages.RemovePage("modal")
					})
					a.SetFocus(modal)
				})
			}()
		},
	}

	httpServer := &menu.Item{
		Index:       4,
		Name:        "启动 HTTP 服务",
		Description: "启动本地 HTTP & MCP 服务器",
		Selected: func(i *menu.Item) {
			modal := tview.NewModal()

			// 根据当前服务状态执行不同操作
			if !a.ctx.HTTPEnabled {
				// HTTP 服务未启动，启动服务
				modal.SetText("正在启动 HTTP 服务...")
				a.mainPages.AddPage("modal", modal, true, true)
				a.SetFocus(modal)

				// 在后台启动服务
				go func() {
					err := a.m.StartService()

					// 在主线程中更新UI
					a.QueueUpdateDraw(func() {
						if err != nil {
							// 启动失败
							modal.SetText("启动 HTTP 服务失败: " + err.Error())
						} else {
							// 启动成功
							modal.SetText("已启动 HTTP 服务")
						}

						// 更改菜单项名称
						a.updateMenuItemsState()

						// 添加确认按钮
						modal.AddButtons([]string{"OK"})
						modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							a.mainPages.RemovePage("modal")
						})
						a.SetFocus(modal)
					})
				}()
			} else {
				// HTTP 服务已启动，停止服务
				modal.SetText("正在停止 HTTP 服务...")
				a.mainPages.AddPage("modal", modal, true, true)
				a.SetFocus(modal)

				// 在后台停止服务
				go func() {
					err := a.m.StopService()

					// 在主线程中更新UI
					a.QueueUpdateDraw(func() {
						if err != nil {
							// 停止失败
							modal.SetText("停止 HTTP 服务失败: " + err.Error())
						} else {
							// 停止成功
							modal.SetText("已停止 HTTP 服务")
						}

						// 更改菜单项名称
						a.updateMenuItemsState()

						// 添加确认按钮
						modal.AddButtons([]string{"OK"})
						modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							a.mainPages.RemovePage("modal")
						})
						a.SetFocus(modal)
					})
				}()
			}
		},
	}

	autoDecrypt := &menu.Item{
		Index:       5,
		Name:        "开启自动解密",
		Description: "自动解密新增的数据文件",
		Selected: func(i *menu.Item) {
			modal := tview.NewModal()

			// 根据当前自动解密状态执行不同操作
			if !a.ctx.AutoDecrypt {
				// 自动解密未开启，开启自动解密
				modal.SetText("正在开启自动解密...")
				a.mainPages.AddPage("modal", modal, true, true)
				a.SetFocus(modal)

				// 在后台开启自动解密
				go func() {
					err := a.m.StartAutoDecrypt()

					// 在主线程中更新UI
					a.QueueUpdateDraw(func() {
						if err != nil {
							// 开启失败
							modal.SetText("开启自动解密失败: " + err.Error())
						} else {
							// 开启成功
							if a.ctx.Version == 3 {
								modal.SetText("已开启自动解密\n3.x版本数据文件更新不及时，有低延迟需求请使用4.0版本")
							} else {
								modal.SetText("已开启自动解密")
							}
						}

						// 更改菜单项名称
						a.updateMenuItemsState()

						// 添加确认按钮
						modal.AddButtons([]string{"OK"})
						modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							a.mainPages.RemovePage("modal")
						})
						a.SetFocus(modal)
					})
				}()
			} else {
				// 自动解密已开启，停止自动解密
				modal.SetText("正在停止自动解密...")
				a.mainPages.AddPage("modal", modal, true, true)
				a.SetFocus(modal)

				// 在后台停止自动解密
				go func() {
					err := a.m.StopAutoDecrypt()

					// 在主线程中更新UI
					a.QueueUpdateDraw(func() {
						if err != nil {
							// 停止失败
							modal.SetText("停止自动解密失败: " + err.Error())
						} else {
							// 停止成功
							modal.SetText("已停止自动解密")
						}

						// 更改菜单项名称
						a.updateMenuItemsState()

						// 添加确认按钮
						modal.AddButtons([]string{"OK"})
						modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							a.mainPages.RemovePage("modal")
						})
						a.SetFocus(modal)
					})
				}()
			}
		},
	}

	setting := &menu.Item{
		Index:       6,
		Name:        "设置",
		Description: "设置应用程序选项",
		Selected:    a.settingSelected,
	}

	selectAccount := &menu.Item{
		Index:       7,
		Name:        "切换账号",
		Description: "切换当前操作的账号，可以选择进程或历史账号",
		Selected:    a.selectAccountSelected,
	}

	a.menu.AddItem(getDataKey)
	a.menu.AddItem(decryptData)
	a.menu.AddItem(httpServer)
	a.menu.AddItem(autoDecrypt)
	a.menu.AddItem(setting)
	a.menu.AddItem(selectAccount)

	exportChatRecords := &menu.Item{
		Index:       9,
		Name:        "Export Chat Records",
		Description: "Export chat records to a file",
		Selected:    a.exportChatRecordsSelected,
	}
	a.menu.AddItem(exportChatRecords)

	a.menu.AddItem(&menu.Item{
		Index:       10,
		Name:        "退出",
		Description: "退出程序",
		Selected: func(i *menu.Item) {
			a.Stop()
		},
	})
}

// exportChatRecordsSelected handles the selection of the "Export Chat Records" menu item.
func (a *App) exportChatRecordsSelected(i *menu.Item) {
	// Fetch chat sessions
	sessionsResponse, err := a.m.db.GetSessions("", 100, 0)
	if err != nil {
		a.showError(fmt.Errorf("failed to get chat sessions: %v", err))
		return
	}

	if len(sessionsResponse.Items) == 0 {
		a.showInfo("No chat sessions available to export.")
		return
	}

	subMenu := menu.NewSubMenu("Select Chat to Export")
	for idx, session := range sessionsResponse.Items {
		// Use session.Talker or session.Name. Assuming session.Talker is the primary identifier.
		// The actual fields depend on the database.Session struct.
		// For description, we can use the last message time.
		talkerName := session.Talker
		if talkerName == "" {
			talkerName = "Unknown Talker" // Fallback if Talker is empty
		}
		// Format NTime to a readable string, assuming NTime is a time.Time object or similar
		description := fmt.Sprintf("Last message: %s", session.NTime.Format("2006-01-02 15:04:05"))
		if session.Name != "" { // If a more friendly name is available
			description = fmt.Sprintf("Name: %s, %s", session.Name, description)
		}


		item := &menu.Item{
			Index:       idx + 1,
			Name:        talkerName,
			Description: description,
			Selected: func(selectedTalker string, talkerDisplayName string) func(*menu.Item) {
				return func(*menu.Item) {
					exportOptionsMenu := menu.NewSubMenu("Export Options for " + talkerDisplayName)
					exportOptionsMenu.AddItem(&menu.Item{
						Index:       1,
						Name:        "Export today's chat records",
						Description: "Export chat records from today for " + talkerDisplayName,
						Selected: func(*menu.Item) {
							go a.exportChatMessages(selectedTalker, talkerDisplayName, "today")
						},
					})
					exportOptionsMenu.AddItem(&menu.Item{
						Index:       2,
						Name:        "Export last 7 days' chat records",
						Description: "Export chat records from the last 7 days for " + talkerDisplayName,
						Selected: func(*menu.Item) {
							go a.exportChatMessages(selectedTalker, talkerDisplayName, "last7days")
						},
					})

					// It's important to use a unique page name for each level of submenu
					// to avoid conflicts and allow proper back navigation with ESC.
					// However, tview's default ESC behavior with AddPage might just go back one page.
					// If "submenu" was used for settings, this needs a different name.
					a.mainPages.AddPage("submenuExportOptions", exportOptionsMenu, true, true)
					a.SetFocus(exportOptionsMenu)
				}
			}(session.Talker, talkerName), // Pass the correct talker identifier and display name
		}
		subMenu.AddItem(item)
	}

	// Use a unique page name for the talker selection submenu
	a.mainPages.AddPage("submenuExportTalker", subMenu, true, true)
	a.SetFocus(subMenu)
}

func sanitizeFilename(name string) string {
	// Remove invalid characters for most file systems
	// Keep it simple: replace non-alphanumeric with underscore
	reg := regexp.MustCompile("[^a-zA-Z0-9_.-]+")
	return reg.ReplaceAllString(name, "_")
}

// exportChatMessages handles the logic for exporting chat messages to a file.
func (a *App) exportChatMessages(talkerID string, talkerDisplayName string, dateRangeType string) {
	modal := tview.NewModal().SetText(fmt.Sprintf("Exporting messages for %s...", talkerDisplayName))
	a.QueueUpdateDraw(func() {
		a.mainPages.AddPage("modalExporting", modal, true, true)
		a.SetFocus(modal)
	})

	defer a.QueueUpdateDraw(func() {
		a.mainPages.RemovePage("modalExporting")
	})

	var startTime, endTime time.Time
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fileNameDateSuffix := ""

	switch dateRangeType {
	case "today":
		startTime = todayStart
		endTime = todayStart.Add(24 * time.Hour).Add(-1 * time.Nanosecond) // End of today
		fileNameDateSuffix = startTime.Format("2006-01-02")
	case "last7days":
		startTime = todayStart.AddDate(0, 0, -6) // 7 days ago including today
		endTime = todayStart.Add(24 * time.Hour).Add(-1 * time.Nanosecond) // End of today
		fileNameDateSuffix = fmt.Sprintf("%s_to_%s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	default:
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Errorf("invalid date range type: %s", dateRangeType))
		})
		return
	}

	// Fetch messages (limit 0 for all messages in range)
	// Assuming model.Message has CreateTime (time.Time), Speaker (string), Content (string)
	messages, err := a.m.db.GetMessages(startTime, endTime, talkerID, "", "", 0, 0)
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Errorf("failed to fetch messages for %s: %v", talkerDisplayName, err))
		})
		return
	}

	if len(messages) == 0 {
		a.QueueUpdateDraw(func() {
			a.showInfo(fmt.Sprintf("No messages found for %s in the selected date range (%s).", talkerDisplayName, dateRangeType))
		})
		return
	}

	var contentBuilder strings.Builder
	for _, msg := range messages {
		// Assuming msg.CreateTime is time.Time, msg.Speaker, msg.StrContent
		// Adjust field names if model.Message struct is different
		// Example: msg.CreateTime.Format might need to be msg.Timestamp if that's the field name
		// For now, let's assume it's msg.NTime like in Session for consistency if it's a direct time field
		// Or it might be CreateTime as is common. Let's use CreateTime as a placeholder.
		// If GetMessages returns model.Message, we need its definition.
		// Let's assume it's like: type Message struct { CreateTime int64; Speaker string; Content string; ... }
		// And that CreateTime is a Unix timestamp.
		// Or if CreateTime is already time.Time, then msg.CreateTime.Format(...) is fine.
		// The database.GetMessages returns []*model.Message. The model.Message struct has `CreateTime int64`.
		var msgTime time.Time
		if msg.CreateTime > 0 { // Assuming CreateTime is int64 Unix timestamp
			msgTime = time.Unix(msg.CreateTime, 0)
		} else {
			msgTime = time.Now() // Fallback, though unlikely if CreateTime is primary timestamp
		}
		contentBuilder.WriteString(fmt.Sprintf("[%s] %s: %s\n", msgTime.Format("2006-01-02 15:04:05"), msg.Speaker, msg.Content))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Errorf("failed to get user home directory: %v", err))
		})
		return
	}

	exportDir := filepath.Join(homeDir, "chatlog_exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Errorf("failed to create export directory %s: %v", exportDir, err))
		})
		return
	}

	baseFilename := sanitizeFilename(talkerDisplayName)
	filename := fmt.Sprintf("%s_%s.txt", baseFilename, fileNameDateSuffix)
	filePath := filepath.Join(exportDir, filename)

	if err := os.WriteFile(filePath, []byte(contentBuilder.String()), 0644); err != nil {
		a.QueueUpdateDraw(func() {
			a.showError(fmt.Errorf("failed to save chat log to %s: %v", filePath, err))
		})
		return
	}

	a.QueueUpdateDraw(func() {
		a.showInfo(fmt.Sprintf("Chat log for %s exported successfully to:\n%s", talkerDisplayName, filePath))
	})
}


// settingItem 表示一个设置项
type settingItem struct {
	name        string
	description string
	action      func()
}

func (a *App) settingSelected(i *menu.Item) {

	settings := []settingItem{
		{
			name:        "设置 HTTP 服务地址",
			description: "配置 HTTP 服务监听的地址",
			action:      a.settingHTTPPort,
		},
		{
			name:        "设置工作目录",
			description: "配置数据解密后的存储目录",
			action:      a.settingWorkDir,
		},
		{
			name:        "设置数据密钥",
			description: "配置数据解密密钥",
			action:      a.settingDataKey,
		},
		{
			name:        "设置数据目录",
			description: "配置微信数据文件所在目录",
			action:      a.settingDataDir,
		},
	}

	subMenu := menu.NewSubMenu("设置")
	for idx, setting := range settings {
		item := &menu.Item{
			Index:       idx + 1,
			Name:        setting.name,
			Description: setting.description,
			Selected: func(action func()) func(*menu.Item) {
				return func(*menu.Item) {
					action()
				}
			}(setting.action),
		}
		subMenu.AddItem(item)
	}

	a.mainPages.AddPage("submenu", subMenu, true, true)
	a.SetFocus(subMenu)
}

// settingHTTPPort 设置 HTTP 端口
func (a *App) settingHTTPPort() {
	// 使用我们的自定义表单组件
	formView := form.NewForm("设置 HTTP 地址")

	// 临时存储用户输入的值
	tempHTTPAddr := a.ctx.HTTPAddr

	// 添加输入字段 - 不再直接设置HTTP地址，而是更新临时变量
	formView.AddInputField("地址", tempHTTPAddr, 0, nil, func(text string) {
		tempHTTPAddr = text // 只更新临时变量
	})

	// 添加按钮 - 点击保存时才设置HTTP地址
	formView.AddButton("保存", func() {
		a.m.SetHTTPAddr(tempHTTPAddr) // 在这里设置HTTP地址
		a.mainPages.RemovePage("submenu2")
		a.showInfo("HTTP 地址已设置为 " + a.ctx.HTTPAddr)
	})

	formView.AddButton("取消", func() {
		a.mainPages.RemovePage("submenu2")
	})

	a.mainPages.AddPage("submenu2", formView, true, true)
	a.SetFocus(formView)
}

// settingWorkDir 设置工作目录
func (a *App) settingWorkDir() {
	// 使用我们的自定义表单组件
	formView := form.NewForm("设置工作目录")

	// 临时存储用户输入的值
	tempWorkDir := a.ctx.WorkDir

	// 添加输入字段 - 不再直接设置工作目录，而是更新临时变量
	formView.AddInputField("工作目录", tempWorkDir, 0, nil, func(text string) {
		tempWorkDir = text // 只更新临时变量
	})

	// 添加按钮 - 点击保存时才设置工作目录
	formView.AddButton("保存", func() {
		a.ctx.SetWorkDir(tempWorkDir) // 在这里设置工作目录
		a.mainPages.RemovePage("submenu2")
		a.showInfo("工作目录已设置为 " + a.ctx.WorkDir)
	})

	formView.AddButton("取消", func() {
		a.mainPages.RemovePage("submenu2")
	})

	a.mainPages.AddPage("submenu2", formView, true, true)
	a.SetFocus(formView)
}

// settingDataKey 设置数据密钥
func (a *App) settingDataKey() {
	// 使用我们的自定义表单组件
	formView := form.NewForm("设置数据密钥")

	// 临时存储用户输入的值
	tempDataKey := a.ctx.DataKey

	// 添加输入字段 - 不直接设置数据密钥，而是更新临时变量
	formView.AddInputField("数据密钥", tempDataKey, 0, nil, func(text string) {
		tempDataKey = text // 只更新临时变量
	})

	// 添加按钮 - 点击保存时才设置数据密钥
	formView.AddButton("保存", func() {
		a.ctx.DataKey = tempDataKey // 设置数据密钥
		a.mainPages.RemovePage("submenu2")
		a.showInfo("数据密钥已设置")
	})

	formView.AddButton("取消", func() {
		a.mainPages.RemovePage("submenu2")
	})

	a.mainPages.AddPage("submenu2", formView, true, true)
	a.SetFocus(formView)
}

// settingDataDir 设置数据目录
func (a *App) settingDataDir() {
	// 使用我们的自定义表单组件
	formView := form.NewForm("设置数据目录")

	// 临时存储用户输入的值
	tempDataDir := a.ctx.DataDir

	// 添加输入字段 - 不直接设置数据目录，而是更新临时变量
	formView.AddInputField("数据目录", tempDataDir, 0, nil, func(text string) {
		tempDataDir = text // 只更新临时变量
	})

	// 添加按钮 - 点击保存时才设置数据目录
	formView.AddButton("保存", func() {
		a.ctx.DataDir = tempDataDir // 设置数据目录
		a.mainPages.RemovePage("submenu2")
		a.showInfo("数据目录已设置为 " + a.ctx.DataDir)
	})

	formView.AddButton("取消", func() {
		a.mainPages.RemovePage("submenu2")
	})

	a.mainPages.AddPage("submenu2", formView, true, true)
	a.SetFocus(formView)
}

// selectAccountSelected 处理切换账号菜单项的选择事件
func (a *App) selectAccountSelected(i *menu.Item) {
	// 创建子菜单
	subMenu := menu.NewSubMenu("切换账号")

	// 添加微信进程
	instances := a.m.wechat.GetWeChatInstances()
	if len(instances) > 0 {
		// 添加实例标题
		subMenu.AddItem(&menu.Item{
			Index:       0,
			Name:        "--- 微信进程 ---",
			Description: "",
			Hidden:      false,
			Selected:    nil,
		})

		// 添加实例列表
		for idx, instance := range instances {
			// 创建一个实例描述
			description := fmt.Sprintf("版本: %s 目录: %s", instance.FullVersion, instance.DataDir)

			// 标记当前选中的实例
			name := fmt.Sprintf("%s [%d]", instance.Name, instance.PID)
			if a.ctx.Current != nil && a.ctx.Current.PID == instance.PID {
				name = name + " [当前]"
			}

			// 创建菜单项
			instanceItem := &menu.Item{
				Index:       idx + 1,
				Name:        name,
				Description: description,
				Hidden:      false,
				Selected: func(instance *wechat.Account) func(*menu.Item) {
					return func(*menu.Item) {
						// 如果是当前账号，则无需切换
						if a.ctx.Current != nil && a.ctx.Current.PID == instance.PID {
							a.mainPages.RemovePage("submenu")
							a.showInfo("已经是当前账号")
							return
						}

						// 显示切换中的模态框
						modal := tview.NewModal().SetText("正在切换账号...")
						a.mainPages.AddPage("modal", modal, true, true)
						a.SetFocus(modal)

						// 在后台执行切换操作
						go func() {
							err := a.m.Switch(instance, "")

							// 在主线程中更新UI
							a.QueueUpdateDraw(func() {
								a.mainPages.RemovePage("modal")
								a.mainPages.RemovePage("submenu")

								if err != nil {
									// 切换失败
									a.showError(fmt.Errorf("切换账号失败: %v", err))
								} else {
									// 切换成功
									a.showInfo("切换账号成功")
									// 更新菜单状态
									a.updateMenuItemsState()
								}
							})
						}()
					}
				}(instance),
			}
			subMenu.AddItem(instanceItem)
		}
	}

	// 添加历史账号
	if len(a.ctx.History) > 0 {
		// 添加历史账号标题
		subMenu.AddItem(&menu.Item{
			Index:       100,
			Name:        "--- 历史账号 ---",
			Description: "",
			Hidden:      false,
			Selected:    nil,
		})

		// 添加历史账号列表
		idx := 101
		for account, hist := range a.ctx.History {
			// 创建一个账号描述
			description := fmt.Sprintf("版本: %s 目录: %s", hist.FullVersion, hist.DataDir)

			// 标记当前选中的账号
			name := account
			if name == "" {
				name = filepath.Base(hist.DataDir)
			}
			if a.ctx.DataDir == hist.DataDir {
				name = name + " [当前]"
			}

			// 创建菜单项
			histItem := &menu.Item{
				Index:       idx,
				Name:        name,
				Description: description,
				Hidden:      false,
				Selected: func(account string) func(*menu.Item) {
					return func(*menu.Item) {
						// 如果是当前账号，则无需切换
						if a.ctx.Current != nil && a.ctx.DataDir == a.ctx.History[account].DataDir {
							a.mainPages.RemovePage("submenu")
							a.showInfo("已经是当前账号")
							return
						}

						// 显示切换中的模态框
						modal := tview.NewModal().SetText("正在切换账号...")
						a.mainPages.AddPage("modal", modal, true, true)
						a.SetFocus(modal)

						// 在后台执行切换操作
						go func() {
							err := a.m.Switch(nil, account)

							// 在主线程中更新UI
							a.QueueUpdateDraw(func() {
								a.mainPages.RemovePage("modal")
								a.mainPages.RemovePage("submenu")

								if err != nil {
									// 切换失败
									a.showError(fmt.Errorf("切换账号失败: %v", err))
								} else {
									// 切换成功
									a.showInfo("切换账号成功")
									// 更新菜单状态
									a.updateMenuItemsState()
								}
							})
						}()
					}
				}(account),
			}
			idx++
			subMenu.AddItem(histItem)
		}
	}

	// 如果没有账号可选择
	if len(a.ctx.History) == 0 && len(instances) == 0 {
		subMenu.AddItem(&menu.Item{
			Index:       1,
			Name:        "无可用账号",
			Description: "未检测到微信进程或历史账号",
			Hidden:      false,
			Selected:    nil,
		})
	}

	// 显示子菜单
	a.mainPages.AddPage("submenu", subMenu, true, true)
	a.SetFocus(subMenu)
}

// showModal 显示一个模态对话框
func (a *App) showModal(text string, buttons []string, doneFunc func(buttonIndex int, buttonLabel string)) {
	modal := tview.NewModal().
		SetText(text).
		AddButtons(buttons).
		SetDoneFunc(doneFunc)

	a.mainPages.AddPage("modal", modal, true, true)
	a.SetFocus(modal)
}

// showError 显示错误对话框
func (a *App) showError(err error) {
	a.showModal(err.Error(), []string{"OK"}, func(buttonIndex int, buttonLabel string) {
		a.mainPages.RemovePage("modal")
	})
}

// showInfo 显示信息对话框
func (a *App) showInfo(text string) {
	a.showModal(text, []string{"OK"}, func(buttonIndex int, buttonLabel string) {
		a.mainPages.RemovePage("modal")
	})
}
