#!/usr/bin/python2
import os
import sys
import getopt
import hashlib

hash_function = "sha512"

def usage():
    print "Usage: ./bootsum.py [options]"
    print "Options:"
    print "-d <directory>:  print checksum for all files in a directory"
    print "-f <filename>:   print checksum for single file"
    print "-c <filename>:   read and verify checksums from file"
    print "Examples:"
    print "./bootsum.py -d /boot > /etc/bootsum.conf"
    print "./bootsum.py -f /etc/shadow"
    print "./bootsum.py -c /etc/bootsum.conf"
    sys.exit(0)

def read_config(config):
    files = {}
    try:
        with open(config, 'r') as fd:
            for line in fd:
                (k, v) = line.rstrip().split(':', 2)
                files[k] = v
    except:
        print "[*] Error opening file %s" % config
        sys.exit(0)
    fd.close()
    print "[*] Read %d entries from %s" % (len(files), config)
    return files

def return_hash(filename):
    global hash_function

    try:
        with open(filename, 'r') as fd:
            h = hashlib.new(hash_function)
            h.update(fd.read())
            fd.close()
            return h.hexdigest()
    except IOError: # file is a directory, permission problems, etc
        pass

def main():
    config = ""
    filename = ""
    directory = ""
    filelist = []
    hashdict = {}
    bad_hash = 0

    if not len(sys.argv[2:]):
        usage()

    try:
        opt, args = getopt.getopt(sys.argv[1:], "d:f:c:", ["dir", "file", "conf"])
    except getopt.GetoptError as err:
        print str(err)
        usage()

    for o, a in opt:
        if o in ("-d", "--dir"):
            directory = a
        elif o in ("-f", "--file"):
            filename = a
        elif o in ("-c", "--conf"):
            config = a
        else:
            assert False, "Unhandled option"

    if filename or directory:
        if directory:
            os.chdir(directory)
            for filename in os.listdir(directory):
                filelist.append(os.path.realpath(filename))
        else:
            filelist = [filename]
    elif config:
        hashdict = read_config(config)
        for k in hashdict.keys():
            filelist.append(k)

    for filename in filelist:
        if os.path.isfile(filename):
            try:
                os.access(filename, os.R_OK)
                result = return_hash(filename)
                if filename in hashdict:
                    if hashdict.get(filename) != result:
                        bad_hash += 1
                        print "[*] WARNING: File %s has bad hash" % filename
                else:
                    print "%s:%s" % (filename, result)
            except IOError as e:
                if e.errno == errno.EACCES:
                    print "[*] Can't access file %s" % filename
                raise

if __name__ == "__main__":
    main()
