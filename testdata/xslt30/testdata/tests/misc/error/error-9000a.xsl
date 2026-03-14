<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                 <!--When a transformation is terminated by use of xsl:message
                     terminate="yes", the effect is the same as when a non-recoverable dynamic error occurs
                     during the transformation. The default error code is
                     XTMM9000; this may be overridden using the
                     error-code attribute of the xsl:message
                     instruction.-->
   <xsl:template name="main">
      <out>
         <xsl:message terminate="1">Kaput</xsl:message>
      </out>
   </xsl:template>
</xsl:stylesheet>
