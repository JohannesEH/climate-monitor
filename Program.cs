using System;
using System.Device.I2c;
using System.Threading;
using Iot.Device.DHTxx;

namespace climate_monitor
{
    public class Program
    {
        public static void Main (string[] args)
        {
            var settings = new I2cConnectionSettings(1, Dht10.DefaultI2cAddress);
            var device = I2cDevice.Create(settings);

            using(var dht = new Dht10(device))
            {
                while(true)
                {
                    Console.WriteLine($"Temperature: {dht.Temperature.Celsius.ToString("0.0")} °C, Humidity: {dht.Humidity.ToString("0.0")} %");
                    Thread.Sleep(1000);
                }
            }
        }
    }
}
