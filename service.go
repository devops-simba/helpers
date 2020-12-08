package helpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const ErrServiceStopped = StringError("Service is stopped")

func IsServiceStoppedError(err error) bool {
	return err == nil || errors.Is(err, ErrServiceStopped) || errors.Is(err, http.ErrServerClosed)
}
func getServiceResult(err error) error {
	if err == nil {
		return nil
	} else if IsServiceStoppedError(err) {
		return nil
	} else {
		return err
	}
}

// Service represent an object that will run a service in a single function
type Service interface {
	// Name will be used in logging
	GetName() string
	// Run execute the service inside this function and in the end return the error.
	Run() error
	// Shutdown shutdown the service, `Run` must stop and return `nil`, `http.ErrServerClosed` or `ErrServiceStopped`
	// if it stopped as a result of calling this function
	Shutdown()
}

// AsyncService represent a service that have a start and a stop.
// A simpler version of this interface is `Service` when entire lifetime of your service may be represented by one function
type AsyncService interface {
	// Name will be used in logging
	GetName() string
	// Start execution of this service and return a channel that we may fetch result of execution of the service from it.
	Start() <-chan error
	// Stop execution of this service.
	Stop()
}

type ServiceExecuter interface {
	// ExecuteServiceAsync Start execution of a service in background and return a channel that you may fetch result of
	// service execution from it.
	// It will return `nil` if service stopped without any error
	ExecuteServiceAsync(service Service, stopRequested <-chan struct{}) (serviceStopped <-chan error)
	// RunService execute a service and wait for its completion, also you may request its stop at any point
	RunService(service Service, stopRequested <-chan struct{}) error
	// ExecuteAsyncService execute an `AsyncService`
	ExecuteAsyncService(service AsyncService, stopRequested <-chan struct{}) (serviceStopped <-chan error)
}

var nullExecuter = loggerServiceExecuter{Factory: NullLoggerFactory}
var globalServiceExecuter ServiceExecuter = nullExecuter

func GetGlobalServiceExecuter() ServiceExecuter { return globalServiceExecuter }
func SetGlobalServiceExecuter(executer ServiceExecuter) ServiceExecuter {
	result := globalServiceExecuter
	globalServiceExecuter = executer
	return result
}
func CreateServiceExecuter(factory LogFactory) ServiceExecuter {
	return loggerServiceExecuter{Factory: factory}
}

type loggerServiceExecuter struct {
	Factory LogFactory
}

func (this loggerServiceExecuter) ExecuteServiceAsync(service Service, stopRequested <-chan struct{}) (serviceStopped <-chan error) {
	var stopped chan error
	logger := this.Factory.CreateLogger(fmt.Sprintf("services/%s", service.GetName()), nil, nil)
	if stopRequested == nil {
		stopped = make(chan error, 1)
		go func() {
			logger.Verbose(10, "Running service in the background")
			err := getServiceResult(service.Run())
			logger.Verbosef(10, "Service stopped: %v", err)
			stopped <- err
		}()
		return stopped
	} else {
		stopped = make(chan error, 2)
		go func() {
			logger.Verbose(10, "Running service in the background")
			err := getServiceResult(service.Run())
			logger.Verbosef(10, "Service stopped: %v", err)
			stopped <- err
			stopped <- err
		}()
		go func() {
			select {
			case <-stopRequested:
				logger.Verbose(10, "Received stop signal, shutting down the service")
				service.Shutdown()
				logger.Verbose(10, "Server shutdown called, Waiting for stop signal")
				<-stopped
				logger.Verbose(10, "Stop signal received after calling Shutdown")
			case <-stopped:
				logger.Verbose(10, "Stop signal received")
			}
		}()
	}
	return stopped
}
func (this loggerServiceExecuter) RunService(service Service, stopRequested <-chan struct{}) error {
	return <-this.ExecuteServiceAsync(service, stopRequested)
}
func (this loggerServiceExecuter) ExecuteAsyncService(service AsyncService, stopRequested <-chan struct{}) (serviceStopped <-chan error) {
	logger := this.Factory.CreateLogger(fmt.Sprintf("asyncServices/%s", service.GetName()), nil, nil)
	logger.Verbose(10, "Starting the service")
	svcStopped := service.Start()
	if stopRequested == nil {
		stopped := make(chan error, 1)
		go func() {
			err := <-svcStopped
			err = getServiceResult(err)
			logger.Verbosef(10, "Service stopped: %v", err)
			stopped <- err
		}()
		return stopped
	} else {
		stopped := make(chan error, 2)
		go func() {
			err := getServiceResult(<-svcStopped)
			logger.Verbosef(10, "Service stopped: %v", err)
			stopped <- err
			stopped <- err
		}()
		go func() {
			select {
			case <-stopRequested:
				logger.Verbose(10, "Stop requested, stopping the service")
				service.Stop()
				logger.Verbose(10, "Service stop called, waiting for stop signal")
				<-stopped
				logger.Verbose(10, "Stop signal received after calling Stop")
			case <-stopped:
				logger.Verbose(10, "Stop signal received")
			}
		}()
		return stopped
	}
}

func ExecuteServiceAsync(service Service, stopRequested <-chan struct{}) (serviceStopped <-chan error) {
	return GetGlobalServiceExecuter().ExecuteServiceAsync(service, stopRequested)
}
func RunService(service Service, stopRequested <-chan struct{}) error {
	return GetGlobalServiceExecuter().RunService(service, stopRequested)
}
func ExecuteAsyncService(service AsyncService, stopRequested <-chan struct{}) (serviceStopped <-chan error) {
	return GetGlobalServiceExecuter().ExecuteAsyncService(service, stopRequested)
}

// Helper that wrap `Service` as `AsyncService`
type serviceToAsyncService struct {
	service Service
}

// ServiceToAsyncService wrap a `Service` in an object that implement `AsyncService` interface
func ServiceToAsyncService(service Service) AsyncService {
	if wrapper, ok := service.(asyncServiceToService); ok {
		return wrapper.asyncService
	}
	return serviceToAsyncService{service: service}
}
func (this serviceToAsyncService) GetName() string { return this.service.GetName() }
func (this serviceToAsyncService) Start() <-chan error {
	stopped := make(chan error)
	go func() { stopped <- this.service.Run() }()
	return stopped
}
func (this serviceToAsyncService) Stop() {
	this.service.Shutdown()
}

// Helper that wrap `AsyncService` as `Service`
type asyncServiceToService struct {
	asyncService AsyncService
}

// AsyncServiceToService wrap an `AsyncService` in an object that implement `Service` interface
func AsyncServiceToService(asyncService AsyncService) Service {
	if wrapper, ok := asyncService.(serviceToAsyncService); ok {
		return wrapper.service
	}
	return asyncServiceToService{asyncService: asyncService}
}
func (this asyncServiceToService) GetName() string { return this.asyncService.GetName() }
func (this asyncServiceToService) Run() error {
	return <-this.asyncService.Start()
}
func (this asyncServiceToService) Shutdown() {
	this.asyncService.Stop()
}

// Helper that wrap run/shutdown function as `Service`
type serviceFuncs struct {
	Name     string
	run      func() error
	shutdown func()
}

func ServiceFuncs(name string, run func() error, shutdown func()) Service {
	return serviceFuncs{Name: name, run: run, shutdown: shutdown}
}
func (this serviceFuncs) GetName() string { return this.Name }
func (this serviceFuncs) Run() error      { return this.run() }
func (this serviceFuncs) Shutdown()       { this.shutdown() }

// Helper that wrap start/stop function as `AsyncService`
type asyncServiceFuncs struct {
	Name  string
	start func() <-chan error
	stop  func()
}

func AsyncServiceFuncs(name string, start func() <-chan error, stop func()) AsyncService {
	return asyncServiceFuncs{Name: name, start: start, stop: stop}
}
func (this asyncServiceFuncs) GetName() string     { return this.Name }
func (this asyncServiceFuncs) Start() <-chan error { return this.start() }
func (this asyncServiceFuncs) Stop()               { this.stop() }

// Helper that wrap a `http.Server` as `Server`
type httpService struct {
	Name   string
	Server *http.Server
	Secure bool
}

func HttpService(name string, server *http.Server, secure bool) Service {
	return httpService{Name: name, Server: server, Secure: secure}
}
func (this httpService) GetName() string { return this.Name }
func (this httpService) Run() error {
	if this.Secure {
		return this.Server.ListenAndServeTLS("", "")
	} else {
		return this.Server.ListenAndServe()
	}
}
func (this httpService) Shutdown() { this.Server.Shutdown(context.Background()) }

// Helper that merge multiple services into a single `Service`
type mergedService struct {
	Name     string
	Services []Service
}

func MergeServices(name string, services ...Service) Service {
	if len(services) == 0 {
		return nil
	}
	if len(services) == 1 {
		return services[0]
	}
	return mergedService{Name: name, Services: services}
}

func (this mergedService) GetName() string { return this.Name }
func (this mergedService) Run() error {
	resultChannel := make(chan error, len(this.Services))
	for i := 0; i < len(this.Services); i++ {
		go func(service Service) {
			err := service.Run()
			if err != nil {
				err = ComponentError{Component: service, Failure: err}
			}
			resultChannel <- err
		}(this.Services[i])
	}

	errBuilder := AggregateErrorBuilder{}
	for i := 0; i < len(this.Services); i++ {
		err := <-resultChannel
		errBuilder.AddError(err) // this will take care of nil errors
	}
	return errBuilder.GetError()
}
func (this mergedService) Shutdown() {
	for i := 0; i < len(this.Services); i++ {
		this.Services[i].Shutdown()
	}
}

// Helper that merge multiple async services into a single `AsyncService`
type mergedAsyncService struct {
	Name          string
	AsyncServices []AsyncService
}

func MergeAsyncServices(name string, asyncServices ...AsyncService) AsyncService {
	if len(asyncServices) == 0 {
		return nil
	}
	if len(asyncServices) == 1 {
		return asyncServices[0]
	}
	return mergedAsyncService{Name: name, AsyncServices: asyncServices}
}

func (this mergedAsyncService) GetName() string { return this.Name }
func (this mergedAsyncService) Start() <-chan error {
	result := make(chan error, 1)
	errChannel := make(chan error, len(this.AsyncServices))
	for i := 0; i < len(this.AsyncServices); i++ {
		go func(asyncService AsyncService) {
			ch := asyncService.Start()
			err := <-ch
			if err != nil {
				err = ComponentError{Component: asyncService, Failure: err}
			}
			errChannel <- err
		}(this.AsyncServices[i])
	}

	go func() {
		errBuilder := AggregateErrorBuilder{}
		for i := 0; i < len(this.AsyncServices); i++ {
			err := <-errChannel
			errBuilder.AddError(err)
		}
		result <- errBuilder.GetError()
	}()

	return result
}
func (this mergedAsyncService) Stop() {
	for i := 0; i < len(this.AsyncServices); i++ {
		this.AsyncServices[i].Stop()
	}
}
