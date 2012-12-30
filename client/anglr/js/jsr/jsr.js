
// a console utility that can be embedded into an html page,
// useful for diagnostics when unable to run a real debugger,
// for example when creating content that is hosted in a 3rd
// party app

(function() {

    var memId = 'jsr_',
        jsrOptions,
        traceTarget_,
        consIn_,
        jsrWindow_,
        histSelect_,
        cmdSelect_,
        scrollMgr,

        callStack = [],
        catchingErrors = false,
        origFuncCall = Function.prototype.call,
        origFuncApply = Function.prototype.apply,
        origWindowError,

        cmdNames = [],
        cmds = [],
        cmdKeyListener,
        
        cfg = {
            winTop: '90px',
            winLeft: '30px',
            winW: '400px',
            winH: '400px',
            catchErrors: false,
            histMax: 100,
            histMemIdx: -1,
            display: 'block'
        },
        histCycleIdx = NaN,
        api;

    function catchErrors(toggleValue) {

        if (toggleValue === undefined) {

            return catchingErrors;

        } else {

            if (catchingErrors !== toggleValue) {

                cfg.catchErrors = toggleValue;
                saveCfg();
                catchingErrors = toggleValue;

                if (toggleValue) {

                    Function.prototype.call = funcCallWrapper;
                    Function.prototype.oCall = origFuncCall;
                    
                    Function.prototype.apply = funcApplyWrapper;
                    Function.prototype.oApply = origFuncApply;

                    origWindowError = window.onerror;
                    window.onerror = handleWindowError;

                } else {

                    Function.prototype.call = origFuncCall;
                    Function.prototype.apply = origFuncApply;
                    window.onerror = origWindowError;
                }
            }
            return api;
        }
    }

    function funcCallWrapper(thisArg) {
        var res;
        try {
            var toStr = fnToShortStr(this),
                callIdx = callStack.length,
                callInfo = {idx: callStack.length, fn: this},
                args;

            if (arguments.length > 1) {
                args = Array.prototype.slice.oApply(arguments, [1]);
                callInfo.args = args;
            }
            Array.prototype.unshift.oCall(callStack, callInfo);
            res = this.oApply(thisArg, args);
            Array.prototype.shift.oCall(callStack);
        } catch (x) {
            logError(x, Array.prototype.shift.oCall(callStack));
            throw x;
        }
        // not reached if an error was encountered bc throws x
        return res;
    }

    function funcApplyWrapper(thisArg, args) {
        var res;
        try {
            var toStr = fnToShortStr(this),
                callIdx = callStack.length;

            Array.prototype.unshift.oCall(callStack, {
                idx: callStack.length,
                fn: this,
                args: args
            });
            res = this.oApply(thisArg, args);
            Array.prototype.shift.oCall(callStack);
            // return res;
        } catch (x) {
            logError(x, Array.prototype.shift.oCall(callStack));
            throw x;
        }
        // not reached if an error was encountered bc throws x
        return res;
    }

    function logError(x, callInfo) {

        scrollMgr.notePos();
        traceTarget_.append('<div class="jsr_co_line_error"><div class="jsr_co_line_error_boundary"></div><div class="jsr_cons_errorIcon"></div><p class="jsr_co_errorInfo">Log Error:</p></div>');


        classyTrace([x], 'jsr_co_line_error');

        if (callInfo) {
            traceTarget_.append('<div class="jsr_co_line_error"><p class="jsr_co_errorInfo">Callstack data:</p></div>');
            traceExpanded(callInfo, 'jsr_co_line_error');
        }
        scrollMgr.refresh();
    }

    function handleWindowError() {

        traceTarget_.append('<div class="jsr_co_line_error"><div class="jsr_co_line_error_boundary"></div><div class="jsr_cons_errorIcon"></div><p class="jsr_co_errorInfo">Unhandled ERRROR:</p></div>');

        traceExpanded(arguments, 'jsr_co_line_error');
    }

    function trace() {
        return classyTrace(arguments);
    }

    function classyTrace(args, coClass) {

        var out;

        coClass = coClass || 'jsr_co_line';

        if (args.length) {
            
            if (args.length > 1) {
                out = Array.prototype.join.oCall(args, [' ']);
            } else if (typeof args[0] === 'string') {
                out = args[0];
            } else {
                out = render(args[0]);
            }
        } else if (typeof args === 'string') {

            out = args;

        } else if (args || args === 0) {

            out = render(args);
        }

        scrollMgr.notePos();
        traceTarget_.append( $('<div class="' + coClass + '"></div>').append(out) );
        scrollMgr.refresh(scrollMgr.bottomNoted, 0);
        return api;
    }

    function traceInput(value) {
        value = '<span class="jsr_cons_inVal"><span class="jsr_cons_inIcon"></span>' + value + '</span>';
        traceTarget_.append('<div class="jsr_co_line_input"><div class="jsr_co_line_input_boundary"></div><p class="jsr_co_basic">' + value + '</p></div>');
    }

    function traceExpanded(data, coClass) {

        scrollMgr.notePos();
        var div_ = $('<div class="' + (coClass || 'jsr_co_line') + '"></div>').append(render(data)),
            type = $.type(data);

        if (type === 'function' ||
            type === 'object' ||
            type === 'array') {

            div_.append(renderDeep(data, 0));
        }

        traceTarget_.append(div_);
        // scrollMgr.refresh();
        scrollMgr.refresh(scrollMgr.bottomNoted, 0);
    }

    function render(value, propName, elmDepth) {

        elmDepth = elmDepth ? elmDepth : 0;

        var endProp,
            type = $.type(value),
            depthStyle = 'jsr_co_propLvl' + elmDepth,
            fnTag,
            fnA;


        if (propName) {
            propName = '<span class="jsr_propName">' + propName + ': </span>';
            endProp = '</span>';
        } else {
            propName = '';
            endProp = '';
        }

        switch ($.type(value)) {
            case 'object':
                return $('<a href="#" class="jsr_consExpander ' + depthStyle + '"></a><a href="#" class="jsr_objectLink">' +
                            propName + '<span>' + value + '</span></a>')
                        .on('click', [value, elmDepth], dataClicked);

            case 'function':
                return $('<a href="#" class="jsr_consExpander ' + depthStyle + '"></a><a href="#" class="jsr_objectLink">' +
                            propName + '<span>' + fnToShortStr(value) + '</span></a>')
                        .on('click', [value, elmDepth], dataClicked);

            case 'array':
                var tos = value.toString();
                tos = propName + '<span>[' + tos.substr(0, 40) + (tos.length > 40 ? '...' : '') + ']' + endProp;
                return $('<a href="#" class="jsr_consExpander ' + depthStyle + '"></a><a href="#" class="jsr_objectLink">' + tos + '</a>')
                        .on('click', [value, elmDepth], dataClicked);

            case 'undefined':
                return propName + '<span>undefined</span>';

            case 'null':
                return propName + '<span>null</span>';

            case 'string':
                return propName + '<span>' + '"' + value + '"</span>';

            case 'date':
                return propName + '<span>' + value + '</span>';

            case 'regexp':
                return propName + '<span>' + value + '</span>';

            case 'boolean':
                return propName + '<span>' + value + '</span>';

            case 'number':
                return propName + '<span>' + value + '</span>';
        }
    }

    function renderDeep(data, depth) {
        var ul_ = $('<ul class="jsr_co_propList"></ul>');
        var li_;

        if ($.type(data) === 'function') {
            li_ = $('<li class="jsr_co_fnBody cm-s-default"></li>');
            CodeMirror.runMode(data.toString(), 'text/javascript', li_[0]);
            ul_.append(li_);
        }

        for (var p in data) {
            ul_.append( $('<li class="jsr_co_oProp"></li>').append(render(data[p], p, depth + 1)) );
        }
        ul_.addClass('jsr_co_zLvl' + depth);
        return ul_;
    }

    function fnToShortStr(fn) {
        var toStr = fn.toString.oCall(fn);
        toStr = toStr.replace.oCall(toStr, /\n+|\t+|\r+| {2,}/gm, ' ');
        var cut = toStr.length > 50;
        toStr = cut ? toStr.substr.oCall(toStr, 0, toStr.lastIndexOf.oCall(toStr, ' ', 50)) : toStr;
        toStr = toStr.trim.oCall(toStr);
        return cut ? toStr + '...' : toStr;
    }

    function dataClicked(event) {

        var uls_ = $(this).parent().find('> ul'),
            data,
            depth;

        if (uls_.length) {

            uls_.toggle();

        } else {

            data = event.data[0];
            depth = event.data[1];
            $(this).parent().append( renderDeep(data, depth));
        }
        return false;
    }

    function seeStackClone() { return callStack.slice(); }

    function cmdSelected(event) {
        var select_ = $(event.currentTarget);
        var val = select_.find('>option:selected').val();
        if (val === '@na') {
            return;
        }
        var consVal = consIn_.val().trimRight();
        select_.val('@na');

        consIn_.val(val);
        consIn_.focus();
    }

    function addCmd(alias, fn, addSnippet) {
        var idx = cmdNames.indexOf(alias);
        if (idx < 0) {
            cmdNames.push(alias);
            cmds.push(fn);
            if (addSnippet) {
                cmdSelect_.prepend(
                    $('<option></option>').attr('value', alias).text(alias)
                );
            }
        } else {
            cmds[idx] = fn;
        }
        return api;
    }

    function addSnippet(value) {

        var existing = cmdSelect_.find('option [value="' + value + '"]');
        if (!existing.length) {
            cmdSelect_.prepend(
                $('<option></option>').attr('value', value).text(value)
            );
        }
        return api;
    }

    function isVisible() {
        return jsrWindow_.css('display') === 'block';
    }

    function setVis(value) {
        var toDisplay = value ? 'block' : 'none';
        jsrWindow_.css({display: toDisplay});
        cfg.display = toDisplay;
        saveCfg();
    }

    function cmd_vis(value) {
        if (value !== undefined) {
            value = value === true || value === 'true';
            setVis(value);
            return api;
        }
        var cur = isVisible();
        trace('Current visibility: ' + cur);
        return cur;
    }

    function cmd_cls() {
        // note: remove() also removes event listeners and releases references to the data used by them
        traceTarget_.find('*').remove();
        return 'cls';
    }

    function cmd_errs(val) {
        if (val !== undefined) {
            catchErrors(val === 'true');
        }
        trace('Error catching enabled: ' + catchErrors());
        return true;
    }

    // scroll command
    // when invoked start listening for arrow input from keypress
    // and scrolls accordingly, end on enter
    var cmd_scroll = (function () {

        var shiftPct = 0.2,
            shiftSize,
            scrollArrowListener = function scrollArrowListener(event) {

                // holding shift increasing scroll amount
                var shiftAmt = shiftSize * (event.shiftKey ? 5 : 1);

                // if is left right down or up, do something;
                switch (event.keyCode) {
                    // enter key
                    case 13:
                        cmdKeyListener = null;
                        jsr('Scroll command exit.');
                        return false;

                    // left arrow
                    case 37:
                        scrollMgr.scrollTo(-shiftAmt, 0, 300, true);
                        return true;

                    // right arrow
                    case 39:
                        scrollMgr.scrollTo(shiftAmt, 0, 300, true);
                        return true;

                    // down arrow
                    case 40:
                        scrollMgr.scrollTo(0, shiftAmt, 300, true);
                        return true;

                    // up arrow
                    case 38:
                        scrollMgr.scrollTo(0, -shiftAmt, 300, true);
                        return true;

                    // for all other keys, continue within command
                    default:
                        return true;
                }
            };

        return function scrollCommand() {

            // start listening for arrow keys, end listening on enter
            jsr('Use the arrow keys to sroll the window.<br/>Hold shift to scroll faster.<br/>Press enter to exit command.');
            cmdKeyListener = scrollArrowListener;
            shiftSize = scrollMgr.getViewHeight() * shiftPct;
            // return false tells not to scroll to bottom
            // (dont want to lose current position)
            return false;
        };
    })();



    // size command
    // when invoked can be passed arguments stating a new size
    // if numeric values are surrounded by quotes then size is relative.
    // if invoked without arguments listens for arrow input from keypress
    // and moves accordingly

    var cmd_size = (function () {

        var shiftSize = 5,
            sizeArrowListener = function sizeArrowListener(event) {

                // holding shift increases size change to 5x
                var shiftAmt = shiftSize * (event.shiftKey ? 5 : 1);

                // if is left right down or up, do something;
                switch (event.keyCode) {
                    // enter key
                    case 13:
                        cmdKeyListener = null;
                        cfg.winW = jsrWindow_.width();
                        cfg.winH = jsrWindow_.height();
                        saveCfg();
                        jsr('Size command exit.');
                        return false;

                    // left arrow
                    case 37:
                        jsrWindow_.width(jsrWindow_.width() - shiftAmt);
                        scrollMgr.refresh();
                        return true;

                    // right arrow
                    case 39:
                        jsrWindow_.width(jsrWindow_.width() + shiftAmt);
                        scrollMgr.refresh();
                        return true;

                    // down arrow
                    case 40:
                        jsrWindow_.height(jsrWindow_.height() + shiftAmt);
                        scrollMgr.refresh();
                        return true;

                    // up arrow
                    case 38:
                        jsrWindow_.height(jsrWindow_.height() - shiftAmt);
                        scrollMgr.refresh();
                        return true;

                    // for all other keys, continue within command
                    default:
                        return true;
                }
            };

        return function sizeCommand() {

            var arg,
                w,
                h;

            // if supplied with arguments on new size, then don't listen for arrow keys
            if (arguments.length) {

                arg = arguments[0];

                if (arg.search(/"|'/g) > -1) {

                    arg = arg.replace(/"|'/g, '');
                    w = jsrWindow_.width() + Number(arg);
                    jsrWindow_.width(w);

                } else {

                    w = Number(arg);
                    jsrWindow_.width(w);
                }
                cfg.winW = w;

                if (arguments.length > 1) {

                    arg = arguments[1];

                    if (arg.search(/"|'/g) > -1) {

                        arg = arg.replace(/"|'/g, '');
                        h = jsrWindow_.height() + Number(arg);
                        jsrWindow_.height(h);

                    } else {

                        h = Number(arg);
                        jsrWindow_.height(h);
                    }
                    cfg.winH = h;
                }
                saveCfg();

            } else {

                // start listening for arrow keys, end listening on enter
                jsr('Use the arrow keys to resize the window.<br/>Hold shift for a larger change in size.<br/>Press enter to exit command.');
                cmdKeyListener = sizeArrowListener;
            }
            // return true tells to scroll to bottom
            return true;
        };
    })();



    // move command
    // when invoked can be passed arguments stating where to move
    // if numeric values are surrounded by quotes then move is relative.
    // if invoked without arguments listens for arrow input from keypress
    // and moves accordingly

    var cmd_move = (function () {

        var shiftSize = 5,
            moveArrowListener = function moveArrowListener(event) {

                var pos = jsrWindow_.offset(),
                    // holding shift increases move amount to 5x
                    shiftAmt = shiftSize * (event.shiftKey ? 5 : 1);

                // if is left right down or up, do something;
                switch (event.keyCode) {
                    // enter key
                    case 13:
                        cmdKeyListener = null;
                        cfg.winTop = pos.top;
                        cfg.winLeft = pos.left;
                        saveCfg();
                        jsr('Move command exit.');
                        return false;

                    // left arrow
                    case 37:
                        pos.left -= shiftAmt;
                        jsrWindow_.offset(pos);
                        return true;

                    // right arrow
                    case 39:
                        pos.left += shiftAmt;
                        jsrWindow_.offset(pos);
                        return true;

                    // down arrow
                    case 40:
                        pos.top += shiftAmt;
                        jsrWindow_.offset(pos);
                        return true;

                    // up arrow
                    case 38:
                        pos.top -= shiftAmt;
                        jsrWindow_.offset(pos);
                        return true;

                    // for all other keys, continue within command
                    default:
                        return true;
                }
            };

        return function moveCommand() {

            var arg,
                pos;

            // if supplied with arguments on where to move, then don't listen for arrow keys
            if (arguments.length) {

                pos = jsrWindow_.offset();
                arg = arguments[0];

                if (arg.search(/"|'/g) > -1) {

                    arg = arg.replace(/"|'/g, '');
                    pos.left += Number(arg);

                } else {

                    pos.left = Number(arg);
                }

                if (arguments.length > 1) {

                    arg = arguments[1];

                    if (arg.search(/"|'/g) > -1) {

                        arg = arg.replace(/"|'/g, '');
                        pos.top += Number(arg);
                    } else {
                        pos.top = Number(arg);
                    }
                }
                jsrWindow_.offset(pos);
                cfg.winTop = pos.top;
                cfg.winLeft = pos.left;
                saveCfg();

            } else {

                // start listening for arrow keys, end listening on enter
                jsr('Use the arrow keys to move the window.<br/>Hold shift to move further.<br/>Press enter to exit command.');
                cmdKeyListener = moveArrowListener;
            }
            // return true tells to scroll to bottom
            return true;
        };
    })();

    function initCommands() {
        addCmd('.mv', cmd_move);
        addCmd('.move', cmd_move, true);
        addCmd('.sz', cmd_size);
        addCmd('.size', cmd_size, true);
        addCmd('.scroll', cmd_scroll, true);
        addCmd('...', cmd_scroll);
        addCmd('.cls', cmd_cls, true);
        addCmd('.errs', cmd_errs, true);
        addCmd('.vis', cmd_vis, true);
    }

    function isString(input) {
        return typeof(input) === 'string';
    }

    function addToHist(cmd) {

        cfg.histMemIdx = (cfg.histMemIdx + 1) % cfg.histMax;
        hist.set(cfg.histMemIdx, cmd);
        saveCfg();

        histSelect_.find('option[value="@na"]').before(createHistOpt(cmd));

        var ops = histSelect_.find('option'),
            cut = ops.length - cfg.histMax - 1;

        if (cut > 0) {
            ops.filter(':lt(' + cut + ')').remove();
        }
    }

    function createHistOpt(cmd) {
        return $('<option></option>').attr('value', cmd).text(cmd);
    }

    function seeCfgClone() {
        return $.extend({}, cfg);
    }

    var mem = (function jsrLocalStorage() {

        var hadVal,
            sawLsErr = false,
            memFunc = function getSet(key, value) {
                if (value !== undefined) {
                    try {
                        return localStorage[memId + key] = value;
                    } catch (err) {
                        if (!sawLsErr) {
                            jsr('Unable to write to localStorage: ').deep(err);
                            sawLsErr = true;
                        }
                    }
                } else {
                    return localStorage[memId + key];
                }
            };

        memFunc.has = function has(k) {
            hadVal = localStorage[memId + k];
            return hadVal !== undefined;
        };

        memFunc.had = function had() { return hadVal; };

        return memFunc;
    })();


    var hist = (function jsrCommandHistory(){

        var hadVal,
            histFunc = function get(idx) {
                return mem('hist' + idx);
            };

        histFunc.get = histFunc;

        histFunc.has = function has(idx) {
            return mem.has('hist' + idx);
        };

        histFunc.had = function had() {
            return mem.had();
        };

        histFunc.set = function set(idx, val) {
            mem('hist' + idx, val);
            return idx;
        };

        histFunc.memPos = function memPos(value) {
            if (value !== undefined && !isNaN(value)) {
                cfg.histMemIdx = value;
                saveCfg();
            }
            return cfg.histMemIdx;
        };

        histFunc.max = function max(value) {
            if (value !== undefined && !isNaN(value)) {
                cfg.histMax = value;
                saveCfg();
            }
            return cfg.histMax;
        };

        return histFunc;
    })();

    function setCfg(a, b) {
        if ($.type(a) === 'object') {
            for (var p in a) {
                cfg[p] = a[p];
            }
            saveCfg();
        } else if ($.type(a) === 'string') {
            cfg[a] = b;
            saveCfg();
        }
        return api;
    }

    function clearCfg(confirm) {
        if (confirm === true || confirm === 'true') {
            for (var p in cfg) {
                delete cfg[p];
            }
            saveCfg();
        }
        return api;
    }

    function initCfg() {
        if (mem.has('cfg')) {
            var fromMem = JSON.parse(mem.had());
            for (var p in fromMem) {
                cfg[p] = fromMem[p];
            }
        }

        // toggle catch errors if is set to true
        catchErrors(cfg.catchErrors);

        // history keeps last cfg.histMax # of commands
        // this is stored in local storage in props named
        // <memId>hist0 to <memId>hist<cfg.histMax>
        // this is cycled through so only cfg.histMax values are ever stored
        // in local storage. When cfg.histMemIdx reaches the max it is reset
        // to 0 and starts to save there. At any point, if there is a value
        // stored at cfg.histMemIdx+1, that is the oldest value
        var i,
            pos,
            naOpt_,
            history = [];

        // validate current index:
        cfg.histMemIdx %= cfg.histMax;

        for (i = 0; i < cfg.histMax; i++) {
            if (hist.has(i)) {
                history[i] = hist.had();
            }
        }
        if (history.length) {

            naOpt_ = histSelect_.find('option[value="@na"]');

            // add oldest values first, ie:
            // add from cfg.histMemIdx+1 to last entry in stored values
            for (i = cfg.histMemIdx + 1; i < history.length; i++) {
                naOpt_.before(createHistOpt(history[i]));
            }
            // then add from 0 to cfg.histMemIdx
            for (i = 0; i <= cfg.histMemIdx && hist.has(i); i++) {
                naOpt_.before(createHistOpt(history[i]));
            }
        }
    }

    function saveCfg() {
        mem('cfg', JSON.stringify(cfg));
    }

    var sizeBound = (function(){
        var minW = 220,
            defaultW = 600,
            minH = 135,
            defaultH = 600;

        return {
            w: function(value) {
                return !isNaN(value) ? Math.max(minW, value) : defaultW;
            },
            h: function(value) {
                return !isNaN(value) ? Math.max(minH, value) : defaultH;
            }
        };

    })();

    function initWindowDrag() {

        var topOffset = 0,
            leftOffset = 0,
            jsrWindow_ = $('#jsr'),
            chromeTop_ = $('#jsr_chrome_top');

        var startWindowDrag = function(e) {
            var pos = jsrWindow_.offset();
            topOffset = e.originalEvent.changedTouches[0].pageY - pos.top;
            leftOffset = e.originalEvent.changedTouches[0].pageX - pos.left;
            jsrWindow_.css('opacity', 0.4);
        };

        var dragWindow = function(e) {
            e.preventDefault();
            var toTop = e.originalEvent.changedTouches[0].pageY - topOffset;
            var toLeft = e.originalEvent.changedTouches[0].pageX - leftOffset;
            jsrWindow_.offset({  top: toTop, left: toLeft  });
        };

        var dragEnd = function(e) {
            var pos = jsrWindow_.offset();
            cfg.winTop = pos.top - $(window).scrollTop();
            cfg.winLeft = pos.left - $(window).scrollLeft();
            jsrWindow_.css('opacity', 1);
            saveCfg();
        };

        var onMouseDown = function(e) {
            $('body')
                .on("mousemove", onMouseMove)
                .on("mouseup", onMouseUp);

            e.originalEvent.changedTouches = [e.originalEvent];
            startWindowDrag(e);
        };

        var onMouseMove = function(e) {
            e.originalEvent.changedTouches = [e.originalEvent];
            dragWindow(e);
        };

        var onMouseUp = function(e) {
            $('body')
                .off("mousemove", onMouseMove)
                .off("mouseup", onMouseUp);
                
            dragEnd(e);
        };

        chromeTop_
            .on("touchstart", startWindowDrag)
            .on("touchmove", dragWindow)
            .on("touchend", dragEnd)
            .on("touchcancel", dragEnd)
            .on("mousedown", onMouseDown);
    }

    function initResizeDrag() {

        var wOffset = 0,
            hOffset = 0,
            jsrWindow_ = $('#jsr'),
            resizeHandle_ = $('#resizeHandle');

        var startWindowResize = function(e) {
            e.preventDefault();
            wOffset = e.originalEvent.changedTouches[0].pageX - jsrWindow_.width();
            hOffset = e.originalEvent.changedTouches[0].pageY - jsrWindow_.height();
        };

        var resizeWindow = function(e) {
            e.preventDefault();
            var w = sizeBound.w(e.originalEvent.changedTouches[0].pageX - wOffset);
            var h = sizeBound.h(e.originalEvent.changedTouches[0].pageY - hOffset);
            jsrWindow_.width(w);
            jsrWindow_.height(h);
            scrollMgr.refresh();
        };

        var resizeEnd = function(e) {
            e.preventDefault();
            cfg.winW = jsrWindow_.width();
            cfg.winH = jsrWindow_.height();
            saveCfg();
        };

        var onMouseDown = function(e) {

            $('body')
                .on("mousemove", onMouseMove)
                .on("mouseup", onMouseUp);

            e.originalEvent.changedTouches = [e.originalEvent];
            startWindowResize(e);
        };

        var onMouseMove = function(e) {
            e.originalEvent.changedTouches = [e.originalEvent];
            resizeWindow(e);
        };

        var onMouseUp = function(e) {

            $('body')
                .off("mousemove", onMouseMove)
                .off("mouseup", onMouseUp);

            resizeEnd(e);
        };

        resizeHandle_
            .on("touchstart", startWindowResize)
            .on("touchmove", resizeWindow)
            .on("touchend", resizeEnd)
            .on("touchcancel", resizeEnd)
            .on("mousedown", onMouseDown);
    }

    /*
    =========
    init core
    =========
    */
    (function() {

        var isiPad = navigator.userAgent.match(/iPad/i) !== null,
            icon_,
            // need something quick and dirty for now
            jsrHtml =  '<div id="jsr"' + (isiPad ? ' class="iPad">' : ' >') +
                            '<div id="jsr_scroller_wrap">' +
                                '<div id="jsr_scroller">' +
                                    '<div id="jsr_scroller_inr">' +
                                        '<div id="jsr_cons_out"></div>' +
                                    '</div>' +
                                '</div>' +
                            '</div>' +
                            '<div id="jsr_cons_in_chrome">' +
                                '<div id="jsr_cons_in_wrap">' +
                                    '<input id="jsr_cons_in" type="text" autocapitalize="off" autocorrect="off" autocomplete="off" />' +
                                    '<div id="jsr_cons_ps1" class="jsrSprite" ></div>  ' +
                                '</div>' +
                                '<div id="historySelectGraphic"></div>' +
                                '<select id="historySelect" required="false">' +
                                    '<option value="@na" selected="selected">Command history:</option>' +
                                '</select>' +
                                '<select id="jsr_cmd_list" required="false">' +
                                    '<option value="@na" selected="selected">Quick C̷̙̲̝͖ͭ̏ͥͮ͟Oͮ͏̮̪̝͍M̲̖͊̒ͪͩͬ̚̚͜M̖͊̒ͬ̚̚͜A̡͊͠͝N̐DS̨̥̫͎̭ͯ̿̔̀ͅ:</option>' +
                                    '<option value=".cls">.cls</option>' +
                                    '<option value="localStorage">localStorage</option>' +
                                    '<option value="jsr(">jsr(</option>' +
                                '</select>' +
                                '<div id="resizeHandle"><span id="resizeHandleIcon"></span></div>' +
                            '</div>' +
                            '<script id="jsr_script" type="text/javascript"></script>' +
                        '<a href="#" onclick="return false;" id="jsr_chrome_top">' +
                            '<div id="jsr_close"></div>' +
                        '</a>' +
                    '</div>',
            jsrIconHtml = '<div id="jsr_icon"></div>',
            prop;

        jsrOptions = {
            id: 'gbl_',
            useiScroll: true
        };
        if (typeof window.jsrOptions === 'string') {
            jsrOptions.id = window.jsrOptions;
            memId = 'jsr_' + window.jsrOptions + '_';
        } else if (window.jsrOptions) {
            for (prop in window.jsrOptions) {
                jsrOptions[prop] = window.jsrOptions[prop];
            }
            memId = 'jsr_' + (jsrOptions.id !== undefined ? jsrOptions.id + '_' : 'gbl_');
        }
        

        jsrWindow_ = $(jsrHtml);
        icon_ = $(jsrIconHtml);

        // add window and icon to the dom
        $('body')
            .append(jsrWindow_)
            .append(icon_);

        histSelect_ = $('#historySelect').on('change', cmdSelected),
        cmdSelect_ = $('#jsr_cmd_list').on('change', cmdSelected),
        traceTarget_ = jsrWindow_.find('#jsr_cons_out'),
        consIn_ = jsrWindow_.find('#jsr_cons_in');

        // to avoid using own error trapping
        Function.prototype.oCall = origFuncCall;
        Function.prototype.oApply = origFuncApply;

        initCfg();
        initCommands();
        initWindowDrag();
        initResizeDrag();

        jsrWindow_.css( {
            top: cfg.winTop,
            left: cfg.winLeft,
            width: cfg.winW,
            height: cfg.winH,
            display: cfg.display
        });

        icon_.add('#jsr_close').click(function(event) {
            setVis(!isVisible());
        });

        consIn_.keydown(function (e) {

            var val,
                vArgs,
                cmdIdx,
                res,
                this_ = $(this);

            // a command is listening to keyboard input,
            // when passed the event, it will return true if should continue listening
            if (cmdKeyListener) {
                if (!cmdKeyListener(e)) {
                    cmdKeyListener = null;
                }
                return;
            } else if (e.keyCode === 13) {

                val = this_.val().trim();

                if (val === '') {
                    return;
                }

                this_.val('');
                traceInput(val);
                scrollMgr.refresh(true, 0);
                addToHist(val);
                histCycleIdx = NaN;

                vArgs = val.split(' ');
                cmdIdx = cmdNames.indexOf(vArgs[0]);

                if (catchingErrors) {
                    if (cmdIdx > -1) {
                        fn = cmds[cmdIdx];
                        res = fn.oApply(window, vArgs.slice(1));
                        if (res) {
                            scrollMgr.refresh(true);
                        }
                    } else {
                        res = eval.oCall(window, val);
                        if (res !== api || val === 'jsr') {
                            trace(res);
                        }
                        scrollMgr.refresh(true);
                    }
                    
                } else {
                    try {
                        if (cmdIdx > -1) {
                            fn = cmds[cmdIdx];
                            res = fn.oApply(window, vArgs.slice(1));
                            if (res) {
                                scrollMgr.refresh(true);
                            }
                        } else {
                            res = eval.oCall(window, val);
                            if (res !== api || val === 'jsr') {
                                trace(res);
                                // traceChain(res);
                            }
                            scrollMgr.refresh(true);
                        }
                    } catch (x) {
                        logError(x);
                        throw x;
                    }
                }
            } else if (e.keyCode === 40) {
                // down arrow was pressed, cycle through history
                if (isNaN(histCycleIdx)) {
                    return;
                }
                if (histCycleIdx !== cfg.histMemIdx) {
                    histCycleIdx = ++histCycleIdx % cfg.histMax;
                    this_.val(hist(histCycleIdx));
                } else {
                    histCycleIdx = NaN;
                    // reload previous input
                    this_.val(this_.data('prevInput') || '');
                }
           } else if (e.keyCode === 38) {
                // up arrow was pressed, cycle through history
                // before hist cycle, always at NaN
                if (isNaN(histCycleIdx)) {
                    histCycleIdx = cfg.histMemIdx;
                    if (hist.has(histCycleIdx)) {
                        // save anything typed for later
                        this_.data('prevInput', this_.val());
                        this_.val(hist.had());
                    } else {
                        // have no history
                        histCycleIdx = NaN;
                    }
                } else {
                    histCycleIdx = histCycleIdx === 0 ? cfg.histMax - 1 : histCycleIdx - 1;
                    if (hist.has(histCycleIdx)) {
                        this_.val(hist.had());
                    } else {
                        // reach the end of the history, so keep it where it was
                        histCycleIdx = ++histCycleIdx % cfg.histMax;
                    }
                }
            }
        });

        if (jsrOptions.useiScroll && window.iScroll) {

            scrollMgr = (function(){
                var scroller = new iScroll('jsr_scroller',
                    {
                        hScrollbar: false,
                        vScrollbar: true,
                        checkDOMChanges: true,
                        useTransform: false,
                        bounce: false
                    }),
                    api = {
                        bottomNoted: false,
                        notePos: function jsrNoteIScrollPosition() {
                            bottomNoted = scroller.y === scroller.maxScrollY;
                        },
                        refresh: function jsrRefreshIScroll(toBottom, duration) {
                            scroller.refresh();
                            // keep at bottom if already there or if should be forced to the bottom
                            // (for instance, if latest content traced was input from console)
                            if (scroller.maxScrollY < 0 && (bottomNoted || toBottom)) {
                                // if duration was supplied, replace 0 with undefined (no scroll animation)
                                // keep otherwise
                                if (!isNaN(duration)) {
                                    duration = duration === 0 ? undefined : duration;
                                } else {
                                    // default scroll duration is 0.2 seconds
                                    duration = 200;
                                }
                                scroller.scrollTo(0, scroller.maxScrollY, duration);
                            }
                            bottomNoted = false;
                        },
                        scrollTo: function(x,y,dur,rel) { scroller.scrollTo(x,y,dur,rel); },
                        getViewHeight: function() { return scroller.wrapperH; }
                    };
                return api;
            })();
        } else {
            scrollMgr = (function(){
                var scrollElm = document.getElementById('jsr_scroller'),
                    scrollAnim = (function jsrScrollAnimMgr() {
                        var intervalY = null,
                            intervalBoth = null,
                            startTime = -1,
                            dur = -1,
                            fromTop = -1,
                            toTop = -1,
                            chgTop = -1,
                            fromLeft = -1,
                            toLeft = -1,
                            chgLeft = -1,
                            updateY = function() {
                                var elapsed = Date.now() - start;
                                if (elapsed >= dur) {
                                    scrollElm.scrollTop = toTop;
                                    stopY();
                                } else {
                                    scrollElm.scrollTop = easeOutCirc(elapsed, fromTop, chgTop, dur);
                                }
                            },
                            updateBoth = function() {
                                console.log('update both: ' + (Date.now() - start) / dur);
                                var elapsed = Date.now() - start;
                                if (elapsed >= dur) {
                                    scrollElm.scrollTop = toTop;
                                    scrollElm.scrollLeft = toLeft;
                                    stopBoth();
                                } else {
                                    scrollElm.scrollTop = easeOutCirc(elapsed, fromTop, chgTop, dur);
                                    scrollElm.scrollLeft = easeOutCirc(elapsed, fromLeft, chgLeft, dur);
                                }
                            },
                            easeOutCirc = function(time, begin, chg, duration) {
                                return chg * Math.sqrt(1 - (time = time / duration - 1) * time) + begin;
                            },
                            stopY = function() {
                                if (intervalY) {
                                    clearInterval(intervalY);
                                    intervalY = null;
                                }
                            },
                            stopBoth = function() {
                                if (intervalBoth) {
                                    clearInterval(intervalBoth);
                                    intervalBoth = null;
                                }
                            },
                            stop = function() {
                                stopBoth();
                                stopY();
                            },
                            go = function(scrollTo, duration) {
                                fromTop = scrollElm.scrollTop;
                                toTop = scrollTo;
                                chgTop = toTop - fromTop;
                                dur = duration;
                                start = Date.now();
                                stopBoth();
                                if (!intervalY) {
                                    intervalY = setInterval(updateY, 33);
                                }
                            },
                            // parameter order is set to match iScroll's scrollTo()
                            scrollTo = function(left, top, duration, isRelative) {
                                console.log('scroll to');
                                fromTop = scrollElm.scrollTop;
                                fromLeft = scrollElm.scrollLeft;
                                if (isRelative) {
                                    toTop = fromTop + top;
                                    chgTop = top;
                                    toLeft = fromLeft + left;
                                    chgLeft = left;
                                } else {
                                    toTop = top;
                                    chgTop = top - scrollElm.scrollTop;
                                    toLeft = left;
                                    chgLeft = left - scrollElm.scrollLeft;
                                }
                                dur = duration;
                                start = Date.now();
                                stopY();
                                if (!intervalBoth) {
                                    intervalBoth = setInterval(updateBoth, 33);
                                }
                            },
                            getViewHeight = function() {
                                return scrollElm.clientHeight;
                            };
                        return {
                            stop: stop,
                            go: go,
                            scrollTo: scrollTo,
                            getViewHeight: getViewHeight
                        };
                    })(),
                    api = {
                        bottomNoted: false,
                        notePos: function jsrNoteScrollPosition() {
                            bottomNoted = scrollElm.scrollHeight - scrollElm.scrollTop - scrollElm.clientHeight === 0;
                        },
                        refresh: function jsrRefreshScroll(toBottom, duration) {
                            // keep at bottom if already there or if should be forced to the bottom
                            // (for instance, if latest content traced was input from console)
                            if (bottomNoted || toBottom) {
                                // if duration is 0 then just set scroll position
                                if (duration === 0) {
                                    scrollAnim.stop();
                                    scrollElm.scrollTop = scrollElm.scrollHeight - scrollElm.clientHeight;
                                } else {
                                    duration = !isNaN(duration) ? duration : 200;
                                    scrollAnim.go(scrollElm.scrollHeight - scrollElm.clientHeight, duration);
                                }
                            }
                        },
                        scrollTo: scrollAnim.scrollTo,
                        getViewHeight: scrollAnim.getViewHeight
                    };
                scrollElm.style.overflow = 'auto';
                return api;
            })();
        }

        // if on the iPad, do fixed position hack:
        if (isiPad) {
            $(window).on('scroll', function jsrFixPosHack(evt){
                var top = document.documentElement.scrollTop || document.body.scrollTop,
                    left = document.documentElement.scrollLeft || document.body.scrollLeft;
                jsrWindow_.css('margin-top', top + 'px').css('margin-left', left + 'px');
            });
        } else {
            // reduce opacity when scrolling
            // (doesn't work on iPad bc infrequent scroll event)
            // var timeoutId = NaN;
            // $(window).on('scroll', function jsrOnScroll(evt) {
            //     var thisTimeoutId = NaN,
            //         toFn = function jsrOnScrollTimeout() {
            //             // if there wasn't another scroll event b4 timeout firing
            //             if (timeoutId === thisTimeoutId) {
            //                 jsrWindow_.css('opacity', 1);
            //                 timeoutId = NaN;
            //             }
            //         };
            //     thisTimeoutId = setTimeout(toFn, 500);
            //     timeoutId = thisTimeoutId;
            //     jsrWindow_.css('opacity', 0.4);
            // });
        }

        trace('jsr instance id: ' + jsrOptions.id);
        trace('jsr local storage id: ' + memId);
        trace('jsr config:');
        traceExpanded(cfg);

        api =  function jsr() { return trace.apply(null, arguments); };
        api.trace = trace;
        api.deep = function jsrDeep(data) { traceExpanded(data); };
        api.errors = catchErrors;
        api.seeStackClone = seeStackClone;
        api.addCmd = addCmd;
        api.addSnippet = addSnippet;
        api.seeCfgClone = seeCfgClone;
        api.setCfg = setCfg;
        api.clearCfg = clearCfg;
        api.vis = cmd_vis;
        api.options = jsrOptions;
        
        window.jsr = api;
    })();
    return api;
})();

