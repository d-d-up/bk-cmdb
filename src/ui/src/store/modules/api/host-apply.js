/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

import $http from '@/api'

const state = {
    propertyConfig: {},
    ruleDraft: {}
}

const getters = {

}

const actions = {
    getRules ({ commit, state, dispatch }, { bizId, params, config }) {
        return $http.post(`findmany/host_apply_rule/bk_biz_id/${bizId}`, params, config)
    },
    getApplyPreview ({ commit, state, dispatch }, { bizId, params, config }) {
        return $http.post(`createmany/host_apply_plan/bk_biz_id/${bizId}/preview`, params, config)
    },
    runApply ({ commit, state, dispatch }, { bizId, params, config }) {
        return $http.post(`updatemany/host_apply_plan/bk_biz_id/${bizId}/run`, params, config)
    },
    getTopopath ({ commit, state, dispatch }, { bizId, params, config }) {
        return $http.post(`find/topopath/biz/${bizId}`, params, config)
    },
    setEnableStatus ({ commit, state, dispatch }, { bizId, moduleId, params, config }) {
        return $http.put(`module/host_apply_enable_status/bk_biz_id/${bizId}/bk_module_id/${moduleId}`, params, config)
    },
    deleteRules ({ commit, state, dispatch }, { bizId, params }) {
        return $http.delete(`deletemany/host_apply_rule/bk_biz_id/${bizId}`, params)
    },
    getProperties (context, config) {
        return $http.post('find/objectattr/host', {}, config)
    }
}

const mutations = {
    setPropertyConfig (state, config) {
        state.propertyConfig = config
    },
    setRuleDraft (state, draft) {
        state.ruleDraft = draft
    },
    clearRuleDraft (state) {
        state.ruleDraft = {}
    }
}

export default {
    namespaced: true,
    state,
    getters,
    actions,
    mutations
}
