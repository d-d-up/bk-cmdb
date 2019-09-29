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
package x19_09_06_01

import (
	"context"

	"configcenter/src/common/blog"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

func init() {
	upgrader.RegistUpgrader("x19_09_06_01", upgrade)
}

func upgrade(ctx context.Context, db dal.RDB, conf *upgrader.Config) (err error) {
	err = createSetTemplateTables(ctx, db, conf)
	if err != nil {
		blog.Errorf("[upgrade x19.09.06.01] create tables failed, err: %s", err.Error())
		return err
	}

	err = addSetProperty(ctx, db, conf)
	if err != nil {
		blog.Errorf("[upgrade x19.09.06.01] addSetProperty failed, err: %s", err.Error())
		return err
	}

	err = addModuleProperty(ctx, db, conf)
	if err != nil {
		blog.Errorf("[upgrade x19.09.06.01] addModuleProperty failed, err: %s", err.Error())
		return err
	}

	err = initializeSetTemplateField(ctx, db, conf)
	if err != nil {
		blog.Errorf("[upgrade x19.09.06.01] initializeSetTemplateField failed, err: %s", err.Error())
		return err
	}

	return nil
}
