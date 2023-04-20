package foundation.icon.ee.util.bn128;

interface Bytes {

    default byte[] concat(byte[]... args) {
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

    default byte[] withZeroPadding(int length, byte[] data) {
        if (data.length >= length) {
            return data;
        }
        byte[] res = new byte[length];
        System.arraycopy(data, 0, res, length - data.length, data.length);
        return res;
    }

}
