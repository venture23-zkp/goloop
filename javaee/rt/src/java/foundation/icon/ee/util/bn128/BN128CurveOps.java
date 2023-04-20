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

package foundation.icon.ee.util.bn128;

import java.math.BigInteger;
import java.util.Arrays;

public class BN128CurveOps {
    public static final int G1_LEN = 32;
    public static final int G2_LEN = 64;

    public static byte[] g1Add(byte[] data, boolean compressed) {
        if (compressed) {
            throw new IllegalArgumentException("BN128: g1Add: compressed points are not supported yet!");
        }
        BN128<Fp> acc = BN128G1.ZERO;
        int size = compressed ? G1_LEN : 2 * G1_LEN;
        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException(
                    "BN128: g1Add: invalid data layout: expected a multiple of " + size + " bytes, got " + data.length);
        }
        for (int i = 0; i < data.length; i += size) {
            byte[] x = Arrays.copyOfRange(data, i, i + G1_LEN);
            byte[] y = Arrays.copyOfRange(data, i + G1_LEN, i + 2 * G1_LEN);
            BN128G1 p = BN128G1.create(x, y);
            acc = acc.add(p);
        }
        return acc.bytes();
    }

    public static byte[] g2Add(byte[] data, boolean compressed) {
        if (compressed) {
            throw new IllegalArgumentException("BN128: g2Add: compressed points are not supported yet!");
        }
        BN128<Fp2> acc = BN128G2.ZERO;
        int size = compressed ? G2_LEN : 2 * G2_LEN;
        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException(
                    "BN128: g2Add: invalid data layout: expected a multiple of " + size + " bytes, got " + data.length);
        }
        for (int i = 0; i < data.length; i += size) {
            byte[] x_i = Arrays.copyOfRange(data, i, i + G2_LEN / 2);
            byte[] x = Arrays.copyOfRange(data, i + G2_LEN / 2, i + 2 * G2_LEN / 2);
            byte[] y_i = Arrays.copyOfRange(data, i + 2 * G2_LEN / 2, i + 3 * G2_LEN / 2);
            byte[] y = Arrays.copyOfRange(data, i + 3 * G2_LEN / 2, i + 4 * G2_LEN / 2);
            BN128G2 p = BN128G2.create(x, x_i, y, y_i);
            acc = acc.add(p);
        }
        return acc.bytes();
    }

    public static byte[] g1ScalarMul(byte[] scalarBytes, byte[] data, boolean compressed) {
        if (compressed) {
            throw new IllegalArgumentException("BN128: g1ScalarMul: compressed points are not supported yet!");
        }

        int size = compressed ? G1_LEN : 2 * G1_LEN;
        if (data.length != size) {
            throw new IllegalArgumentException(
                    "BN128: g1ScalarMul: invalid data layout: expected=" + size + " bytes, got " + data.length);
        }

        BigInteger scalar = new BigInteger(1, scalarBytes);
        if (scalar.compareTo(Params.R) >= 0) {
            throw new IllegalArgumentException(
                    "BN128: g1ScalarMul: invalid scalar, expected less than " + Params.R + ", got " + scalar);
        }

        byte[] x = Arrays.copyOfRange(data, 0, G1_LEN);
        byte[] y = Arrays.copyOfRange(data, G1_LEN, 2 * G1_LEN);

        BN128G1 point = BN128G1.create(x, y);

        BN128<Fp> r = point.mul(scalar).toEthNotation();
        return r.bytes();
    }

    public static byte[] g2ScalarMul(byte[] scalarBytes, byte[] data, boolean compressed) {
        if (compressed) {
            throw new IllegalArgumentException("BN128: g2ScalarMul: compressed points are not supported yet!");
        }

        int size = compressed ? G2_LEN : 2 * G2_LEN;
        if (data.length != size) {
            throw new IllegalArgumentException(
                    "BN128: g2ScalarMul: invalid data layout: expected=" + size + " bytes, got " + data.length);
        }

        BigInteger scalar = new BigInteger(1, scalarBytes);
        if (scalar.compareTo(Params.R) >= 0) {
            throw new IllegalArgumentException(
                    "BN128: g2ScalarMul: invalid scalar, expected less than " + Params.R + ", got " + scalar);
        }

        byte[] x_i = Arrays.copyOfRange(data, 0 * G2_LEN / 2, 1 * G2_LEN / 2);
        byte[] x = Arrays.copyOfRange(data, 1 * G2_LEN / 2, 2 * G2_LEN / 2);
        byte[] y_i = Arrays.copyOfRange(data, 2 * G2_LEN / 2, 3 * G2_LEN / 2);
        byte[] y = Arrays.copyOfRange(data, 3 * G2_LEN / 2, 4 * G2_LEN / 2);

        BN128G2 p = BN128G2.create(x, x_i, y, y_i);

        BN128<Fp2> r = p.mul(scalar).toEthNotation();
        return r.bytes();
    }

    public static boolean pairingCheck(byte[] data, boolean compressed) {
        if (compressed) {
            throw new IllegalArgumentException("BN128: g1ScalarMul: compressed points are not supported yet!");
        }

        int g1Size = compressed ? G1_LEN : 2 * G1_LEN;
        int g2Size = compressed ? G2_LEN : 2 * G2_LEN;
        int size = g1Size + g2Size;

        if (data.length == 0 || data.length % size != 0) {
            throw new IllegalArgumentException("BN128: pairingCheck: invalid data layout: expected a multiple of "
                    + size + " bytes, got " + data.length);
        }

        PairingCheck check = PairingCheck.create();

        for (int i = 0; i < data.length; i += size) {
            int offset = i;
            BN128G1 p1 = BN128G1.create(
                    Arrays.copyOfRange(data, offset, offset + G1_LEN),
                    Arrays.copyOfRange(data, offset + G1_LEN, offset + 2 * G1_LEN));

            offset += 2 * G1_LEN;
            BN128G2 p2 = BN128G2.create(
                    Arrays.copyOfRange(data, offset + G2_LEN / 2 * 1, offset + G2_LEN / 2 * 2),
                    Arrays.copyOfRange(data, offset + G2_LEN / 2 * 0, offset + G2_LEN / 2 * 1),
                    Arrays.copyOfRange(data, offset + G2_LEN / 2 * 3, offset + G2_LEN / 2 * 4),
                    Arrays.copyOfRange(data, offset + G2_LEN / 2 * 2, offset + G2_LEN / 2 * 3)
            );

            if (p1 == null || p2 == null)
                throw new IllegalArgumentException("BN128: pairingCheck: G1 or G2 point not in subgroup!");

            check.addPair(p1, p2);
        }

        check.run();

        return check.result() > 0;
    }

}
