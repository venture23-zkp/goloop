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

 package testcases;

 import score.Context;
 import score.annotation.External;
 
 import java.math.BigInteger;
 import java.util.Arrays;
 
 public class BN128TestScore {
     static private byte[] hexToBytes(String s) {
         int len = s.length();
         byte[] data = new byte[len / 2];
         for (int i = 0; i < len; i += 2) {
             data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                     + Character.digit(s.charAt(i + 1), 16));
         }
         return data;
     }

     @External
     public void test() { 
         testBN128ecAddG1();
         testBN128ecScalarMulG1();
         testBN128ecAddG2();
         testBN128ecScalarMulG2();
         testBN128ecPairingCheck();
         testBN128InvalidDataEncoding();
     }
 
     public void testBN128ecAddG1() {
         byte[] g1b = hexToBytes("00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002");
         byte[] g1x2b = hexToBytes("030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd315ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4");
         byte[] g1x3b = hexToBytes("0769bf9ac56bea3ff40232bcb1b6bd159315d84715b8e679f2d355961915abf02ab799bee0489429554fdb7c8d086475319e63b40b9c5b57cdf1ff3dd9fe2261");
         byte[] g1x6b = hexToBytes("09f4ca411a3f52f4e0792fd9e792779856719215d3b32a762afe3d5b8c684af90d8ef3d795acd4b35d4366ab22e4ad335273aa59429e26929d0f64583474d9c8");
 
         byte[] out = Context.ecAdd("bn128-g1", concatBytes(g1b, g1x2b, g1x3b), false);
         Context.require(Arrays.equals(g1x6b, out), "incorrect ecAddG1 result");
 
         Context.println("testBN128ecAddG1 - OK");
     }
 
     public void testBN128ecScalarMulG1() {
         byte[] g1b = hexToBytes("00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002");
         byte[] g1x2b = hexToBytes("030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd315ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4");
         byte[] g1x6b = hexToBytes("09f4ca411a3f52f4e0792fd9e792779856719215d3b32a762afe3d5b8c684af90d8ef3d795acd4b35d4366ab22e4ad335273aa59429e26929d0f64583474d9c8");
 
         byte[] out;
 
         out = Context.ecScalarMul("bn128-g1", new BigInteger("2").toByteArray(), g1b, false);
         Context.require(Arrays.equals(g1x2b, out), "incorrect ecAdd result");
 
         out = Context.ecScalarMul("bn128-g1", new BigInteger("3").toByteArray(), g1x2b, false);
         Context.require(Arrays.equals(g1x6b, out), "incorrect ecScalarMulG1 result");
 
         Context.println("testBN128ecScalarMulG1 - OK");
     }
 
     public void testBN128ecAddG2() {
         byte[] g2b = hexToBytes("198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa");
         byte[] g2x2b = hexToBytes("203e205db4f19b37b60121b83a7333706db86431c6d835849957ed8c3928ad7927dc7234fd11d3e8c36c59277c3e6f149d5cd3cfa9a62aee49f8130962b4b3b9195e8aa5b7827463722b8c153931579d3505566b4edf48d498e185f0509de15204bb53b8977e5f92a0bc372742c4830944a59b4fe6b1c0466e2a6dad122b5d2e");
         byte[] g2x3b = hexToBytes("1014772f57bb9742735191cd5dcfe4ebbc04156b6878a0a7c9824f32ffb66e8506064e784db10e9051e52826e192715e8d7e478cb09a5e0012defa0694fbc7f5021e2335f3354bb7922ffcc2f38d3323dd9453ac49b55441452aeaca147711b2058e1d5681b5b9e0074b0f9c8d2c68a069b920d74521e79765036d57666c5597");
         byte[] g2x6b = hexToBytes("1b4b60273ae700a7e2ffc04e19e316074a5977c8da56b75675927e2eee23772e1687f985433b446b85eb6d0a574fc152f681c032d27e6207569faca9c8329b961e7cf2fd8b4bc0d81e4719f009a5ecb7d925c970bc57889f3627d86629dc31d824fb6baf4cf6d7ca7eaa668cda36d088502b3587667b6eb8f2b874622575e586");
 
         byte[] out = Context.ecAdd("bn128-g2", concatBytes(g2b, g2x2b, g2x3b), false);
         Context.require(Arrays.equals(g2x6b, out), "incorrect ecAdd.G2 result");
 
         Context.println("testBN128ecAddG2 - OK");
     }
 
     public void testBN128ecScalarMulG2() {
         byte[] g2b = hexToBytes("198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa");
         byte[] g2x2b = hexToBytes("203e205db4f19b37b60121b83a7333706db86431c6d835849957ed8c3928ad7927dc7234fd11d3e8c36c59277c3e6f149d5cd3cfa9a62aee49f8130962b4b3b9195e8aa5b7827463722b8c153931579d3505566b4edf48d498e185f0509de15204bb53b8977e5f92a0bc372742c4830944a59b4fe6b1c0466e2a6dad122b5d2e");
         byte[] g2x6b = hexToBytes("1b4b60273ae700a7e2ffc04e19e316074a5977c8da56b75675927e2eee23772e1687f985433b446b85eb6d0a574fc152f681c032d27e6207569faca9c8329b961e7cf2fd8b4bc0d81e4719f009a5ecb7d925c970bc57889f3627d86629dc31d824fb6baf4cf6d7ca7eaa668cda36d088502b3587667b6eb8f2b874622575e586");
 
         byte[] out;
 
         out = Context.ecScalarMul("bn128-g2", new BigInteger("2").toByteArray(), g2b, false);
         Context.require(Arrays.equals(g2x2b, out), "incorrect ecAdd result");
 
         out = Context.ecScalarMul("bn128-g2", new BigInteger("3").toByteArray(), g2x2b, false);
         Context.require(Arrays.equals(g2x6b, out), "incorrect ecAdd result");
 
         Context.println("testBN128ecScalarMulG2 - OK");
     }
 
     public void testBN128ecPairingCheck() {
         byte[] g1b = hexToBytes("00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002");
         byte[] g1Negb = hexToBytes("000000000000000000000000000000000000000000000000000000000000000130644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd45");
         byte[] g2b = hexToBytes("198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa");
         byte[] g2Negb = hexToBytes("198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed275dc4a288d1afb3cbb1ac09187524c7db36395df7be3b99e673b13a075a65ec1d9befcd05a5323e6da4d435f3b617cdb3af83285c2df711ef39c01571827f9d");
 
         boolean res = Context.ecPairingCheck("bn128", concatBytes(
             g1b, g2b,
             g1Negb, g2b,
             g1b, g2b,
             g1b, g2Negb
         ), false);
         Context.require(res, "incorrect ecPairingCheck result");
 
         Context.println("testBN128ecPairingCheck - OK");
     }
 
     public void testBN128InvalidDataEncoding() {
         byte[] g1b = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e0");
         byte[] g2b = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b828010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be");
 
         try {
             Context.ecAdd("bn128-g1", concatBytes(g1b, g1b), false);
             Context.require(false, "ecAddG1: should not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecAddG1)");
         }
 
         try {
             Context.ecAdd("bn128-g1", concatBytes(g1b, g1b), true);
             Context.require(false, "ecAddG1Compressed: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecAddG1Compressed)");
         }
 
         try {
             Context.ecAdd("bn128-g2", concatBytes(g1b, g1b), false);
             Context.require(false, "ecAddG2: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecAddG2)");
         }
 
         try {
             Context.ecAdd("bn128-g2", concatBytes(g1b, g1b), true);
             Context.require(false, "ecAddG2Compressed: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecAddG2Compressed)");
         }
 
         try {
             Context.ecScalarMul("bn128-g1", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), false);
             Context.require(false, "ecScalarMulG1: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecScalarMulG1)");
         }
 
         try {
             Context.ecScalarMul("bn128-g1", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), true);
             Context.require(false, "ecScalarMulG1Compressed: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecScalarMulG1Compressed)");
         }
 
         try {
             Context.ecScalarMul("bn128-g2", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), false);
             Context.require(false, "ecScalarMulG2: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecScalarMulG2)");
         }
 
         try {
             Context.ecScalarMul("bn128-g2", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), true);
             Context.require(false, "ecScalarMulG2Compressed: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecScalarMulG2Compressed)");
         }
 
         try {
             Context.ecPairingCheck("bn128", concatBytes(g1b, g2b), false);
             Context.require(false, "ecPairingCheck: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecPairingCheck)");
         }
 
         try {
             Context.ecPairingCheck("bn128", concatBytes(g1b, g2b), true);
             Context.require(false, "ecPairingCheckCompressed: shall not reach here");
         } catch (IllegalArgumentException e) {
             Context.println("testBN128InvalidPointEncoding - OK (ecPairingCheckCompressed)");
         }
     }
 
 
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
 }
 