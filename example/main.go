package main

import (
	"fmt"
	"github.com/kimxuanhong/go-cron/cron"
	"github.com/kimxuanhong/go-utils/config"
	"os"
)

type MyHandler struct{}

func (m MyHandler) SayHello() {
	println("Hello, this runs on a schedule!")
}

// Every30Sec
// Giây     Phút     Giờ     Ngày     Tháng     Thứ
// */30      *        *       *         *        *
// @Cron */30 * * * * *
func (m MyHandler) Every30Sec() error {
	println("Every 30 sec task")
	return nil
}

// Every30SecSayHello
// Giây     Phút     Giờ     Ngày     Tháng     Thứ
// */30      *        *       *         *        *
// @Cron cron.Every30SecSayHello
func (m MyHandler) Every30SecSayHello() error {
	println("Every 30 sec Every30SecSayHello")
	return nil
}

func main() {
	dir, _ := os.Getwd()
	fmt.Println("📂 Working Directory:", dir)
	config.LoadConfigFile()
	c := cron.NewCronJob()
	c.SetDirs(dir)
	c.RegisterHandlers(MyHandler{}) // Đăng ký handler chứa các hàm
	c.ParseSourceForCronJobs()
	_ = c.Start()

	select {} // block mãi mãi
}
