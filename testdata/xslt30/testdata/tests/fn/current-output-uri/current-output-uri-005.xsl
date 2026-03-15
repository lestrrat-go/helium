<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://f.com/" exclude-result-prefixes="xs f">

  <!-- current-output-uri() while evaluating a stylesheet function -->

  <xsl:function name="f:get">
    <xsl:sequence select="current-output-uri()"/>
  </xsl:function>
  
  <xsl:template name="main">
    <out><xsl:value-of select="f:get()"/></out>
  </xsl:template>
  

</xsl:stylesheet>
