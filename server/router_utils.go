// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/rs/zerolog/log"

	sptypes "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	// OSDEligibleParam parameter
	OSDEligibleParam = "osd_eligible"
	// GetDisabledParam parameter
	GetDisabledParam = "get_disabled"
	// ImpactingParam parameter used to show/hide recommendations not hitting any clusters
	ImpactingParam = "impacting"
)

func readRuleIDWithErrorKey(writer http.ResponseWriter, request *http.Request) (types.RuleID, types.ErrorKey, error) {
	ruleIDWithErrorKey, err := httputils.GetRouterParam(request, "rule_id")
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return types.RuleID(""), types.ErrorKey(""), err
	}

	splitedRuleID := strings.Split(string(ruleIDWithErrorKey), "|")

	if len(splitedRuleID) != 2 {
		err = fmt.Errorf("invalid rule ID, it must contain only rule ID and error key separated by |")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			paramName:  "rule_id",
			paramValue: ruleIDWithErrorKey,
			errString:  err.Error(),
		})
		return types.RuleID(""), types.ErrorKey(""), err
	}

	IDValidator := regexp.MustCompile(`^[a-zA-Z_0-9.]+$`)

	isRuleIDValid := IDValidator.MatchString(splitedRuleID[0])
	isErrorKeyValid := IDValidator.MatchString(splitedRuleID[1])

	if !isRuleIDValid || !isErrorKeyValid {
		err = fmt.Errorf("invalid rule ID, each part of ID must contain only latin characters, number, underscores or dots")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			paramName:  "rule_id",
			paramValue: ruleIDWithErrorKey,
			errString:  err.Error(),
		})
		return types.RuleID(""), types.ErrorKey(""), err
	}

	return types.RuleID(splitedRuleID[0]), types.ErrorKey(splitedRuleID[1]), nil
}

func (server HTTPServer) readParamsGetRecommendations(writer http.ResponseWriter, request *http.Request) (
	userID types.UserID,
	orgID types.OrgID,
	impactingFlag sptypes.ImpactingFlag,
	err error,
) {

	orgID, userID, err = server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Err(err).Msg("error retrieving orgID and userID from auth token")
		return
	}

	impactingParam := request.URL.Query().Get(ImpactingParam)
	if len(impactingParam) == 0 {
		// impacting control flag is missing, display all recommendations
		impactingFlag = IncludingImpacting
		return
	}

	impactingParamBool, err := readImpactingParam(request)
	if err != nil {
		log.Err(err).Msgf("Error parsing `%s` URL parameter.", ImpactingParam)
		handleServerError(writer, &RouterParsingError{
			paramName: ImpactingParam,
			errString: "Unparsable boolean value",
		})
		return
	}

	if impactingParamBool {
		// param impacting=true means to only include impacting recommendations
		impactingFlag = OnlyImpacting
	} else {
		// param impacting=false means to return all rules that aren't impacting any clusters
		impactingFlag = ExcludingImpacting
	}

	return
}

// readQueryParam return the value of the parameter in the query. If not found, defaults to false
func readQueryBoolParam(name string, defaultValue bool, request *http.Request) (bool, error) {
	value := request.URL.Query().Get(name)
	if len(value) == 0 {
		return defaultValue, nil
	}
	return strconv.ParseBool(value)
}

// readGetDisabledParam returns the value of the "get_disabled" parameter in query
// if available
func readGetDisabledParam(request *http.Request) (bool, error) {
	return readQueryBoolParam(GetDisabledParam, false, request)
}

// readOSDEligibleParam returns the value of the "osd_eligible" parameter in query
// if available
func readOSDEligible(request *http.Request) (bool, error) {
	return readQueryBoolParam(OSDEligibleParam, false, request)
}

// readImpactingParam returns the value of the "impacting" parameter in query if available
func readImpactingParam(request *http.Request) (bool, error) {
	return readQueryBoolParam(ImpactingParam, true, request)
}
