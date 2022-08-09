# KMFDDM Quickstart Guide

This quickstart guide is intended to get a very basic Apple Declarative Management (DDM) server running and configured with KMFDDM.

## Initial setup, dependencies & requirements

For this guide you'll need to:

1. Have familiarity with Declarative Management (DDM) concepts including declarations, tokens, status reports, etc. Please review Apple's WWDC 2021 video ["Meet Declarative Device Management"](https://developer.apple.com/videos/play/wwdc2021/10131/) as a primer.
1. Have familiarity with KMFDDM. Please read project the [README](../README.md).
1. Have a working NanoMDM v0.3.0+ environment setup and running with devices able to enroll.
1. Have a compatible device like iOS 15.0+ or macOS Ventura 13.0+ enrolled into your NanoMDM environment. **You'll need to know the enrollment ID of the device(s).**
1. Take note of your NanoMDM server's command enqueue HTTP endpoint and API key (password). For this guide we'll be using `http://[::1]:9000/v1/enqueue` as if you're running NanoMDM locally.
1. Obtain the KMFDDM server by either downloading a release zip or checking out the code and compiling from source. Take note of where the KMFDDM server binary and the helper scrips are. They're in the [tools](../tools) directory in the source repository — but should also be in the binary release zip.
1. Create and setup the MySQL schema using the [schema file](../storage/mysql/schema.sql) (e.g. creating a new database, users, and executing the `CREATE TABLE` statements). Note the [DSN](https://github.com/go-sql-driver/mysql#dsn-data-source-name) where you created this.

With those steps taken care of we can now start the KMFDDM server.

### Starting KMFDDM

Starting the server looks like this:

```sh
./kmfddm-darwin-amd64 \
    -api supersecret \
    -enqueue 'http://[::1]:9000/v1/enqueue/' \
    -enqueue-key supernanosecret \
    -storage-dsn 'kmfddm:kmfddm@tcp(192.168.0.1:3306)/kmfddm' \
    -debug
```

* `-api` sets the *KMFDDM* API key to "supersecret"
* `-enqueue` sets the URL for the enqueue endpoint for NanoMDM. By default NanoMDM listens on port 9000 and the enqueue API endpoint is at "/v1/enqueue/". Be sure to include the trailing slash "/".
* `-enqueue-key` sets the API key for enqueueing commands to *NanoMDM* to "supernanosecret" (this is configured in your NanoMDM environment)
* `-storage-dsn` sets the storage DSN for the MySQL storage backend. In this example case we're connecting to a "kmfddm" database at host "192.168.0.1" with username and password "kmfddm".
* `-debug` turns on additional debug logging

If the server started successfully you should see:

```
2022/08/08 14:21:45 level=info msg=starting server listen=:9002
```

Note that it started on port 9002. This can be changed with the `-listen` switch.

### Reconfigure NanoMDM

We'll need to "point" NanoMDM at our KMFDDM instance. This is done by utilizing the `-dm` switch when starting NanoMDM. NanoMDM v0.3.0+ is required for this to work. This might look like:

```sh
./nanomdm-darwin-amd64 -ca ca.pem -api nanomdm -debug -dm 'http://[::1]:9002/'
```

After this (and the above) steps are done NanoMDM should be "pointed" at KMFDDM (for the Declarative Management protocol) and KMFDDM should be "pointed" back at NanoMDM (to enqueue classic MDMv1 commands to enrollments).

### Setup environment

To use the helper shell scripts you'll want to set some environment variables first:

```sh
export BASE_URL='http://[::1]:9002'
export API_KEY='kmfddm'
```

The base URL is where KMFDDM is running and the API key is what you gave to the `-api` switch when you started KMFDDM. The shell scripts are simple `curl` wrapprs that access KMFDDM's REST-ish API and need to know where it is running and how to authenticate.

## Basic setup

Awesome, you got things running! Now we'll setup a very basic DDM environment to play with. This is written as a sort of story to follow along with yourself. Hopefully it'll help to get you started and explore some of the DDM concepts.

### Our first declaration

To create a simple declaration we can use the `ideclr.py` tool:

```sh
$ ./tools/ideclr.py org-info 'ACME Widgets Co.'
{
    "Type": "com.apple.management.organization-info",
    "Payload": {
        "Name": "ACME Widgets Co."
    },
    "Identifier": "274c52b0-fbc5-4453-af46-c4c493a75c6f"
}
```

As you can see a declaration is just JSON. For our initial setup we'll want to use a *configuration* declaration (the above is a *management* declaration) and we'll also want to use a sane identifier — manually keeping track of UUIDs is no fun. Let's use a "test" declaration:

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM'
{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "KMFDDM"
    },
    "Identifier": "com.example.test"
}
```

Here we've used the `-i` siwtch to `ideclr.py` to provide a custom identifier. We've specified the "test" sub-command which represents the `com.apple.configuration.management.test` declaration type — a *configuration* declaration. We'll proceed with this declaration. Let's upload it:

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM' | ./tools/api-declaration-put.sh -
Response HTTP Code: 204
```

Here we've run the same command to generate the same declaration but we've asked a helper tool to upload the declaration to the server. This declaration could have also just been in a file, too. Since we provided the "-" parameter to the shell script the JSON is read from standard in via a pipe which takes the output of the `ideclr.py` tool (the declaration JSON) directly. The 204 response here means it was a success. Fruther, the server logs show us the declaration was uploaded and was new or the content had changed (`changed=true`):

```
2022/08/08 14:53:22 level=debug handler=put-declaration trace_id=e8ea613504d9ba03 decl_id=com.example.test decl_type=com.apple.configuration.management.test msg=stored declaration changed=true
2022/08/08 14:53:22 level=debug service=notifier msg=no enrollments to notify
```

Don't worry about the "no enrollments" message just yet — this is expected (this declaration isn't available to any enrollments yet).

Let's check that this declaration is on the server by listing all the declarations:

```sh
$ ./tools/api-declarations-get.sh 
["com.example.test"]
```

Good: the server sees our one declaration we just uploaded. Let's retrieve it from the server:

```sh
$ ./tools/api-declaration-get.sh com.example.test | jq .
{
  "Type": "com.apple.configuration.management.test",
  "Payload": {
    "Echo": "KMFDDM"
  },
  "Identifier": "com.example.test",
  "ServerToken": "3959b17e63154d7d81e3ecd699fcae974c3a4fca"
}
```

Here I've piped the result through [jq](https://stedolan.github.io/jq/) just so it can pretty-print and colorize the output for easier inspection. In this output we can also see that the declaration now has a "ServerToken" property. This is a unique token used to identify changes in the declaration. It can be thought of a little like a version. So what happens if we *change* the declaration (i.e. by adding an exclamation to the `Echo` property):

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM!' | ./tools/api-declaration-put.sh -
Response HTTP Code: 204
$ ./tools/api-declaration-get.sh com.example.test | jq .
{
  "Type": "com.apple.configuration.management.test",
  "Payload": {
    "Echo": "KMFDDM!"
  },
  "Identifier": "com.example.test",
  "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
}
```

We can see that the `ServerToken` key changed to reflect the updated declaration. What happens if we try to update again but don't changed it?

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM!' | ./tools/api-declaration-put.sh -
Response HTTP Code: 304
```

Our 304 response tells us the resource has not been modified. We can also see this in the server log (`changed=false`):

```
2022/08/08 15:05:29 level=debug handler=put-declaration trace_id=f3646fcb3df3ce73 decl_id=com.example.test decl_type=com.apple.configuration.management.test msg=stored declaration changed=false
```

Cool! So we have a declaration uploaded into KMFDDM. Now what?

### Our first set

Next we'll want to associate our declaration with a set. We'll start with a generic set name for now:

```sh
$ ./tools/api-set-declarations-put.sh default com.example.test
Response HTTP Code: 204
```

Here we've associated the "com.example.test" declaration with the set named "default." It doesn't need to be pre-created or initialized or anything. Just associate the declaration to the name and it'll be "created." Let's verify the set has the declaration:

```sh
$ ./tools/api-set-declarations-get.sh default | jq .
[
  "com.example.test"
]
```

Good, the set named "default" contains the "com.example.test" declaration. Let's move on.

### Our first "enrollment"

As noted in the [README](../README.md) sets don't really do anything until they are associated with enrollment IDs. But once we do that some of the fun will start to happen so I'll try and walk us through it. 

When we say "enrollment" here it's not quite like an MDM enrollment. Here I'm specifically talking about associating a NanoMDM enrollment ID to the "default" set that we just created, above. Of course, once this happens, as we'll see, we'll "turn on" DDM for the enrollment which acts kinda sorta like an "enrollment" *for the DDM protocol* so it can get a little confusing. Overall when referring to "enrollments" throughout this document we're almost certainly talking about NanoMDM enrollment IDs.

So, let's assign my enrollment ID (`2FF3196C-CACE-4AFE-9918-01C38160006F`) to the default set and see what happens:

```sh
$ ./tools/api-enrollment-sets-put.sh 2FF3196C-CACE-4AFE-9918-01C38160006F default
Response HTTP Code: 204
```

Seemed to work (204 response code)! But what's happened under the hood? Let's break down the server logs to see what's happened:

```
2022/08/08 15:19:20 level=info handler=log trace_id=30d85f2c9843fcc3 addr=::1 method=PUT path=/v1/enrollment-sets/2FF3196C-CACE-4AFE-9918-01C38160006F agent=curl/7.54.0
2022/08/08 15:19:20 level=info service=notifier msg=sending command count=1 include tokens=true command_uuid=6cccd191-5709-41a7-b534-69c9a7525f23 http_status=200
```

The above two lines tell us that we submitted an API request to change the associated sets for the enrollment ID. Notably the second line tells us that it has generated a `DeclarativeManagement` MDM command, reported that command's UUID, sent the command (enqueued it) to NanoMDM and received a 200 (success) response.

The rest of the lines are the enrollment responding to that MDM command which is the DDM protocol communication:

```
2022/08/08 15:19:27 level=info handler=log trace_id=4eac6d3268ebd00d addr=::1 method=PUT path=/status agent=Go-http-client/1.1
2022/08/08 15:19:27 level=debug handler=status-report trace_id=4eac6d3268ebd00d enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=status report decl_count=0 error_count=0 value_count=36
```

Here the enrollment has sent us its very first initial Status Report. We'll talk more about status reports in a bit. It hasn't reported any errors or declarations (yet, anyway). It has, though, sent us "36" "values." Again, more on that later.

```
2022/08/08 15:19:27 level=info handler=log trace_id=03dceeef5ba616e5 addr=::1 method=GET path=/tokens agent=Go-http-client/1.1
2022/08/08 15:19:27 level=debug handler=tokens trace_id=03dceeef5ba616e5 enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=retrieved tokens
2022/08/08 15:19:27 level=info handler=log trace_id=afd56a390cf5555b addr=::1 method=GET path=/declaration-items agent=Go-http-client/1.1
2022/08/08 15:19:27 level=debug handler=declaration-items trace_id=afd56a390cf5555b enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=retrieved declaration items
```

Here the enrollment retrieves its DDM "tokens" and "declaration-items" from KMFDDM. These are endpoints the enrollment uses to retrieve the list of declarations and to tell if they have changed or not. It'll do this alot. If the declarations have changed the enrollment will fetch them. Since the enrollment doesn't have any matching declarations yet (meaning, all of them have "changed") it should fetch the declaration we've assigned.

```
2022/08/08 15:19:28 level=info handler=log trace_id=78a0c9b159e15776 addr=::1 method=GET path=/declaration/configuration/com.example.test agent=Go-http-client/1.1
2022/08/08 15:19:28 level=debug handler=declaration trace_id=78a0c9b159e15776 decl_id=com.example.test decl_type=configuration enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=retrieved declaration
```

And indeed in the above logs the enrollment has requested the "com.example.test" declaration (of type "configuration").

```
2022/08/08 15:19:28 level=info handler=log trace_id=ec46322eecd2b6af addr=::1 method=PUT path=/status agent=Go-http-client/1.1
2022/08/08 15:19:28 level=debug handler=status-report trace_id=ec46322eecd2b6af enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=status report decl_count=1 error_count=1 value_count=0
```

And finally the enrollment has given us a status update on the declarations that it has applied (`decl_count=1`). However it has also reported an error (`error_count=1`). What's that about? Let's investigate!

## Lets investigate

We saw an error, above, what is it?

### Tokens and Declaration-Items

First, let's check that what the enrollment is retrieving from the server looks correct as far as the Tokens, Declaration Items, and Declarations endpoints go:

```sh
$ ./tools/ddm-tokens.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "SyncTokens": {
    "DeclarationsToken": "ed534ed640fa03dc",
    "Timestamp": "2022-08-08T22:45:10Z"
  }
}
```

This helper script queries the same endpoint that an enrollment does when it talks the DDM protocol (note the `ddm-*` prefix of the script). By using the same endpoint we can see exactly what a response *for a given enrollment ID* might look like. The response to the tokens endpoint itself is not much to look at but what about the Declarations Items endpoint?

```sh
$ ./tools/ddm-declaration-items.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "Declarations": {
    "Activations": [],
    "Assets": [],
    "Configurations": [
      {
        "Identifier": "com.example.test",
        "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
      }
    ],
    "Management": []
  },
  "DeclarationsToken": "ed534ed640fa03dc"
}
```

Ah, here we see that the declaration items for this enrollment (in exactly the way the enrollment would query it) indeed lists our "com.example.test" declaration that we uploaded and assigned to a set and assigned to our enrollment, above. Also the `DeclarationsToken` matches for both it and the tokens endpoint. Looks good. Let's make sure the declaration that the client sees also matches:

```sh
$ ./tools/ddm-declaration.sh 2FF3196C-CACE-4AFE-9918-01C38160006F configuration/com.example.test | jq .
{
  "Type": "com.apple.configuration.management.test",
  "Payload": {
    "Echo": "KMFDDM!"
  },
  "Identifier": "com.example.test",
  "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
}
```

This looks good too and is, again, the way the enrollment would request this declaration from the server itself. The "configuration/com.example.test" matches the URL request the enrollment sent from URL in the logs, above. The `ServerToken` matches for both, and the payload, type, and identifier all look good.

### Status Values

When the enrollment first sent us its status report it sent along extra data. This data is commonly enrollment (and/or device) related and can include details about the particulars of its DDM support (like supported declaration types, status subscriptions, etc.). KMFDDM keeps note of this. We can query it like so:

```sh
$ ./tools/api-status-values-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "path": ".StatusItems.management.client-capabilities.supported-payloads.status-items",
      "value": "device.model.marketing-name"
    },
...
```

A "normalized" set of all the non-error, non-declaration data from the enrollment's status reports is returned. Since KMFDDM is agnostic to this data it just tries to store it for later retrieval — it does not act on it (that's up to you and what you want to place into your own declarations). The `path` element describes the location in the original status structure that it was found.

We can also query for specific path items, too (the trailing percent `%` character is passed onto an SQL `LIKE` condition):

```sh
$ ./tools/api-status-values-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F '.StatusItems.device.operating-system.%' | jq .
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "Path": ".StatusItems.device.operating-system.build-version",
      "Value": "22A5311f"
    },
    {
      "Path": ".StatusItems.device.operating-system.family",
      "Value": "macOS"
    },
    {
      "Path": ".StatusItems.device.operating-system.version",
      "Value": "13.0"
    }
  ]
}
```

This was a fun aside but doesn't particularly help us with tracking down our error, so lets move on.

### Declaration Status

DDM-enabled enrollments send the status of their declarations with their status reports. KMFDDM keeps note of this. We can query KMFDDM for the collection of declarations that are assigned to an enrollment (via sets) and the status (if any) that the enrollment has reported back to us. Let's check that out:

```sh
$ ./tools/api-status-declaration-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "identifier": "com.example.test",
      "active": false,
      "valid": "unknown",
      "server-token": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f",
      "current": true,
      "status_received": "2022-08-08T22:19:28Z",
      "reasons": [
        {
          "code": "Info.NotReferencedByActivation",
          "description": "Configuration “com.example.test:cfc394dc3f39b8da909bdf1b2cae3a22a405e49f” is not referenced by an activation.",
          "details": {
            "Identifier": "com.example.test",
            "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
          }
        }
      ]
    }
  ]
}
```

Aha! What's returned here is an object literal of enrollment IDs. The enrollment ID object literal contains an array literal of declarations that are both assigned to this enrollment ID and that we've received some status back from the enrollment in a status report. We can tell the enrollment has the "current" declaration because the reported server-token is the same as the declaration's server-token (and we report this here with the `"current": true` line). But we can see that `"active": false` and `"valid": "unknown` and the enrollment has given us `"reasons": []` in the status report on what any problems might be with our "com.example.test" declaration.

It tells us plainly: it is not referenced by an activation. You see I led you astray by setting up a declaration for an enrollment without first including that declaration in an activation. In the DDM protocol configuration declarations need to be "activated" by activation declarations. If that was confusing then please watch Apple's video referenced above in this guide. If that was mean: I apologize. 😉 But I hoped to be able to walk you through troubleshooting your declarations and how they're reported by the client. We'll fix it in a sec.

### Status Errors

If a declaration is reported as neither active nor valid then we also append that as an *error* to the enrollment's error log along with the path we saw the error. So we can see that error here:

```sh
$ ./tools/api-status-errors-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "path": ".StatusItems.management.declarations.configurations",
      "error": {
        "active": false,
        "identifier": "com.example.test",
        "reasons": [
          {
            "code": "Info.NotReferencedByActivation",
            "description": "Configuration “com.example.test:cfc394dc3f39b8da909bdf1b2cae3a22a405e49f” is not referenced by an activation.",
            "details": {
              "Identifier": "com.example.test",
              "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
            }
          }
        ],
        "server-token": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f",
        "valid": "unknown"
      },
      "timestamp": "2022-08-08T22:19:28Z"
    }
  ]
}
```

Other errors can also be in the error log as reported in status reports, too. But this is the only error we see now.

So, what do we do about our unreferenced delcaration?  We make an activation that references it, of course!

## Activate a declaration

### Our second declaration

Let's use `ideclr.py` again. We know our existing declaration is called "com.example.test" so let's specify that declaration to activate:

```sh
$ ./tools/ideclr.py -i com.example.act activation com.example.testt
{
    "Type": "com.apple.activation.simple",
    "Payload": {
        "StandardConfigurations": [
            "com.example.testt"
        ]
    },
    "Identifier": "com.example.act"
}
```

We used "com.example.act" as the identifier for this declaration. This is a trivial activation that 'activates' a single declaration. Let's upload it!

```sh
$ ./tools/ideclr.py -i com.example.act activation com.example.testt | ./tools/api-declaration-put.sh -
{"error":"Error 1452: Cannot add or update a child row: a foreign key constraint fails (`kmfddm`.`declaration_references`, CONSTRAINT `declaration_references_ibfk_2` FOREIGN KEY (`declaration_reference`) REFERENCES `declarations` (`identifier`))"}
Response HTTP Code: 500
```

Oops, what happend? Oh, I had a typo. Because KMFDDM's SQL schema maintains referential integrity (SQL foreign keys) with the declarations it references this declaration can't be added because it references a missing declaration — "com.example.testt" doesn't exit! Let's try again this time without the typo:

```sh
$ ./tools/ideclr.py -i com.example.act activation com.example.test | ./tools/api-declaration-put.sh -
Response HTTP Code: 204
$ ./tools/api-declarations-get.sh 
["com.example.act","com.example.test"]
```

Ah, much better. Now KMFDDM has two declarations and our new activation references our old one. Let's look at our enrollment ID's declarations items to make sure it'll see this declaration:

```sh
$ ./tools/ddm-declaration-items.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "Declarations": {
    "Activations": [],
    "Assets": [],
    "Configurations": [
      {
        "Identifier": "com.example.test",
        "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
      }
    ],
    "Management": []
  },
  "DeclarationsToken": "ed534ed640fa03dc"
}
```

We only have our old declaration in there. Why is that? Another clue is that the `DeclarationsToken` has not changed (i.e. the enrollment will not consider any changes to its declarations).

### Our "second" set

Declarations can exist by themsleves unassigned to any anything. And they'll be just that — floating out there; disconnected. Like we initially did we need to assign declarations to sets. Sets also need to be assigned to enrollments — but we already did that, right? Our enrollment ID is already associated with the "default" set, isn't it? Let's answer that question:

```sh
$ ./tools/api-enrollment-sets-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F
["default"]
```

Yes, the enrollment ID is associated with the "default" set. Good.

What declarations are in that set?

```sh
$ ./tools/api-set-declarations-get.sh default
["com.example.test"]
```

Only our "test" declaration. Now let's assign this new activation declaration to the set we already created:

```sh
$ ./tools/api-set-declarations-put.sh default com.example.act
Response HTTP Code: 204
```

Cool, now if we double-check the declaration items for this enrollment we should see now:

```sh
$ ./tools/ddm-declaration-items.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "Declarations": {
    "Activations": [
      {
        "Identifier": "com.example.act",
        "ServerToken": "b52d83bd680fec2608059723b86f5e89efc1323d"
      }
    ],
    "Assets": [],
    "Configurations": [
      {
        "Identifier": "com.example.test",
        "ServerToken": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f"
      }
    ],
    "Management": []
  },
  "DeclarationsToken": "738e76cddb3a796e"
}
```

Yes, good, there's our new activation! Also notice in the server logs again:

```
2022/08/08 22:49:34 level=info handler=log trace_id=0f8f848fab2ae795 addr=::1 method=PUT path=/v1/set-declarations/default agent=curl/7.64.1
2022/08/08 22:49:35 level=info service=notifier msg=sending command count=1 include tokens=true command_uuid=0ed9dd7d-31f2-4969-9797-09d06647def3 http_status=200
```

The above lines have our API call which assigned the declaration to the set. KMFDDM figured out that doing this would require notifying any assigned enrollments and, indeed, it did so, getting a 200 (success) response. Meaning the enrollment will "automatically" be asked to check back into its declarations. The following lines show the enrollment doing that bu requesting its declaration items (as an aside it did not need to retrieve its tokens because those were sent along with the notification — so it skips right to the declaration items):

```
2022/08/08 22:49:41 level=info handler=log trace_id=49c4db24a5b964ba addr=172.22.0.60 method=GET path=/declaration-items agent=Go-http-client/1.1
2022/08/08 22:49:41 level=debug handler=declaration-items trace_id=49c4db24a5b964ba enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=retrieved declaration items
```

The enrollment sees the changed declarations and requests our new activation:

```
2022/08/08 22:49:41 level=info handler=log trace_id=d5e46f75c12ac168 addr=172.22.0.60 method=GET path=/declaration/activation/com.example.act agent=Go-http-client/1.1
2022/08/08 22:49:41 level=debug handler=declaration trace_id=d5e46f75c12ac168 decl_id=com.example.act decl_type=activation enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=retrieved declaration
```

Finally it sends a status report back. And what do we see?

```
2022/08/08 22:49:47 level=info handler=log trace_id=f8b5fd39f5f37430 addr=172.22.0.60 method=PUT path=/status agent=Go-http-client/1.1
2022/08/08 22:49:47 level=debug handler=status-report trace_id=f8b5fd39f5f37430 enroll_id=2FF3196C-CACE-4AFE-9918-01C38160006F msg=status report decl_count=2 error_count=0 value_count=0
```

The enrollment reports 2 declarations and 0 errors! Let's check our declaration status again, to see what's changed:

```sh
$ ./tools/api-status-declaration-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq . 
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "identifier": "com.example.act",
      "active": true,
      "valid": "valid",
      "server-token": "b52d83bd680fec2608059723b86f5e89efc1323d",
      "current": true,
      "status_received": "2022-08-09T05:49:47Z"
    },
    {
      "identifier": "com.example.test",
      "active": true,
      "valid": "valid",
      "server-token": "cfc394dc3f39b8da909bdf1b2cae3a22a405e49f",
      "current": true,
      "status_received": "2022-08-09T05:49:47Z"
    }
  ]
}
```

Ah, much better. We can see that both declarations have been reported back and both are all of active, valid, and current. This means that the enrollment and server are all in "sync" and all declarations are applied successfully. It's up to the client to apply those settings now.

## Inducing another error

One fun part of the "test" declaration is that it can purposely produce an error for us, for science. If we try to "re-upload" our existing test declaration:

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM!' | ./tools/api-declaration-put.sh -
Response HTTP Code: 304
```

We're just met with a 304 which does nothing and notifies no enrollments. Let's change it to include a return status:

```sh
$ ./tools/ideclr.py -i com.example.test test -r Failed 'KMFDDM!' | ./tools/api-declaration-put.sh -
Response HTTP Code: 204
$ ./tools/api-declaration-get.sh com.example.test | jq .                                  
{
  "Type": "com.apple.configuration.management.test",
  "Payload": {
    "Echo": "KMFDDM!",
    "ReturnStatus": "Failed"
  },
  "Identifier": "com.example.test",
  "ServerToken": "092b18d26c60523d014719685a663c1659a7ef34"
}
```

We received a 204 which means it was a success and the declaration changed. KMFDDM figured out that it needed to notify our enrollment ID (because it was assigned to a set which contains that declaration). The logs show us that the enrollment fetched the updated declaration and that it reported back it status. What does that status look like?

```sh
$ ./tools/api-status-declaration-get.sh 2FF3196C-CACE-4AFE-9918-01C38160006F | jq .
{
  "2FF3196C-CACE-4AFE-9918-01C38160006F": [
    {
      "identifier": "com.example.act",
      "active": true,
      "valid": "valid",
      "server-token": "b52d83bd680fec2608059723b86f5e89efc1323d",
      "current": true,
      "status_received": "2022-08-09T05:49:47Z"
    },
    {
      "identifier": "com.example.test",
      "active": true,
      "valid": "invalid",
      "server-token": "092b18d26c60523d014719685a663c1659a7ef34",
      "current": true,
      "status_received": "2022-08-09T06:07:29Z",
      "reasons": [
        {
          "code": "Error.ConfigurationCannotBeApplied",
          "description": "Configuration cannot be applied",
          "details": {
            "Error": "An internal error has occurred."
          }
        }
      ]
    }
  ]
}
```

This is an "artificial" error, of sorts, that the "com.example.test" declaration is able to generate for demonstration and testing purposes. We can change it back easily enough:

```sh
$ ./tools/ideclr.py -i com.example.test test 'KMFDDM!' | ./tools/api-declaration-put.sh - 
Response HTTP Code: 204
```

The declaration changed, the device was notified, and hopefully a status reported back with no errors.

Congratulations! You're managing enrollments *declaratively*!

## Next steps

- Create more declarations and activate them!
  - Like, say, activations that do *useful* things like install legacy profiles or configure CardDAV servers, for example.
  - Apple's maintains [documentation for declaration](https://developer.apple.com/documentation/devicemanagement/declarations)
- Play with the killer DDM feature: predicates!
  - To get you started, here's a purposefully failing predicate: `./tools/ideclr.py -i com.example.act activation -p '1==0' com.example.test`
  - A predicate that matches a serial number might look like: `@status(device.identifier.serial-number) == 'ZYXW4321'`
    - Note that you may need to include `device.identifier.serial-number` in a subscription management declaration for this to work. You can make one by doing: `./tools/ideclr.py -i com.example.sub subscription device.identifier.serial-number`. You'll also have to add this declaration to a set (and include it in an activation, of course).
  - You can create *arbtirary* predicate keys & values by creating a "properties" declaration: `./tools/ideclr.py properties` (then edit the object literal in the "Payload" key to include keys and values). This too, needs to be in a set. You can reference them in predicates using `@property(keyname)`.
    - Note that property declarations are *management* declarations and not *configuration* declarations. So they don't need to be included in an activation.
    - Because these declarations may include properties for *just* this enrollment it may go into a set that is only assigned to this enrollment. If so it may be to for the benefit of your sanity to name these sets and declarations something that includes the enrollment ID — `enrollment.2FF3196C-CACE-4AFE-9918-01C38160006F` for example — just as a breadcrumb that this is only intended for a single enrollment.
  - Apple talks more about predicates in their second DDM video for WWDC 2022: ["Adopt declarative device management"](https://developer.apple.com/videos/play/wwdc2022/10046/).
- Decide on a naming scheme for declarations and sets.
  - I picked the "reverse-dot" notation for declarations above, but it's whatever you want, really. Apple's original examples used UUIDs for declaration identifiers.
  - Stay away from forward-slashes or anything else that would cause trouble in a URL.
- Develop your own tools that create a manage declarations. Or manage them as files. Up to you!
- Figure out a system whereby DDM is enabled for your enrollments automatically. Something like:
  1. Send `DeviceInformation` MDM commands enrollments to figure out their OS version.
  1. Your webhook checks this MDM command response to see if they're a DDM-capable device.
  1. Uses the KMFDDM API to create declarations or add them to set(s).
  1. This will send a notification to the device and turn on DDM for them.
- A proper deployment
  - Behind HTTPS/proxies
  - Behind firewalls or in a private cloud/VPC
  - In a container environment like Docker, Kubernetes, etc. or even just running as a service with systemctl.
