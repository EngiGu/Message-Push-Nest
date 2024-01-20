package models

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"message-nest/pkg/util"
)

type SendTasks struct {
	UUIDModel

	Name string `json:"name" gorm:"type:varchar(100) comment '任务名称';default:'';"`
}

func GenerateTaskUniqueID() string {
	newUUID := util.GenerateUniqueID()
	return fmt.Sprintf("T-%s", newUUID)
}

// AddSendTaskWithID 添加实例的时候添加任务
func AddSendTaskWithID(name string, id string, createdBy string) error {
	err := db.Where("id = ?", id).Find(&SendTasks{}).Error
	if err == nil {
		return nil
	}
	task := SendTasks{
		UUIDModel: UUIDModel{
			ID:         id,
			CreatedBy:  createdBy,
			ModifiedBy: createdBy,
		},
		Name: name,
	}
	if err := db.Create(&task).Error; err != nil {
		return err
	}
	return nil
}

// AddSendTask 添加任务
func AddSendTask(name string, createdBy string) error {
	newUUID := GenerateTaskUniqueID()
	task := SendTasks{
		UUIDModel: UUIDModel{
			ID:         newUUID,
			CreatedBy:  createdBy,
			ModifiedBy: createdBy,
		},
		Name: name,
	}
	if err := db.Create(&task).Error; err != nil {
		return err
	}
	return nil
}

// GetSendTasks 获取所有任务
func GetSendTasks(pageNum int, pageSize int, name string, maps interface{}) ([]SendTasks, error) {
	var (
		tasks []SendTasks
		err   error
	)
	query := db.Where(maps)
	if name != "" {
		query = query.Where("name like ?", fmt.Sprintf("%%%s%%", name))
	}
	query = query.Order("created_on DESC")
	if pageSize > 0 || pageNum > 0 {
		query = query.Offset(pageNum).Limit(pageSize)
	}
	err = query.Find(&tasks).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return tasks, nil
}

// GetSendTasksTotal 获取所有任务总数
func GetSendTasksTotal(name string, maps interface{}) (int, error) {
	var (
		err   error
		total int
	)
	query := db.Model(&SendTasks{}).Where(maps)
	if name != "" {
		query = query.Where("name like ?", fmt.Sprintf("%%%s%%", name))
	}

	err = query.Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

type SendTasksInsRes struct {
	SendTasksIns

	WayName string `json:"way_name"`
}

type TaskIns struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	InsData []SendTasksInsRes `json:"ins_data"`
}

// GetTasksIns 获取所有任务下所有的实例
func GetTasksIns(id string) (TaskIns, error) {
	insTable := db.NewScope(SendTasksIns{}).TableName()
	waysTable := db.NewScope(SendWays{}).TableName()
	var (
		task       SendTasks
		taskIns    []SendTasksInsRes
		taskResult TaskIns
	)
	err := db.Where("id = ?", id).First(&task).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return taskResult, err
	}

	db.
		Table(insTable).
		Select(fmt.Sprintf("%s.*, %s.name as way_name", insTable, waysTable)).
		Joins(fmt.Sprintf("JOIN %s ON %s.way_id = %s.id", waysTable, insTable, waysTable)).
		Where(fmt.Sprintf("%s.task_id = ?", insTable), id).
		Order(fmt.Sprintf("%s.created_on DESC", insTable)).
		Scan(&taskIns)

	taskResult.ID = task.ID
	taskResult.Name = task.Name
	taskResult.InsData = taskIns
	return taskResult, nil
}

// FindTaskByWayId 通过way_id找到关联的任务
func FindTaskByWayId(wayId string) []SendTasks {
	insTable := db.NewScope(SendTasksIns{}).TableName()
	taskTable := db.NewScope(SendTasks{}).TableName()
	var (
		tasks []SendTasks
	)

	db.
		Table(taskTable).
		Select(fmt.Sprintf("%s.*", taskTable)).
		Joins(fmt.Sprintf("JOIN %s ON %s.task_id = %s.id", insTable, insTable, taskTable)).
		Where(fmt.Sprintf("%s.way_id = ?", insTable), wayId).
		Scan(&tasks)

	return tasks
}

// 删除任务并删除所有关联的实例
func DeleteMsgTask(id string) error {
	tx := db.Begin()
	if err := db.Where("id = ?", id).Delete(&SendTasks{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := db.Where("task_id = ?", id).Delete(&SendTasksIns{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func EditSendTask(id string, data interface{}) error {
	if err := db.Model(&SendTasks{}).Where("id = ? ", id).Updates(data).Error; err != nil {
		return err
	}

	return nil
}
