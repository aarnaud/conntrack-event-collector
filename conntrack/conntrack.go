package conntrack

/*
#include <libnetfilter_conntrack/libnetfilter_conntrack.h>
#include <errno.h>
#cgo LDFLAGS: -lnetfilter_conntrack -lnfnetlink
typedef int (*cb)(enum nf_conntrack_msg_type type,
                                            struct nf_conntrack *ct,
                                            void *data);
int event_cb_cgo(int type, struct nf_conntrack *ct, void *data);
*/
import "C"
import (
	"os"
	"log"
	"unsafe"
	"os/signal"
	"syscall"
)

const (
	CT_BUFF_SIZE = 8388608
)

func Watch(flowChan chan FlowRecord){
	SetChanFlowRecord(flowChan)

	//Connect to Netlink
	ct_handle, err := C.nfct_open(C.NFNL_SUBSYS_CTNETLINK,
		C.NF_NETLINK_CONNTRACK_NEW|C.NF_NETLINK_CONNTRACK_UPDATE|C.NF_NETLINK_CONNTRACK_DESTROY)
	if ct_handle == nil {
		panic(err)
	}
	defer C.nfct_close(ct_handle)

	//Stop netlink on signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Print("Exiting...")
		C.nfct_close(ct_handle)
		os.Exit(0)
	}()

	//Increase bufffer
	bsize := C.nfnl_rcvbufsiz(C.nfct_nfnlh(ct_handle), CT_BUFF_SIZE);
	log.Print("Netlink buffer set to: ", bsize)

	//Link netlink and processing function
	C.nfct_callback_register(ct_handle, C.NFCT_T_NEW, (C.cb)(unsafe.Pointer(C.event_cb_cgo)), nil);
	log.Print("Netlink callback installed")

	//Start even processing!

	status, err := C.nfct_catch(ct_handle)
	if status == -1 {
		log.Print(err)
	}
}
