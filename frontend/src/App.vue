<template>
	<nav id="navbar" class="navbar navbar-expand-sm bg-dark navbar-dark">
		<a class="navbar-brand" href="/">Industrial.Market</a>
		<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#main_nav">
			<span class="navbar-toggler-icon"></span>
		</button>

		<div class="navbar-collapse collapse" id="main_nav">
			<ul class="nav navbar-nav">
				<li><router-link class="nav-link" to="/">Поиск товаров</router-link></li>
				<li><router-link class="nav-link" to="/orders">Заказы</router-link></li>
			</ul>
			<ul class="nav navbar-nav ml-auto">
				<li class="nav-item"><hr class="border-top"></li>
				<li> <a id="navbar-userinfo" class="nav-link" href="/profile_settings/user-profile-editor"> Username [user@example.com]</a> </li>
			</ul>
		</div>
	</nav>
	<router-view/>
</template>

<style>
#app {
	font-family: Avenir, Helvetica, Arial, sans-serif;
	-webkit-font-smoothing: antialiased;
	-moz-osx-font-smoothing: grayscale;
	text-align: center;
	color: #2c3e50;
}

#nav {
  padding: 30px;
}

#nav a {
  font-weight: bold;
  color: #2c3e50;
}

#nav a.router-link-exact-active {
  color: #42b983;
}
</style>
