#!/usr/bin/env python



import      commands
#import      csv
#import      decimal
import      math
import      optparse
import      os
import      re
import      sys
import      time
import      user

oOptions = None


################################################################################
#                           Object Classes and Functions
################################################################################

#===============================================================================
#                           execute an OS Command Class
#===============================================================================

class       execCmd:

    def __init__( self, fExec=True, fAccum=False ):
        self.fAccum = fAccum
        self.fExec = fExec
        self.fNoOutput = False
        self.iRC = 0
        self.szCmdList = []


    def __getitem__( self, i ):
        szLine = self.szCmdList[i]
        if szLine:
            return szLine
        else:
            raise IndexError


#-------------------------------------------------------------------------------
#                   Execute a set of Bash Commands.
#-------------------------------------------------------------------------------
    def doBashSys( self, oCmds, fIgnoreRC=False ):
        "Execute a set of Bash Commands."
        if oOptions.fDebug:
            print "execCmd::doBashSys(%s)" % ( oCmds )

        # Make sure that we have a sequence type for the commands.
        import  types
        if isinstance( oCmds, types.ListType ) or isinstance( oCmds, types.TupleType):
            pass
        else:
            oCmds = [ oCmds ]

        # Build a stub bash script that will run in the chrooted environment.
        oStub,oFilePath = tempfile.mkstemp( ".sh", "bashStub", '.', text=True )
        if oOptions.fDebug:
            print "\toFilePath='%s'" % ( oFilePath )
            os.write( oStub, "#!/bin/bash -xv\n\n" )
        else:
            os.write( oStub, "#!/bin/bash\n\n" )
        for szCmd in oCmds:
            os.write( oStub, szCmd + "\n" )
        os.write( oStub, "exit $?\n" )
        os.close( oStub )

        # Now execute the Bash Stub with cleanup.
        oFileBase = os.path.basename( oFilePath )
        szCmd = "chmod +x " + oFilePath
        self.doCmd( szCmd, fIgnoreRC )
        try:
            szCmd = oFilePath
            self.doSys( szCmd, fIgnoreRC )
        finally:
            os.unlink( oFilePath )


#-------------------------------------------------------------------------------
#                           Execute a System Command.
#-------------------------------------------------------------------------------
    def doCmd( self, szCmd, fIgnoreRC=False ):
        "Execute a System Command."

        # Do initialization.
        if oOptions.fDebug:
            print "execCmd::doCmd(%s)" % (szCmd)
        if 0 == len( szCmd ):
            if oOptions.fDebug:
                print "\tcmdlen==0 so rc=0"
            raise ValueError
        szCmd = os.path.expandvars( szCmd )
        if self.fNoOutput:
            szCmd += " 2>/dev/null >/dev/null"
        if self.fAccum:
            self.szCmdList.append( szCmd )
        self.szCmd = szCmd

        #  Execute the command.
        if oOptions.fDebug:
            print "\tcommand(Debug Mode) = %s" % ( szCmd )
        if szCmd and self.fExec:
            tupleResult = commands.getstatusoutput( szCmd )
            if oOptions.fDebug:
                print "\tResult = %s, %s..." % ( tupleResult[0], tupleResult[1] )
            self.iRC = tupleResult[0]
            self.szOutput = tupleResult[1]
            if fIgnoreRC:
                return
            if 0 == tupleResult[0]:
                return
            else:
                if oOptions.fDebug:
                    print "OSError cmd:    %s" % ( szCmd )
                    print "OSError rc:     %d" % ( self.iRC )
                    print "OSError output: %s" % ( self.szOutput )
                raise OSError, szCmd
        if szCmd and not self.fExec:
            if oOptions.fDebug:
                print "\tNo-Execute enforced! Cmd not executed, but good return..."
            return

        # Return to caller.
        self.iRC = -1
        self.szOutput = None
        raise ValueError


#-------------------------------------------------------------------------------
#                       Execute a list of System Commands.
#-------------------------------------------------------------------------------
    def doCmds( self, oCmds, fIgnoreRC=False ):
        "Execute a list of System Commands."

        # Make sure that we have a sequence type for the commands.
        import  types
        if isinstance( oCmds, types.ListType ) or isinstance( oCmds, types.TupleType):
            pass
        else:
            oCmds = [ oCmds ]

        # Execute each command.
        for szCmd in oCmds:
            self.doCmd( szCmd + "\n", fIngnoreRC )


#-------------------------------------------------------------------------------
#           Execute a System Command with output directly to terminal.
#-------------------------------------------------------------------------------
    def doSys( self, szCmd, fIgnoreRC=False ):
        "Execute a System Command with output directly to terminal."

        # Do initialization.
        if oOptions.fDebug:
            print "execCmd::doSys(%s)" % (szCmd)
        if 0 == len( szCmd ):
            if oOptions.fDebug:
                print "\tcmdlen==0 so rc=0"
            raise ValueError
        szCmd = os.path.expandvars( szCmd )
        if self.fNoOutput:
            szCmd += " 2>/dev/null >/dev/null"
        if self.fAccum:
            self.szCmdList.append( szCmd )
        self.szCmd = szCmd

        #  Execute the command.
        if oOptions.fDebug:
            print "\tcommand(Debug Mode) = %s" % ( szCmd )
        if szCmd and self.fExec:
            self.iRC = os.system( szCmd )
            self.szOutput = None
            if oOptions.fDebug:
                print "\tResult = %s" % ( self.iRC )
            if fIgnoreRC:
                return
            if 0 == self.iRC:
                return
            else:
                raise OSError, szCmd
        if szCmd and not self.fExec:
            if oOptions.fDebug:
                print "\tNo-Execute enforced! Cmd not executed, but good return..."
            return

        # Return to caller.
        self.iRC = -1
        raise ValueError


    def getOutput( self ):
        return self.szOutput


    def getRC( self ):
        return self.iRC


    def len( self ):
        return len( self.szCmdList )


    def save( self ):
        return 0


    def setExec( self, fFlag=True ):
        self.fExec = fFlag


    def setNoOutput( self, fFlag=False ):
        self.fNoOutput = fFlag




#===============================================================================
#                               Miscellaneous
#===============================================================================

#---------------------------------------------------------------------
#       getAbsolutePath -- Convert a Path to an absolute path
#---------------------------------------------------------------------

def getAbsolutePath( szPath ):
    "Convert Path to an absolute path."
    if oOptions.fDebug:
        print "getAbsolutePath(%s)" % ( szPath )

    # Convert the path.
    szWork = os.path.normpath( szPath )
    szWork = os.path.expanduser( szWork )
    szWork = os.path.expandvars( szWork )
    szWork = os.path.abspath( szWork )

    # Return to caller.
    if oOptions.fDebug:
        print "\tabsolute_path=", szWork
    return szWork





################################################################################
#                           Main Program Processing
################################################################################

def         mainCLI( listArgV=None ):
    "Command-line interface."
    global      oDB
    global      oOptions
    
    # Do initialization.
    iRc = 20

    # Parse the command line.       
    szUsage = "usage: %prog [options] sourceDirectoryPath [destinationDirectoryPath]"
    oCmdPrs = optparse.OptionParser( usage=szUsage )
    oCmdPrs.add_option( "-d", "--debug", action="store_true",
                        dest="fDebug", default=False,
                        help="Set debug mode"
    )
    oCmdPrs.add_option( "-v", "--verbose",
                        action="count",
                        dest="iVerbose",
                        default=0,
                        help="Set verbose mode"
    )
    (oOptions, oArgs) = oCmdPrs.parse_args( listArgV )
    if oOptions.fDebug:
        print "In DEBUG Mode..."
        print 'Args:',oArgs

    if len(oArgs) < 1:
        szSrc = os.getcwd( )
    else:
        szSrc = oArgs[0]
    if len(oArgs) > 1:
        print "ERROR - too many command arguments!"
        oCmdPrs.print_help( )
        return 4
    if oOptions.fDebug:
        print 'szSrc:',szSrc

    # Perform the specified actions.
    iRc = 0
    try:
        # Read in the tag file.
        with open('tag.txt', 'r') as tag:
            ver = tag.read().strip().split('.')

        # Update the version.
        #print('.'.join(map(str, ver)))
        ver[2] = int(ver[2]) + 1
        newVer = '.'.join(map(str, ver))
        print newVer

        # Write out the new file
        tagOut = open("tag.txt", "w")
        tagOut.write(newVer)
        tagOut.close()

        # Now tag the git repo (git tag -a version_string -m "New Release"
        oExec = execCmd()
        cmd = "git tag -a {0} -m \"New Release\"".format(newVer)
        if not oOptions.fDebug:
            oExec.doSys(cmd)
        else:
            print "Debug:",cmd
        tupleResult = commands.getstatusoutput("git remote")
        if int(tupleResult[0]) == 0:
            remotes = tupleResult[1]
            for remote in remotes.splitlines():
                cmd = "git push  {0} --tag".format(remote.strip())
                if not oOptions.fDebug:
                    oExec.doSys(cmd)
                else:
                    print "Debug:",cmd
    finally:
        pass
    return iRc




################################################################################
#                           Command-line interface
################################################################################

if '__main__' == __name__:
    startTime = time.time( )
    iRc = mainCLI( sys.argv[1:] )
    if oOptions.iVerbose or oOptions.fDebug:
        if 0 == iRc:
            print "...Successful completion."
        else:
            print "...Completion Failure of %d" % ( iRc )
    endTime = time.time( )
    if oOptions.iVerbose or oOptions.fDebug:
        print "Start Time: %s" % (time.ctime( startTime ) )
        print "End   Time: %s" % (time.ctime( endTime ) )
    diffTime = endTime - startTime      # float Time in seconds
    iSecs = int(diffTime % 60.0)
    iMins = int((diffTime / 60.0) % 60.0)
    iHrs = int(diffTime / 3600.0)
    if oOptions.iVerbose or oOptions.fDebug:
        print "run   Time: %d:%02d:%02d" % ( iHrs, iMins, iSecs )
    sys.exit( iRc or 0 )


