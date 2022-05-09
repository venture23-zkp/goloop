/*
 * Copyright 2022 ICON Foundation
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

package btp

import "github.com/icon-project/goloop/common/codec"

type NetworkType struct {
	UID                  string
	NextProofContextHash []byte
	NextProofContext     []byte
	OpenNetworkIDs       []int64
}

func (nt *NetworkType) Bytes() []byte {
	return codec.MustMarshalToBytes(nt)
}

type Network struct {
	NetworkTypeID           int64
	Open                    bool
	NextMessageSN           int64
	NextProofContextChanged bool
	PrevNetworkSectionHash  []byte
	LastNetworkSectionHash  []byte
}

func (nw *Network) Bytes() []byte {
	return codec.MustMarshalToBytes(nw)
}
