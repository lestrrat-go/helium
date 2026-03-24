<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- Test circular reference, trying to catch the error XPDE0640 when dynamically raised with xsl:evaluate
    See bug 28368 about capturing circularity errors (will likely become disallowed) -->
 
    <xsl:param name="circular" static="yes" select="$other" />
    
    <xsl:param name="other" static="yes" select="$circular" />
  
    <xsl:template name="xsl:initial-template" use-when="$other">
        <out />
    </xsl:template>
   
</xsl:stylesheet>

