package main

import (
	"github.com/kimxuanhong/go-cron/cron"
	"github.com/kimxuanhong/go-cron/example/jobs"
	"github.com/kimxuanhong/go-utils/config"
)

func main() {
	config.LoadConfigFile()
	c := cron.NewCronJob()
	c.RegisterJobs(&jobs.SayHelloJob{}) // Đăng ký handler chứa các hàm
	c.RegisterJobWithTags(&jobs.SayHelloJob{})
	_ = c.Start()

	select {} // block mãi mãi
}
