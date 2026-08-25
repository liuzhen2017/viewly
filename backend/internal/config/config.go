package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port      int    `mapstructure:"port"`
		JWTSecret string `mapstructure:"jwt_secret"`
		MockPay   bool   `mapstructure:"mock_pay"`
	} `mapstructure:"server"`

	MySQL struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Database string `mapstructure:"database"`
	} `mapstructure:"mysql"`

	App struct {
		Timezone       string `mapstructure:"timezone"`
		SignupBonus    int64  `mapstructure:"signup_bonus"`
		CheckinRewards []int  `mapstructure:"checkin_rewards"`
	} `mapstructure:"app"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.App.Timezone == "" {
		c.App.Timezone = "UTC"
	}
	if len(c.App.CheckinRewards) == 0 {
		c.App.CheckinRewards = []int{50, 50, 50, 50, 50, 50, 99}
	}
	return &c, nil
}
