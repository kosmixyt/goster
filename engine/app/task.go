package engine

import (
	"time"

	"gorm.io/gorm"
)

var Tasks = make([]*Task, 0)

type Task struct {
	gorm.Model
	Name string `gorm:"not null" json:"name"`
	// long
	Logs            string     `gorm:"type:longtext" json:"logs"`
	last_saved_logs *time.Time `json:"-" gorm:"-"`
	Status          string     `gorm:"not null" json:"status"`
	Started         *time.Time `gorm:"null" json:"started"`

	Finished *time.Time `gorm:"null" json:"finished"`
	User     *User      `gorm:"foreignKey:USER_ID" json:"user"`
	// Error     string       `json:"error"`
	USER_ID         uint             `gorm:"not null" json:"user_id"`
	on_Cancel       func() error     `json:"-" gorm:"-"`
	listen_progress [](func(string)) `json:"-" gorm:"-"`
}

func (t *Task) SetAsStarted() {
	if t.Status != "PENDING" {
		panic("Task is not pending")
	}
	t.Status = "RUNNING"
	start := time.Now()
	t.Started = &start
	t.Logs += "Task started at " + start.String() + "\n"
	db.Save(t)
	t.last_saved_logs = &start
}
func (t *Task) Cancel() error {
	if t.Status != "RUNNING" {
		panic("Task is not running (can only cancel running tasks)")
	}
	if err := t.on_Cancel(); err != nil {
		return err
	}
	t.Status = "CANCELLED"
	t.AddLog("Task cancelled\n")
	endTime := time.Now()
	t.Finished = &endTime
	db.Save(t)
	return nil
}
func (t *Task) SetAsFinished() {
	if t.Status != "RUNNING" {
		panic("Task is not running")
	}
	t.Status = "FINISHED"
	finish := time.Now()
	t.Finished = &finish
	t.AddLog("Task finished at " + finish.String() + "\n")
	db.Save(t)
}
func (t *Task) AddLog(log ...string) {
	m := ""
	for _, l := range log {
		m += l + " "
	}
	t.Logs += m + "\n"
	for _, f := range t.listen_progress {
		f(m)
	}
	if t.last_saved_logs == nil || time.Since(*t.last_saved_logs) > 5*time.Second {
		// db.Save(t)
		db.Model(t).Update("Logs", gorm.Expr("CONCAT(logs, ?)", m))
		t.last_saved_logs = new(time.Time)
		*t.last_saved_logs = time.Now()
	}
}
func (t *Task) GetLogs() string {
	return t.Logs
}
func (t *Task) UpdateName(Addname string) {
	t.Name += Addname
	db.Save(t)
}

func (t *Task) SetAsError(err interface{}) interface{} {
	var message string
	s, ok := err.(string)
	if ok {
		message = s
	} else {
		message = err.(error).Error()
	}
	if t.Status != "RUNNING" && t.Status != "PENDING" {
		// t.Logs += "Task Init Error: " + message + "\n"
		panic("Task is not running or pending")
	}
	t.Status = "ERROR"
	t.AddLog("Error: " + message + "\n")
	endTime := time.Now()
	t.Finished = &endTime
	db.Save(t)
	return err
}
func (t *Task) ListenProgress(f func(string)) {
	t.listen_progress = append(t.listen_progress, f)
}
func GetRuntimeTask(id uint) *Task {
	for _, t := range Tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// task cycle
// PENDING -> RUNNING -> FINISHED
// PENDING -> RUNNING -> ERROR
// PENDING -> ERROR
// PENDING -> FINISHED
// RUNNING -> ERROR
// RUNNING -> FINISHED
