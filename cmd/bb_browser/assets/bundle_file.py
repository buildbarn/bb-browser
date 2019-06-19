import sys

with open(sys.argv[1], "rb") as fin:
    with open(sys.argv[2], "w") as fout:
        fout.write("package %s\n" % sys.argv[3])
        fout.write("var %s = []byte{" % sys.argv[4])
        while True:
            chunk = fin.read(1024)
            if not chunk:
                break
            for c in chunk:
                # Python 2 requires explicit conversion to an integer.
                try:
                    c = ord(c)
                except TypeError:
                    pass
                fout.write("%d," % c)
        fout.write("}")
