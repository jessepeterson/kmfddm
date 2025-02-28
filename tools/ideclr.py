#!/usr/bin/env python3

import argparse
import uuid
import sys
import json

# closure which generates a function that returns a simple DDM declaration
def make_simple_wrap(decl_type):
    def make_simple(args):
        nonlocal decl_type
        return {
            "Type": decl_type,
            "Payload": {},
        }

    return make_simple


def make_activation(args):
    decl = {
        "Type": "com.apple.activation.simple",
        "Payload": {
            "StandardConfigurations": args.declaration,
        },
    }
    if hasattr(args, "predicate") and args.predicate:
        decl["Payload"]["Predicate"] = args.predicate
    return decl


def make_subscription(args):
    decl = {
        "Type": "com.apple.configuration.management.status-subscriptions",
        "Payload": {
            "StatusItems": [{"Name": x} for x in args.item],
        },
    }
    return decl


def make_profile_wrap(decl_type="com.apple.configuration.legacy"):
    def make_profile(args):
        payload = {
            "ProfileURL": args.url,
        }
        if decl_type == "com.apple.configuration.legacy.interactive":
            payload["VisibleName"] = args.visiblename
        decl = {
            "Type": decl_type,
            "Payload": payload,
        }
        return decl

    return make_profile


def make_org(args):
    payload = {
        "Name": args.name,
    }
    if hasattr(args, "email") and args.email:
        payload["Email"] = args.email
    if hasattr(args, "url") and args.url:
        payload["URL"] = args.url
    if hasattr(args, "identitytoken") and args.identitytoken:
        payload["Proof"] = {"IdentityToken": args.identitytoken}
    return {
        "Type": "com.apple.management.organization-info",
        "Payload": payload,
    }


def make_test(args):
    payload = {
        "Echo": args.echo,
    }
    if hasattr(args, "returnstatus") and args.returnstatus:
        payload["ReturnStatus"] = args.returnstatus
    return {
        "Type": "com.apple.configuration.management.test",
        "Payload": payload,
    }


def make_test_subp(parent_parser):
    decl_type = "com.apple.configuration.management.test"
    p = parent_parser.add_parser("test", help=decl_type + " DDM declaration")
    p.add_argument(
        "echo",
        type=str,
        help="echo string",
    )
    p.add_argument(
        "-r",
        "--returnstatus",
        type=str,
        help="email address",
    )
    p.set_defaults(func=make_test)
    return p


def make_org_subp(parent_parser):
    decl_type = "com.apple.management.organization-info"
    p = parent_parser.add_parser("org-info", help=decl_type + " DDM declaration")
    p.add_argument(
        "name",
        type=str,
        help="name of organization",
    )
    p.add_argument(
        "-e",
        "--email",
        type=str,
        help="email address",
    )
    p.add_argument(
        "-u",
        "--url",
        type=str,
        help="URL of the organization",
    )
    p.add_argument(
        "-t",
        "--identitytoken",
        type=str,
        help="Organization verification identity token",
    )
    p.set_defaults(func=make_org)
    return p


def make_profile_subp(parent_parser):
    decl_type = "com.apple.configuration.legacy"
    p = parent_parser.add_parser("profile", help=decl_type + " DDM declaration")
    p.add_argument(
        "url",
        type=str,
        help="URL of profile",
    )
    p.set_defaults(func=make_profile_wrap(decl_type))
    return p


def make_iprofile_subp(parent_parser):
    decl_type = "com.apple.configuration.legacy.interactive"
    p = parent_parser.add_parser("iprofile", help=decl_type + " DDM declaration")
    p.add_argument(
        "url",
        type=str,
        help="URL of profile",
    )
    p.add_argument(
        "visiblename",
        type=str,
        help="Visible name of configuration.",
    )
    p.set_defaults(func=make_profile_wrap(decl_type))
    return p


def make_subscription_subp(parent_parser):
    decl_type = "com.apple.configuration.management.status-subscriptions"
    p = parent_parser.add_parser("subscription", help=decl_type + " DDM declaration")
    p.add_argument(
        "item",
        nargs="+",
        type=str,
        help="status item to subscribe to",
    )
    p.set_defaults(func=make_subscription)
    return p


def make_activation_subp(parent_parser):
    decl_type = "com.apple.activation.simple"
    p = parent_parser.add_parser("activation", help=decl_type + " DDM declaration")
    p.add_argument(
        "-p",
        "--predicate",
        type=str,
        help="activation predicate",
    )
    p.add_argument(
        "declaration",
        nargs="+",
        type=str,
        help="declaration to activate",
    )
    p.set_defaults(func=make_activation)
    return p


def make_simple_decl_subp(simple_name, decl_type, parent_parser):
    p = parent_parser.add_parser(
        simple_name,
        help=decl_type + " DDM declaration",
    )
    p.set_defaults(func=make_simple_wrap(decl_type))
    return p

def make_watch_enrollment(args):
    decl = {
        "Type": "com.apple.configuration.watch.enrollment",
        "Payload": {
            "EnrollmentProfileURL": args.url,
        },
    }
    return decl


def make_watch_enrollment_subp(parent_parser):
    decl_type = "com.apple.configuration.watch.enrollment"
    p = parent_parser.add_parser("watch-enrollment", help=decl_type + " DDM declaration")
    p.add_argument(
        "url",
        type=str,
        help="URL of the enrollment profile",
    )
    p.set_defaults(func=make_watch_enrollment)
    return p


def main():
    p = argparse.ArgumentParser(description="DDM declaration generator")
    p.add_argument(
        "-i",
        "--identifier",
        type=str,
        default=str(uuid.uuid4()),
        help="declaration identifier (auto-generated UUID if not specified)",
    )
    p.add_argument(
        "-t",
        "--token",
        type=str,
        help="declaration ServerToken",
    )
    subps = p.add_subparsers(
        title="DDM declarations",
        help="supported DDM declarations",
    )

    make_simple_decl_subp("properties", "com.apple.management.properties", subps)
    make_activation_subp(subps)
    make_subscription_subp(subps)
    make_profile_subp(subps)
    make_iprofile_subp(subps)
    make_org_subp(subps)
    make_test_subp(subps)

    args = p.parse_args()

    # command and random are mutually exclusive
    if not hasattr(args, "func"):
        p.print_help()
        sys.exit(2)

    d = args.func(args)
    d["Identifier"] = args.identifier
    if hasattr(args, "token") and args.token:
        d["ServerToken"] = args.token
    json.dump(d, sys.stdout, indent=4)
    print()


if __name__ == "__main__":
    main()
