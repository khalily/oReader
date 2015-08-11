

var rssApp = angular.module('rssApp', []);

rssApp.controller('rssController', ['$scope', '$http', function ($scope, $http) {
    $scope.login_failed = false;
    $scope.token = "";

    $scope.login = function login(user) {
        if ($scope.login_form.$valid) {
            var req = {
                method: 'GET',
                url: '/api/get_token',
                headers: {
                    Accept: 'application/json',
                    Authorization: 'Basic ' + btoa(user.email + ':' + user.password)
                }
            };
            $http(req).
            then(function(response) {
                    console.log(response.data);
                    $scope.token = response.data.token;
                }, function(error) {
                    console.log(error.data);
                    $scope.login_failed = true;
                }
            )
        }
    };

    $scope.stories = [];
}]);

