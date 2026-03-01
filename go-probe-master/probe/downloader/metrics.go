// Copyright 2015 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/probechain/go-probe/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("probe/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("probe/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("probe/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("probe/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("probe/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("probe/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("probe/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("probe/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("probe/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("probe/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("probe/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("probe/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("probe/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("probe/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("probe/downloader/throttle", nil)
)
