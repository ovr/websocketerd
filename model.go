package main

import "strconv"

type User struct {
	Id             uint64 `gorm:"primary_key"`
	ThreadsShardNo uint64 `gorm:"column:threads_shard_no"`
}

func (User) TableName() string {
	return "users"
}

type Thread struct {
	threadSharedId uint64
}

func (this Thread) TableName() string {
	return "users" + strconv.FormatUint(this.threadSharedId, 64)
}
