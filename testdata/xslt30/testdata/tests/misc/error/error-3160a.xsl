<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                <!--It is a non-recoverable dynamic error if the target expression
                     of an xsl:evaluate instruction is
                     not a valid XPath 3.0 expression (that is, if a static error occurs when
                     analyzing the string according to the rules of the XPath 3.0
                     specification).-->
   <xsl:template name="main">
      <out>
         <xsl:variable name="v" select="current-date()"/>
         <xsl:evaluate xpath="concat($v, '/')"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
