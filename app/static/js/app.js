'use strict';

var oReaderApp = angular.module('oReaderApp', ['ngRoute', 'ngCookies', 'angular-storage']);

oReaderApp.config(function ($routeProvider, $locationProvider) {
    $routeProvider
        .when('/', {
            templateUrl: 'static/partials/welcome.html',
            controller: 'RootCtrl'
        })
        .when('/login', {
            controller: 'LoginCtrl'
        })
        .when('/logout', {
            controller: 'LogoutCtrl'
        })
        .when('/about', {
            templateUrl: 'static/partials/about.html',
            controller: 'AboutCtrl'
        })
        .otherwise({
            redirectTo: '/'
        });

    $locationProvider.html5Mode({
        enabled: true,
        requireBase: false
    });
});