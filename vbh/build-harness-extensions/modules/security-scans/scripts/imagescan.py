#
# Check image scan status from Quay registry
# 
# Input: image and token
# Output: image scan status
#
import base64
import http.client
import ssl
import json
import time
import sys
import re
import logging
import argparse

log = logging.getLogger('imagescan')
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s'))
log.addHandler(handler)
log.setLevel(logging.INFO)

def getManifestDigest(host, port, token, namespace, imageName, imageTag):
    url = "/api/v1/repository/{0}/{1}/tag/?limit=100&page=1&onlyActiveTags=true".format(namespace, imageName)
    headers = {}
    if token:
        headers = {"Authorization": "Bearer {0}".format(token)}
    max_tries = 2
    json_data = None
    manifest_digest = None
    conn = None

    for _ in range(max_tries):
        try:
            conn = http.client.HTTPSConnection(host, port, context=ssl._create_unverified_context())
            conn.request("GET", url, headers=headers)

            response = conn.getresponse()
            if response.status == 200:
                data = response.read().decode('utf-8')
                conn.close()
                json_data = json.loads(data)
                break
            else:
                log.warning("getManifestDigest:  {0}".format(response.status))
                conn.close()
                time.sleep(3)
                continue
        except Exception as e:
            log.warning("Exception: {0}".format(e))
            if conn != None:
                conn.close()
                time.sleep(3)
                continue

    if json_data and "tags" in json_data:
        for tag in json_data["tags"]:
            if tag["name"] == imageTag:
                manifest_digest = tag["manifest_digest"]
                break

    return manifest_digest


def getVulnerabilitiesReport(host, port, token, namespace, imageName, manifestDigest):
    #log.info("getVulnerabilitiesReport")
    url = "/api/v1/repository/{0}/{1}/manifest/{2}/security?vulnerabilities=true".format(namespace, imageName, manifestDigest)
    headers = {}
    if token:
        headers = {"Authorization": "Bearer {0}".format(token)}
    max_tries = 2
    json_data = None
    conn = None
    for _ in range(max_tries):
        try:
            conn = http.client.HTTPSConnection(host, port, context=ssl._create_unverified_context())
            conn.request("GET", url, headers=headers)

            response = conn.getresponse()
            if response.status == 200:
                data = response.read().decode('utf-8')
                conn.close()
                json_data = json.loads(data)
                break
            elif response.status == 404:
                log.warning("'{0}' image scan result not found!".format(imageName))
                conn.close()
                break            
            else:
                log.warning("getVulnerabilitiesReport:  {0}".format(response.status))
                conn.close()
                time.sleep(3)
                continue
        except Exception as e:
            log.warning("Exception: {0}".format(e))
            if conn != None:
                conn.close()
                time.sleep(3)
                continue

    return json_data


def getScanReport(host, port, token, namespace, imageName, imageTag):
    global MANIFEST_DIGEST
    scan_report = {'status': 'pending', 'result': ''}
    if not MANIFEST_DIGEST:
        MANIFEST_DIGEST = getManifestDigest(host, port, token, namespace, imageName, imageTag)
    scan_result = getVulnerabilitiesReport(host, port, token, namespace, imageName, MANIFEST_DIGEST)
    if not scan_result:
        return {'status': 'failed', 'result': 'no imagescan report'}

    score = {"Unknown": 0, "Negligible": 1, "Low": 2, "Medium": 3, "High": 4, "Critical": 5}

    if "status" in scan_result and scan_result["status"] == "scanned":
        total_vulnerable_packages = 0
        if "data" in scan_result and "Layer" in scan_result["data"] and "Features" in scan_result["data"]["Layer"]:
            vulnerabilities = []
            for package in scan_result["data"]["Layer"]["Features"]:
                fix_version = None
                severity = "Unknown"
                cve = []
                if "Vulnerabilities" in package:
                    for vul in package["Vulnerabilities"]:
                        cve.append("{0}: {1}".format(vul["Name"], vul["Link"]))
                        if "FixedBy" in vul:
                            if not fix_version or vul["FixedBy"] > fix_version:
                                fix_version = vul["FixedBy"]

                        if "Severity" in vul:
                            if score[vul["Severity"]] > score[severity]:
                                severity = vul["Severity"]

                    if fix_version:
                        vulnerability = {"package_name": package["Name"],
                                        "current_version": package["Version"],
                                        "fix_version": fix_version,
                                        "severity": severity,
                                        "vulnerabilities": cve
                                        }
                        vulnerabilities.append(vulnerability)
                        total_vulnerable_packages += 1

            if len(vulnerabilities) > 0:
                scan_report['status'] = 'failed'
                scan_report['result'] = vulnerabilities
            else:
                scan_report['status'] = 'passed'

        log.info("Total vulnerable packages: {}".format(total_vulnerable_packages))

    return scan_report


def parseImage(image):
    host = ""
    namespace = ""
    imageName = ""
    imageTag = ""
    matchObj = re.match( r'(.*)/(.*)/(.*):(.*)', image)
    if matchObj:
        host = matchObj.group(1)
        namespace = matchObj.group(2)
        imageName = matchObj.group(3)
        imageTag = matchObj.group(4)

    return (host, namespace, imageName, imageTag)


def main():
    global MANIFEST_DIGEST
    MANIFEST_DIGEST = None
    parser = argparse.ArgumentParser(description="Check image scan report")
    parser.add_argument("--imagescan_token", default=None, help="Quay API acess token")
    parser.add_argument("--image", help="The image you want to check the result of the vulnerability scan")

    args = parser.parse_args()
    token = args.imagescan_token
    image = args.image
    port = 443

    if not image:
        print("ERROR: you must provide image")
        exit(-1)

    # if the token is encoded then decode it
    if token:
        try:
            encoded_bytes = token.encode("ascii")
            decoded_bytes = base64.b64decode(encoded_bytes)
            decoded = decoded_bytes.decode("ascii")
            token = decoded
        except Exception as e:
            log.debug("It is not an encoded token")

    (host, namespace, imageName, imageTag) = parseImage(image)
    if not host:
        print("ERROR: image is not in the format of xxxx/xxxx/xxxx:xxxx")
        exit(-1)
    
    print("") 
    print("Checking image {0} scan status...".format(image)) 
    print("") 
    MAX_TIMEOUT = 30 # mins
    maxWaitTime = MAX_TIMEOUT * 60 # in seconds
    start = int(round(time.time()))
    while True:
        scan_report = getScanReport(host, port, token, namespace, imageName, imageTag)
        now = int(round(time.time()))
        elapsedTime = now - start
        if scan_report:
            log.info("Image scan status: {0}".format(scan_report["status"]))
            if scan_report["status"] == "failed":
                print("")
                print(json.dumps(scan_report["result"], indent=4, sort_keys=True))
                print("")
                print("Image scan failed!")
                print("")
                print("Elapsed time: {0} seconds".format(elapsedTime))
                print("")
                exit(-1)
            elif scan_report["status"] == "passed":
                print("")
                print("Image scan passed.")
                print("")
                print("Elapsed time: {0} seconds".format(elapsedTime))
                print("")
                exit()

        if  now - start > maxWaitTime:
           print("ERROR: Quay scan timeout!")
           exit(-1)
        else:
           log.info("sleep 10 seconds...")
           time.sleep(10)
       

if __name__ == '__main__':
    main()
