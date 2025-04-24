#!/usr/bin/env python3

"""
KMFDDM sync tool. This tool synchronizes declarations and sets from
a directory and uploads them to a KMFDDM server. It tries to be smart
about only notifying the changed items (declarations, sets) and only
performing one set of notifications for all changed items.

The specified directory is walked to find files that can be synced. Any
file with a ".json" extension is assumed to be declaration and is
uploaded as such. Files matching "set.$SET.txt" are assumed to contain
one line for each declaration identifier wanting to be associated to
"$SET" set name. Include a minus ("-") sign in front of a declaration
to dissociate it, or an octothorp/hash ("#") for a comment.

For example a directory might look like this:

  ./a/com.example.test.json
  ./b/com.example.act.json
  ./c/set.default.txt

Here each of the two JSON files will be treated as declarations and the
txt file will apply each declaration (one per line) to the "default"
set.
"""

import os
import json
import ssl
import urllib.request
from urllib.parse import urlencode
import argparse
import base64
import re

ssl._create_default_https_context = ssl._create_stdlib_context

set_pattern = r"set\.(.*?)\.txt"

nonotify_params = urlencode(
    {
        "nonotify": "1",
    }
)


def make_declaration_req(
    api_base_url: str, auth_header: str, declaraion_data: bytes
) -> urllib.request.Request:
    url = api_base_url + "/declarations?" + nonotify_params
    req = urllib.request.Request(url=url, method="PUT")
    req.add_header("Content-Type", "application/json")
    req.add_header("Authorization", f"Basic {auth_header}")
    req.data = declaraion_data
    return req


def make_set_req(
    api_base_url: str, auth_header: str, method: str, set_name: str, declaration_id: str
) -> urllib.request.Request:
    url = (
        api_base_url
        + "/set-declarations/"
        + set_name
        + "?"
        + urlencode(
            {
                "declaration": declaration_id,
                "nonotify": "1",
            }
        )
    )
    req = urllib.request.Request(url=url, method=method)
    req.add_header("Authorization", f"Basic {auth_header}")
    return req


def sync_dir(dir, api_base_url, user, key):
    # collectors for later notifying/reporting
    changed_decls = []
    unchanged_decls = []
    changed_sets = []
    unchanged_sets = []

    auth_header = base64.b64encode(f"{user}:{key}".encode("utf-8")).decode("utf-8")
    set_files = []
    for root, dirs, files in os.walk(dir):
        for file in files:
            file_path = os.path.join(root, file)
            if file.endswith(".json"):
                with open(file_path, "rb") as f:
                    try:
                        data = f.read()
                        decl = json.loads(data)
                        req = make_declaration_req(api_base_url, auth_header, data)
                        response = urllib.request.urlopen(req)
                        status_code = response.getcode()
                        if status_code == 204:
                            id = decl["Identifier"]
                            print(f"changed declaration {id}")
                            changed_decls.append(id)
                        else:
                            print(
                                f"WARNING: unknown status code declaration: {status_code}"
                            )
                    except json.JSONDecodeError as e:
                        print(f"ERROR parsing {file}: {str(e)}")
                    except urllib.error.HTTPError as e:
                        if e.code == 304:
                            unchanged_decls.append(decl["Identifier"])
                        else:
                            json_error = ""
                            try:
                                json_error = json.loads(e.read())["error"]
                            except:
                                pass
                            print(f"ERROR uploading {file}: {str(e)}: {json_error}")
                    except urllib.error.URLError as e:
                        print(f"ERROR uploading {file}: {str(e)}")
            else:
                match = re.search(set_pattern, file)
                if not match:
                    continue
                # just collect them for now as we want to process
                # sets after all of the declarations have been uploaded
                set_files.append((file_path, file, match.group(1)))

    for file_path, file, set_name in set_files:
        with open(file_path, "r") as f:
            decls = [line.strip() for line in f]
            changed_set = False
            for decl_id in decls:
                if decl_id == "" or decl_id[0] == "#":
                    # comment
                    continue
                method = "PUT"
                if decl_id[0] == "-":
                    # a declaration in a set file preceded with a minus (-)
                    # indicates removal of the association
                    decl_id = decl_id[1:].strip()
                    method = "DELETE"
                req = make_set_req(api_base_url, auth_header, method, set_name, decl_id)
                try:
                    response = urllib.request.urlopen(req)
                    status_code = response.getcode()
                    if status_code == 204:
                        if method == "PUT":
                            print(
                                f"associated declaration {decl_id} with set {set_name}"
                            )
                        elif method == "DELETE":
                            print(
                                f"dissociated declaration {decl_id} from set {set_name}"
                            )
                        changed_set = True
                    else:
                        print(f"WARNING: unknown status code sets: {status_code}")
                except urllib.error.HTTPError as e:
                    if e.code != 304:
                        json_error = ""
                        try:
                            json_error = json.loads(e.read())["error"]
                        except:
                            pass
                        print(
                            f"ERROR associating {decl_id} with {set_name}: {str(e)}: {json_error}"
                        )
                except urllib.error.URLError as e:
                    print(f"ERROR associating {decl_id} with {set_name}: {str(e)}")
            if changed_set:
                changed_sets.append(set_name)
            else:
                unchanged_sets.append(set_name)

    if len(unchanged_decls) > 0:
        print(f"unchanged declarations: {len(unchanged_decls)}")
    if len(unchanged_sets) > 0:
        print(f"unchanged sets: {len(unchanged_sets)}")

    if len(changed_decls) > 0 or len(changed_sets) > 0:
        params = {}
        if len(changed_decls) > 0:
            params["declaration"] = changed_decls
            print(
                f"changed declarations ({len(changed_decls)}): "
                + ", ".join(changed_decls)
            )
        if len(changed_sets) > 0:
            params["set"] = changed_sets
            print(f"changed sets ({len(changed_sets)}): " + ", ".join(changed_sets))
        encoded_params = urlencode(params, doseq=True)
        notify_url = api_base_url + "/notify?" + encoded_params
        req = urllib.request.Request(url=notify_url, method="POST")
        req.add_header("Authorization", f"Basic {auth_header}")
        try:
            response = urllib.request.urlopen(req)
            status_code = response.getcode()
            if status_code == 204:
                print(f"sent notify")
            else:
                print(f"WARNING: unknown status code notify: {status_code}")
        except (urllib.error.HTTPError, urllib.error.URLError) as e:
            print(f"ERROR notifying to {notify_url}: {str(e)}")
    else:
        print("no changed declarations or sets")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="KMFDDM syncer",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.epilog = __doc__
    parser.add_argument(
        "dir",
        type=str,
        help="path to the directory containing declarations and set files",
    )
    parser.add_argument(
        "--apibaseurl",
        type=str,
        default=os.environ.get("API_BASE_URL", "http://[::1]:9002/v1"),
        help="URL for uploading the JSON files (default: http://[::1]:9002/v1)",
    )
    parser.add_argument(
        "--key",
        type=str,
        default=os.environ.get("API_KEY", "kmfddm"),
        help="Password for HTTP Basic authentication",
    )
    parser.add_argument(
        "--user",
        type=str,
        default=os.environ.get("API_USER", "kmfddm"),
        help="Username for HTTP Basic authentication",
    )
    args = parser.parse_args()

    sync_dir(args.dir, args.baseurl, args.user, args.key)
