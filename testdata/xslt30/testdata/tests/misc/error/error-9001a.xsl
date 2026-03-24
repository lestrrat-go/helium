<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 xmlns:xs="http://www.w3.org/2001/XMLSchema">

              <!--When a transformation is terminated by use of xsl:assert, 
                  the effect is the same as when a non-recoverable dynamic error occurs
                  during the transformation. The default error code is
                  XTMM9001; this may be overridden using the
                  error-code attribute of the xsl:assert instruction.-->
                  
   <xsl:template name="main">
      <out>
         <xsl:assert test="current-date() lt xs:date('1900-01-01')"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
