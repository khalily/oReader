'use strict';

var oReaderApp = angular.module('oReaderApp');

oReaderApp.service('auth', ['$http', function ($http) {

    this.signin = function (options, successCallback, errorCallback) {
        var req = {
            method: 'GET',
            url: '/api/get_token',
            headers: {
                Accept: 'application/json',
                Authorization: 'Basic ' + btoa(options.email + ':' + options.password)
            }
        };
        $http(req).
        then(function(response) {
                var token = response.data.token;
                successCallback(token);
            }, function(error) {
                console.log(error.message);
                errorCallback();
            }
        )
    };
    
    this.signout = function () {
    }
}]);