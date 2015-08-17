'use strict';

var oReaderApp = angular.module('oReaderApp');

oReaderApp.controller('RootCtrl', ['$scope', 'auth','$location', function ($scope, auth, $location) {
    $scope.hasLogin = false;
    $scope.profile = {
        email: ''
    };

    $scope.token = "";
    //$scope.$watch('', function(isAuthenticated) {
    //    if (isAuthenticated) {
    //        $scope.hasLogin = true;
    //    } else {
    //        $scope.hasLogin = false;
    //    }
    //    $scope.profile.email = auth.profile.email;
    //});

    $scope.stories = [];
}]);

oReaderApp.controller('LoginCtrl', ['auth', 'store', '$location', '$scope', function(auth, store, $location, $scope) {
    $scope.test = '123';
    $scope.user = {
        email: '',
        password: ''
    };
    $scope.hitMsg = '';

    function onLoginSuccess(token) {
        $scope.loading = false;
        store.set('token', token);
        $scope.hitMsg = '登录成功';
        $('#login_dialog').modal('hide');
        $location.path('/about');
    }

    function onLoginFailed() {
       $scope.loading = false;
       $scope.hitMsg = '邮箱或密码不正确';
    }

    $scope.submit = function() {
        $scope.hitMsg = '';
        $scope.loading = true;
        auth.signin({
            email: $scope.user.email,
            password: $scope.user.password
        }, onLoginSuccess, onLoginFailed);
    }

}]);

oReaderApp.controller('LogoutCtrl', ["auth", "$location", "store", function (auth, $location, store) {
    auth.singout();
    store.del('token');
    $location.path('/');
}]);

oReaderApp.controller('AboutCtrl', function() {

});
