<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:f="http://local.functions.net/" version="3.0">
  <!--[ERR XTSE3155] It is a static error if an xsl:function element
    with no xsl:param children has a streamability attribute with any
    value other than unclassified.
  -->
  
<xsl:function name="f:x" streamability="absorbing">
  <xsl:sequence select="22"/>
</xsl:function>  
  
   <xsl:template name="main">
      <out>3</out>
   </xsl:template>
</xsl:stylesheet>
