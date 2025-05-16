package cron

// Scheduler là interface bọc các phương thức của cron để quản lý cron jobs.
type Scheduler interface {
	AddJob(cronExpr string, jobFunc func()) error
	RegisterJobs(handlers ...Job)
	RegisterJobWithTags(jobs ...interface{})
	Start() error
	Stop()
}

type Job interface {
	CronExpr() string
	Run()
}
