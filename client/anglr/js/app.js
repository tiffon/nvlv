// 'use strict';


// Declare app level module which depends on filters, and services
var myModule = angular.module('myApp', ['myApp.filters', 'myApp.services', 'myApp.directives']).
  factory('output', ['$window', function(win) {
    win.output = win.output || [];
    return win.output;
  } ]);

  myModule.factory('nvlvConnection', function() {
	var shinyNewServiceInstance;
	return shinyNewServiceInstance;
});

  // config(['$routeProvider', function($routeProvider) {
  //   $routeProvider.when('/view1', {templateUrl: 'partials/partial1.html', controller: MyCtrl1});
  //   $routeProvider.when('/view2', {templateUrl: 'partials/partial2.html', controller: MyCtrl2});
  //   $routeProvider.otherwise({redirectTo: '/view1'});
  // }]);
