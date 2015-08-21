'use strict';

var oReaderApp = angular.module('oReaderApp', ['ngResource', 'ngRoute', 'ngCookies', 'angular-storage']);

oReaderApp.config(function ($routeProvider, $locationProvider, $httpProvider, $resourceProvider) {
    $routeProvider
        .when('/', {
            templateUrl: 'static/partials/welcome.html',
            controller: 'RootCtrl',
            loginRequired: false
        })
        .when('/login', {
            templateUrl: 'static/partials/login.html',
            controller: 'LoginCtrl',
            loginRequired: false
        })
        .when('/logout', {
            controller: 'LogoutCtrl',
            templateUrl: 'static/partials/welcome.html',
            loginRequired: true
        })
        .when('/about', {
            templateUrl: 'static/partials/about.html',
            controller: 'AboutCtrl',
            loginRequired: false
        })
        .when('/feeds', {
            templateUrl: 'static/partials/feeds.html',
            controller: 'FeedCtrl',
            loginRequired: true
        })
        .when('/addSubscription', {
            templateUrl: 'static/partials/add_subscription.html',
            controller: 'SubscriptionCtrl',
            loginRequired: true
        })
        .otherwise({
            redirectTo: '/'
        });



    //$locationProvider.html5Mode({
    //    enabled: true,
    //    requireBase: false
    //});

    $resourceProvider.defaults.stripTrailingSlashes = false;

    $httpProvider.interceptors.push('tokenInjector');
    $httpProvider.interceptors.push('tokenInvalidInterceptor');
}).run(['auth', 'store', '$rootScope', '$location', function (auth, store, $rootScope, $location) {

    $rootScope.$on('$routeChangeStart', function (event, next, current) {
        if ($rootScope.loggedUser == null && next.templateUrl) {
            var token = store.get('token');
            if (token) {
                auth.authenticate().then(function() {
                }, function() {
                    store.remove('token');
                });
            } else if (next.loginRequired) {
                $location.path('/');
                $('#login_dialog').modal('show');
            }
        }
    });


}]);