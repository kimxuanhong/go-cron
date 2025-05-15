package cron

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// Cron là struct thực thi interface Scheduler, dùng để quản lý cron jobs.
type Cron struct {
	scheduler    *cron.Cron    // Đối tượng scheduler từ thư viện robfig/cron
	dirs         []string      // Danh sách thư mục để quét các file Go tìm cron
	cronHandlers []interface{} // Danh sách các struct chứa các hàm xử lý cron
}

// NewCronJob tạo mới một đối tượng Cron.
//
// Cách sử dụng:
//
//	cron := cron.NewCronJob()
func NewCronJob() Scheduler {
	return &Cron{
		scheduler: cron.New(cron.WithSeconds()),
	}
}

// AddJob thêm một cron job mới vào cron scheduler.
//
// Parameters:
//   - cronExpr: biểu thức cron (ví dụ: "0 0 * * * *" chạy mỗi giờ)
//   - jobFunc: hàm sẽ được thực thi theo lịch.
//
// Trả về lỗi nếu biểu thức cron không hợp lệ hoặc không thể thêm job.
//
// Cách sử dụng:
//
//	cron.AddJob("0 0 * * * *", myFunc)
func (c *Cron) AddJob(cronExpr string, jobFunc func()) error {
	if !isValidCronExpr(cronExpr) {
		cronExpr = viper.GetString(cronExpr)
		if !isValidCronExpr(cronExpr) {
			return fmt.Errorf("biểu thức cron không hợp lệ: %s", cronExpr)
		}
	}
	_, err := c.scheduler.AddFunc(cronExpr, jobFunc)
	if err != nil {
		return fmt.Errorf("không thể thêm cron job: %v", err)
	}
	return nil
}

// Start bắt đầu thực thi các cron jobs.
//
// Cách sử dụng:
//
//	cron.Start()
func (c *Cron) Start() error {
	if c.scheduler == nil {
		return fmt.Errorf("cron scheduler chưa được khởi tạo")
	}
	c.scheduler.Start()
	return nil
}

// Stop dừng thực thi tất cả cron jobs.
//
// Cách sử dụng:
//
//	cron.Stop()
func (c *Cron) Stop() {
	if c.scheduler != nil {
		c.scheduler.Stop()
	}
}

// SetDirs thiết lập danh sách thư mục để quét các file Go có định nghĩa cron.
//
// Cách sử dụng:
//
//	cron.SetDirs("./jobs", "./tasks")
func (c *Cron) SetDirs(dirs ...string) {
	c.dirs = append(c.dirs, dirs...)
}

// RegisterHandlers đăng ký các handler struct có chứa các hàm cron cần gọi.
//
// Cách sử dụng:
//
//	cron.RegisterHandlers(MyJobHandler{}, AnotherHandler{})
func (c *Cron) RegisterHandlers(handlers ...interface{}) {
	c.cronHandlers = append(c.cronHandlers, handlers...)
}

// ParseSourceForCronJobs tìm và đăng ký các cron job từ comment trong file Go.
//
// Tự động quét các thư mục được chỉ định (hoặc thư mục hiện tại nếu chưa chỉ định),
// tìm các comment dạng `// @Cron <biểu thức>` trước các hàm trong struct đã đăng ký,
// sau đó thêm các job này vào scheduler.
//
// Cách sử dụng:
//
//	cron.ParseSourceForCronJobs()
func (c *Cron) ParseSourceForCronJobs() {
	if len(c.cronHandlers) == 0 {
		return
	}

	if len(c.dirs) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("failed to get current working directory: %v", err)
		}
		c.dirs = append(c.dirs, dir)
	}

	var scanDirs func(string) error
	scanDirs = func(dir string) error {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			log.Printf("failed to scan folder %s: %v", dir, err)
			return err
		}

		for _, file := range files {
			cronEntries := ParseCronFromFile(file)
			for _, cronJob := range cronEntries {
				for _, handler := range c.cronHandlers {
					val := reflect.ValueOf(handler)
					method := val.MethodByName(cronJob.Handler)
					if !method.IsValid() {
						log.Printf("method %s not found in handler %T", cronJob.Handler, handler)
						continue
					}

					if method.Type().NumIn() == 0 {
						jobFunc := func(m reflect.Value) func() {
							return func() {
								defer func() {
									if r := recover(); r != nil {
										log.Printf("panic trong cron job %s: %v", cronJob.Handler, r)
									}
								}()
								m.Call(nil) // Gọi hàm không có tham số
							}
						}(method)

						if err := c.AddJob(cronJob.CronExpr, jobFunc); err != nil {
							log.Printf("failed to add cron job %s: %v", cronJob.Handler, err)
						} else {
							log.Printf("registered cron job %s with expression [%s]", cronJob.Handler, cronJob.CronExpr)
						}
					} else {
						log.Printf("invalid method %s (must have no parameters)", cronJob.Handler)
					}
				}
			}
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Printf("failed to read directory %s: %v", dir, err)
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				err := scanDirs(filepath.Join(dir, entry.Name()))
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	for _, dir := range c.dirs {
		err := scanDirs(dir)
		if err != nil {
			log.Printf("error scanning directory %s: %v", dir, err)
		}
	}
}

// ParseCron chứa biểu thức cron và tên hàm handler tương ứng được tìm thấy trong file.
type ParseCron struct {
	CronExpr string // biểu thức cron (ví dụ: "0 * * * * *")
	Handler  string // tên hàm cần thực thi
}

// ParseCronFromFile tìm tất cả các hàm có chú thích @Cron trong file Go.
//
// Cách sử dụng nội bộ: được gọi trong ParseSourceForCronJobs()
func ParseCronFromFile(filename string) []ParseCron {
	set := token.NewFileSet()
	node, err := parser.ParseFile(set, filename, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse file: %v", err)
	}

	var cronEntries []ParseCron
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		for _, comment := range fn.Doc.List {
			if strings.HasPrefix(comment.Text, "// @Cron") {
				cronExpr := strings.TrimSpace(strings.TrimPrefix(comment.Text, "// @Cron"))
				if !isValidCronExpr(cronExpr) {
					cronExpr = viper.GetString(cronExpr)
					if !isValidCronExpr(cronExpr) {
						log.Printf("biểu thức cron không hợp lệ: %s", cronExpr)
					}
				}
				cronEntries = append(cronEntries, ParseCron{
					CronExpr: cronExpr,
					Handler:  fn.Name.Name,
				})
			}
		}
	}
	return cronEntries
}

// isValidCronExpr kiểm tra tính hợp lệ của biểu thức cron.
//
// Cách sử dụng nội bộ: dùng trong AddJob().
func isValidCronExpr(expr string) bool {
	_, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(expr)
	return err == nil
}
