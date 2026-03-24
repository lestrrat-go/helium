<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    version="3.0">

<!-- Test circular reference, trying to catch the error XPDE0640
    See bug 28368 about capturing circularity errors (will likely become disallowed) -->
    
    <!-- What the spec actually says is: 
        
        Although a circularity is detected as a dynamic error, there is no unique instruction whose 
        evaluation triggers the error condition, and the result of any attempt to catch the error using 
        an xsl:try instruction is therefore implementation-dependent.
        
    -->    
 
  <xsl:variable name="circular">
      <xsl:try>
          <xsl:sequence select="$other" />
          <xsl:catch errors="err:XTDE0640">
              A) You cannot catch this error!
          </xsl:catch>
      </xsl:try>
  </xsl:variable>
  
  <xsl:variable name="other">
      <xsl:try>
          <xsl:sequence select="$circular" />
          <xsl:catch errors="err:XTDE0640">
              B) You cannot catch this error!
          </xsl:catch>
      </xsl:try>
  </xsl:variable>

  <xsl:template name="xsl:initial-template">
      <out wrong="{$circular}"/>
  </xsl:template>
   
</xsl:stylesheet>

