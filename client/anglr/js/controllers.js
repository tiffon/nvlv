// 'use strict';

/* Controllers */

function AwesomeCtrl($scope, output, $rootScope) {


    var msgTmplGdb = {
            ctx: 'gdb',
            data: null
        },
        msgTmplSh = {
            ctx: 'sh',
            data: null
        },
        msgTmplCmd = {
            ctx: 'cmd',
            data: null
        },
        joinFn = Array.prototype.join,
        wsUri = "ws://localhost:12345/nvlv",
        msglog = "";


    $scope.output = output;
    
    $scope.refresh = function awsm_refresh() {};

    $rootScope.$on('out_upd', function() {
        this.output.push('test val');
    });

    // $scope.testYoSelf = function awsm_test() {
    //     output.push('we got mad tests, son');
    // };



    jsr.addCmd("-open", initNvlvConn);
    jsr.addCmd("-init", initGdbSsn);
    jsr.addCmd("-t", function(){ output.push('done'); $scope.$apply('refresh()');});
    jsr.addCmd("-", jsrSendCmd, true);
    jsr.addCmd("!", jsrSendSh, true);
    jsr.addCmd(":", jsrSendGdb, true);
    jsr.addSnippet(": /_assets/tools/gdb/gdb-7.5/build/gdb/gdb --interpreter mi dev_0");
    jsr.addSnippet(": source /usr/local/go/src/pkg/runtime/runtime-gdb.py");

    function initGdbSsn() {
        initNvlvConn();
        setTimeout(function() {
            jsrSendCmd("-gdb-start", "dev_0");
            jsrSendGdb("-break-insert", "main.main");
            jsrSendCmd("-gdb-run");
        }, 1000);
    }

    function jsrSendCmd() {
        msgTmplCmd.data = {cmd: arguments[0]};
        if (arguments.length > 1) {
            msgTmplCmd.data.args = Array.prototype.slice.apply(arguments, [1]);
        }
        doSend(JSON.stringify(msgTmplCmd), true);
    }

    function jsrSendSh() {
        msgTmplSh.data = {cmd: joinFn.call(arguments, [' '])};
        doSend(JSON.stringify(msgTmplSh), true);
    }

    function jsrSendGdb() {
        msgTmplGdb.data = {cmd: joinFn.call(arguments, [' '])};
        doSend(JSON.stringify(msgTmplGdb), true);
    }

    function outputAppend(data) {
        // output.unshift(data);
        $scope.noop();
        $scope.$apply('refresh()');
    }

    $scope.noop = function noop(data) {
        $scope.output.unshift(data);
        $scope.refresh();
        $rootScope.$emit('out_upd');
    };

    

    function initNvlvConn() {
        websocket = new WebSocket(wsUri);
        websocket.onopen = onOpen;
        websocket.onclose = onClose;
        websocket.onerror = onError;
        websocket.onmessage = onMessage;
    }

    function onOpen(evt) {
        jsr("CONNECTED");
        jsrSendSh("pwd");
    }

    function onClose(evt) {
        jsr("DISCONNECTED");
    }

    function onMessage(evt) {

        msglog += evt.data;

        var data = JSON.parse(evt.data),
            i = 0;

        if (data === null) {
            jsr('on message, JSON parsed is null, raw: ')(evt.data);
            return;
        }
        if (data.Ctx == "gdb") {
            data = data.Data;
            if (data.raw !== undefined) {
                outputAppend(data.raw);
                // jsr('raw gdb output:')(data.raw);
            }
            if (data.data) {
                data = data.data;
                if (data.hasOwnProperty("length") && data[0] !== undefined) {
                    for (i = 0; i < data.length; i++) {
                        jsr(data[i]);
                        // jsr.deep(data[i]);
                        // outputAppend(data[i]);
                    }
                }
            } else {
                jsr(data.Data.data);
                // jsr.deep(data.Data.data);
                // outputAppend(data.Data.data);
            }
            return;
        }
        data = data.Data;

        if (data.err !== undefined) {
            jsr.deep(data.err);
            outputAppend('error: -------------- \n' + data.err);
        } else {
            jsr.deep(data);
            outputAppend(data);
        }
    }

    function onError(evt) {
        jsr('sending [ ' + evt.data + ' ]');
    }

    function doSend(message, quiet) {
        websocket.send(message);
        if (!quiet) {
            jsr(">>  " + message);
        }
    }



    // (function init() {
    //     msglog = "";
    //     initNvlvConn();
    // })();

}
