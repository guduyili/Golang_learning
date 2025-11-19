package session

import (
	"orm/log"
	"reflect"
)

const (
	BeforeQuery  = "BeforeQuery"
	AfterQuery   = "AfterQuery"
	BeforeInsert = "BeforeInsert"
	AfterInsert  = "AfterInsert"
	BeforeUpdate = "BeforeUpdate"
	AfterUpdate  = "AfterUpdate"
	BeforeDelete = "BeforeDelete"
	AfterDelete  = "AfterDelete"
)

// CallMethod 调用注册在模型或当前传入对象上的 Hook 方法（通过反射）
func (s *Session) CallMethod(method string, value interface{}) {
	// 在test中  s.RefTable().Model 返回 指针 *Account{}
	//MethodByName 返回与给定名称的方法对应的函数值。
	fm := reflect.ValueOf(s.RefTable().Model).MethodByName(method)
	if value != nil {
		fm = reflect.ValueOf(value).MethodByName(method)
	}

	// Hook 约定只接收一个参数（*Session）。接收者（*Account）由反射自动绑定。
	param := []reflect.Value{reflect.ValueOf(s)}

	//确认方法存在才调用。
	if fm.IsValid() {
		//fm.Call(param) 执行：隐式把接收者设为 value（或 s.RefTable().Model），显式参数是 s。
		//`Call` 函数会调用函数 `fm`，并将输入参数 `param` 传递给它。
		if v := fm.Call(param); len(v) > 1 {
			if err, ok := v[1].Interface().(error); ok {
				log.Error(err)
			}
		}
	}
	return
}
