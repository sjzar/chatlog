/*
 * Copyright (c) 2023 shenjunzheng@gmail.com
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"reflect"

	"github.com/spf13/viper"
)

func SetDefaults(v *viper.Viper, c any, defaults map[string]any) {
	keys := GetStructKeys(reflect.TypeOf(c), "mapstructure", "squash")
	for _, key := range keys {
		v.SetDefault(key, nil)
	}
	for key := range defaults {
		v.SetDefault(key, defaults[key])
	}
}
