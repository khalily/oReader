

var rssApp = angular.module('rssApp', []);

rssApp.controller('rssController', ['$scope', '$http', function ($scope, $http) {
    $scope.login = function login(user) {
        if (user.email) {
            console.log("test login")
        }
        $http.post('/auth/login', {email: user.email}).
            then(function(response) {
                console.log(response.data)
            }, function(error) {
                console.log(error.data)
        })
    };
    $scope.stories = [];
}]);

