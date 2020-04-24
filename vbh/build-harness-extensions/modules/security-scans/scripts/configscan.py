#
# Scan the image config to find potential leaked credentials
#
# Input: image and patterns
# Output: print out matching patterns found
#
import sys
import subprocess
import re
import logging
import argparse
import queue as queue
from threading import Thread

log = logging.getLogger('configscan')
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s'))
log.addHandler(handler)
log.setLevel(logging.INFO)

def execute_command(command, patterns):
    result = ""
    patarray = patterns.split(";")
    p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = p.communicate()
    if stdout:
        out = stdout.decode()
        for pattern in patarray:
            pattern = pattern.strip()
            if pattern:
                match = re.findall(r'.*{0}.*'.format(pattern), out, re.IGNORECASE)
                if match:
                    result += "==========================================================================================\n"
                    result += "[configscan] Command: {0}=\n".format(command)
                    result += "[configscan] -----------------------------------------------------------------------------\n"
                    result += "[configscan] Found pattern: '{0}'. Please verify if this is a credential or something should not be here or false positive.\n".format(pattern)
                    result += "[configscan] ===>>> {0}\n".format(match)
                    result += "==========================================================================================\n"
    return result


class ThreadImageConfigScan(Thread):
    """Threaded image config scan"""
    def __init__(self, in_queue, out_queue, patterns):
        Thread.__init__(self)
        self.in_queue = in_queue
        self.out_queue = out_queue
        self.patterns = patterns

    def run(self):
        while True:
            result = ""
            # Grabs command from queue
            command = self.in_queue.get()

            try:
               result = execute_command(command, self.patterns)
               if result:
                  self.out_queue.put(result)
            except Exception as e:
                info.error("Error: ThreadImageConfigScan Exception for: {0}  Message: {1}".format(command, e))
            finally:
                # Signals to queue job is done
                self.in_queue.task_done()


def runConfigScan(image, patterns):
    in_queue = queue.Queue()
    out_queue = queue.Queue()

    commands = []
    commands.append("docker history --no-trunc {0}".format(image))
    commands.append("docker inspect {0}".format(image))
    commands.append("docker run --rm --entrypoint /bin/sh -e LICENSE=accept {0} -c \"env\"".format(image))

    max_threads = len(commands)

    # Spawn a pool of threads, and pass them queue instance
    for i in range(max_threads):
        t = ThreadImageConfigScan(in_queue, out_queue, patterns)
        t.daemon = True
        t.start()

    # Populate queue with data
    for command in commands:
        in_queue.put(command)

    # Wait on the queue until everything has been processed
    in_queue.join()

    result = ""

    while True:
        if not out_queue.empty():
           result += out_queue.get()
        else:
           break

    return result


def main():
    parser = argparse.ArgumentParser(description="Scan the image config to find potential leaked credentials")
    parser.add_argument("--image", help="The image you want to check the config")
    parser.add_argument("--patterns", help="Scan patterns")

    args = parser.parse_args()
    image = args.image
    patterns = args.patterns

    if not image:
        log.info("ERROR: you must provide image")
        exit(-1)

    if not patterns:
        log.info("ERROR: you must provide patterns (a list of pattern to scan)")
        exit(-1)

    log.info("Configscan image: {0}".format(image))
    log.info("Configscan patterns: {0}".format(patterns))

    result = runConfigScan(image, patterns)
    if result:
        print(result)


if __name__ == '__main__':
    main()
