/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package auth

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/util"
	"configcenter/src/web_server/middleware/types"

	"github.com/gin-gonic/gin"
	"github.com/holmeswang/contrib/sessions"
)

var (
	createObjectInstanceBizRegexp   = regexp.MustCompile(`.*/create/instance/object/[^\s/]+/?$`)
	createObjectInstanceRegexp      = regexp.MustCompile(`.*/inst/[^\s/]+/[^\s/]+/?$`)
	deleteObjectInstanceRegexp      = regexp.MustCompile(`.*/inst/[^\s/]+/[^\s/]+/[0-9]+/?$`)
	deleteObjectInstanceBizRegexp   = regexp.MustCompile(`.*/delete/instance/object/[^\s/]+/inst/[0-9]+/?$`)
	deleteObjectInstanceBatchRegexp = regexp.MustCompile(`.*/inst/[^\s/]+/[^\s/]+/batch/?$`)
	updateObjectInstanceRegexp      = regexp.MustCompile(`.*/inst/[^\s/]+/[^\s/]+/[0-9]+/?$`)
	updateObjectInstanceBizRegexp   = regexp.MustCompile(`.*/update/instance/object/[^\s/]+/inst/[0-9]+/?$`)
	updateObjectInstanceBatchRegexp = regexp.MustCompile(`.*/updatemany/instance/object/[^\s/]+/?$`)
	searchObjectInstanceRegexp      = regexp.MustCompile(`.*/inst/search/[^\s/]+/[^\s/]+/?$`)
	searchObjectInstAndAssoRegexp   = regexp.MustCompile(`.*/inst/search/owner/[^\s/]+/object/[^\s/]+/detail/?$`)
	instSearchRegexp                = regexp.MustCompile(`.*/inst/search/owner/[^\s/]+/object/[^\s/]+/?$`)
	getInstRegexp                   = regexp.MustCompile(`.*/inst/search/owner/[^\s/]+/object/[^\s/]+/[0-9]+/?$`)
	exportObjectInstanceRegexp      = regexp.MustCompile(`/insts/owner/[^\s/]+/object/[^\s/]+/export/?$`)
	importObjectInstanceRegexp      = regexp.MustCompile(`/insts/owner/[^\s/]+/object/[^\s/]+/import/?$`)
)

// validModelConfigPrivi valid model inst privilege
func validModelConfigPrivi(modelPrivi string, method string, pathArr []string) bool {

	var mPrivi map[string][]string
	var objName string
	err := json.Unmarshal([]byte(modelPrivi), &mPrivi)
	if nil != err {
		blog.Error("get model privilege json error")
		return false
	}

	pathStr := strings.Join(pathArr, "/")
	switch {
	case createObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-1]

	case createObjectInstanceBizRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-1]

	case deleteObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodDelete:
		objName = pathArr[len(pathArr)-2]

	case updateObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodPut:
		objName = pathArr[len(pathArr)-2]

	case updateObjectInstanceBizRegexp.MatchString(pathStr) && method == http.MethodPut:
		objName = pathArr[len(pathArr)-3]

	case updateObjectInstanceBatchRegexp.MatchString(pathStr) && method == http.MethodPut:
		objName = pathArr[len(pathArr)-1]

	case searchObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-1]

	case searchObjectInstAndAssoRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-2]

	case instSearchRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-1]

	case getInstRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-2]

	case importObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-2]

	case exportObjectInstanceRegexp.MatchString(pathStr) && method == http.MethodPost:
		objName = pathArr[len(pathArr)-2]

	case deleteObjectInstanceBatchRegexp.MatchString(pathStr) && method == http.MethodDelete:
		objName = pathArr[len(pathArr)-2]

	case deleteObjectInstanceBizRegexp.MatchString(pathStr) && method == http.MethodDelete:
		objName = pathArr[len(pathArr)-3]

	}

	priviArr, ok := mPrivi[objName]
	if false == ok {
		blog.Error("get object privilege for %s error", objName)
		return false
	}

	// merge update&&create privilege
	if method == common.HTTPUpdate || method == common.HTTPCreate {
		if util.InArray(types.BK_CC_UPDATE, priviArr) {
			return true
		}
		if util.InArray(types.BK_CC_CREATE, priviArr) && !util.InArray(types.BK_CC_SEARCH, pathArr) {
			return true
		}

	}

	// valid delete privilege
	if method == common.HTTPDelete && util.InArray(types.BK_CC_DELETE, priviArr) {
		return true
	}

	// valid search privilege
	if method == common.HTTPSelectPost && util.InArray(types.BK_CC_SEARCH, priviArr) && util.InArray(types.BK_CC_SEARCH, pathArr) {
		return true
	}
	blog.Error("model privilege valid not pass")
	return false
}

// validAppConfigPrivi valid app privilege
func validAppConfigPrivi(c *gin.Context, method, pathStr string) bool {

	//validate host update privilege
	if strings.Contains(pathStr, types.BK_HOST_UPDATE) && method == common.HTTPUpdate {
		return validAppAccessPrivi(c, types.BK_CC_HOSTUPDATE)
	}

	//validate host trans privilege
	if strings.Contains(pathStr, types.BK_HOST_TRANS) {
		return validAppAccessPrivi(c, types.BK_CC_HOSTTRANS)
	}

	//validate topo update privilege
	if strings.Contains(pathStr, types.BK_SET) || strings.Contains(pathStr, types.BK_MODULE) || strings.Contains(pathStr, types.BK_INSTS) || strings.Contains(pathStr, types.BK_TOPO) {
		return validAppAccessPrivi(c, types.BK_CC_TOPOUPDATE)
	}

	//validate user customer api privilege
	if strings.Contains(pathStr, types.BK_USER_API_S) {
		return validAppAccessPrivi(c, types.BK_CC_CUSTOMAPI)
	}
	//validate process config privilege
	if strings.Contains(pathStr, types.BK_PROC_S) {
		return validAppAccessPrivi(c, types.BK_CC_PROCCONFIG)
	}

	return true
}

// validate app access privilege
func validAppAccessPrivi(c *gin.Context, appResource string) bool {
	session := sessions.Default(c)
	appID := c.Request.Header.Get(common.BKAppIDField)
	if "" == appID {
		blog.Error("no app id in header")
		return false
	}
	userPriviAppStr, ok := session.Get("userPriviApp").(string)
	if false == ok {
		blog.Error("get user privilege from session error")
		return false
	}

	rolePrivilege, ok := session.Get("rolePrivilege").(string)
	if false == ok {
		blog.Error("get role privilege from session error")
		return false
	}

	//valid opearion under biz
	var userPriviApp, rolePrivi map[string][]string
	err := json.Unmarshal([]byte(userPriviAppStr), &userPriviApp)
	if nil != err {
		blog.Error("user privi app json error")
		return false
	}
	appRole, ok := userPriviApp[appID]
	if false == ok {
		blog.Error("no user privi app ")
		return false
	}
	//maintainer role pass the valid
	if util.InArray(types.BK_CC_MAINTAINERS, appRole) {
		return true
	}

	err = json.Unmarshal([]byte(rolePrivilege), &rolePrivi)
	if nil != err {
		blog.Error("role privi json error ")
		return false
	}
	priviArr := make([]string, 0)
	for _, role := range appRole {
		privi := rolePrivi[role]
		for _, p := range privi {
			priviArr = append(priviArr, p)
		}
	}
	if util.Contains(priviArr, appResource) {
		return true
	}
	blog.Error("valid user app privilege false")
	return false
}

// validSysConfigPrivi valid system access privilege
func validSysConfigPrivi(sysPrivi interface{}, config string) bool {
	if nil != sysPrivi {
		ssysPrivi := sysPrivi.(string)
		if 0 == len(ssysPrivi) {
			blog.Error("no system config privilege")
			return false
		}
		var sysPriObj []string
		err := json.Unmarshal([]byte(ssysPrivi), &sysPriObj)
		if nil != err {
			blog.Error("no system config privilege not json")
			return false
		}
		if util.InArray(config, sysPriObj) {
			return true
		}
	}
	blog.Error("system privilege not pass")
	return false
}

// validInstsOpPrivi  valid inst operation privilege
func validInstsOpPrivi(modelPrivi, method string, pathArr []string) bool {
	var mPrivi map[string][]string
	var objName string
	err := json.Unmarshal([]byte(modelPrivi), &mPrivi)
	if nil != err {
		blog.Error("get model privilege json error")
		return false
	}
	objName = pathArr[len(pathArr)-2]
	priviArr, ok := mPrivi[objName]
	if false == ok {
		blog.Error("get object privilege  error")
		return false
	}
	if util.InArray(types.BK_CC_UPDATE, priviArr) {
		return true
	}
	if util.InArray(types.BK_CC_CREATE, priviArr) && !util.InArray(types.BK_CC_SEARCH, pathArr) {
		return true
	}

	blog.Error("inst op privilege valid not pass")
	return false
}
