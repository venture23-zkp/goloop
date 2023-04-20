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

 package foundation.icon.ee.util;

 import foundation.icon.ee.util.bn128.BN128G1;
 import foundation.icon.ee.util.bn128.BN128G2;
 import foundation.icon.ee.util.bn128.BN128CurveOps;

import org.aion.avm.core.util.Helpers;
import org.junit.Test;
 import org.junit.jupiter.api.Assertions;
 import java.math.BigInteger;
import java.util.Arrays;
 
 
 public class BN128Test {
 
     private static byte[] concatBytes(byte[]... args) {
         int length = 0;
         for (int i = 0; i < args.length; i++) {
             length += args[i].length;
         }
         byte[] out = new byte[length];
         int offset = 0;
         for (int i = 0; i < args.length; i++) {
             System.arraycopy(args[i], 0, out, offset, args[i].length);
             offset += args[i].length;
         }
         return out;
     }
 
     @Test
     public void pairingCheck() {
         BN128G1 g1 = BN128G1.generator();
         BN128G2 g2 = BN128G2.generator();
 
         Assertions.assertTrue(BN128CurveOps.pairingCheck(
             concatBytes(
                 g1.bytes(), g2.bytes(),
                 g1.negate().bytes(), g2.bytes(),
                 g1.bytes(), g2.bytes(),
                 g1.bytes(), g2.negate().bytes()
             ), false));
     }
 
     @Test
     public void addAndScalarMul() {
         byte[] out;
 
         // g1 add and scalarMul tests
         BN128G1 g1 = BN128G1.generator();
 
         byte[] g1x2b = BN128CurveOps.g1ScalarMul(new BigInteger("2").toByteArray(), g1.bytes(), false);
         byte[] g1x3b = BN128CurveOps.g1ScalarMul(new BigInteger("3").toByteArray(), g1.bytes(), false);
 
         out = BN128CurveOps.g1Add(concatBytes(g1.bytes(), g1x2b), false);

         Assertions.assertArrayEquals(out, g1x3b, "should be equal");
 
         // g2 add and scalarMul tests
         BN128G2 g2 = BN128G2.generator();
 
         byte[] g2x2b = BN128CurveOps.g2ScalarMul(new BigInteger("2").toByteArray(), g2.bytes(), false);
         byte[] g2x3b = BN128CurveOps.g2ScalarMul(new BigInteger("3").toByteArray(), g2.bytes(), false);
 
         out = BN128CurveOps.g2Add(concatBytes(g2.bytes(), g2x2b), false);

         Assertions.assertArrayEquals(out, g2x3b, "should be equal");
     }
 
 }
 