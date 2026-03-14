<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:my="http://example.com/myfunctions"
    version="3.0">

<!-- Test circular reference, trying to catch the error XPDE0640 when dynamically raised with xsl:evaluate
    See bug 28368 about capturing circularity errors (effect is implementation-dependent) -->
 
    <xsl:variable name="circular">
        <xsl:evaluate xpath="'$other'" >
          <xsl:with-param name="other" select="$other"/>
        </xsl:evaluate>
    </xsl:variable>
    
    <xsl:variable name="other">
        <xsl:evaluate xpath="'$circular'" >
            <xsl:with-param name="circular" select="$circular"/>
        </xsl:evaluate>
    </xsl:variable>
  
    <xsl:template name="xsl:initial-template">
        <out wrong="{my:catch-circular()}"/>
    </xsl:template>
    
    <xsl:function name="my:catch-circular">
        <xsl:try>
            <xsl:value-of select="$other" />
            <xsl:catch errors="err:XTDE0640">
                B) You cannot reliably catch this error!
            </xsl:catch>
        </xsl:try>
    </xsl:function>
   
</xsl:stylesheet>

