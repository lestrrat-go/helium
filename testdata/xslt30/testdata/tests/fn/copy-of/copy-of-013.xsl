<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://f.org/"
  exclude-result-prefixes="f">
  
  <!-- FileName: copy2198 -->
  <!-- Purpose: Use copy-of() atomic values and function items -->
  
  
  <xsl:template match="/">
    <out>
      <xsl:sequence select="f:copy((1 to 5, name#1))"/>
    </out>
  </xsl:template>
  
  <xsl:function name="f:copy">
    <xsl:param name="p"/>
    <xsl:variable name="n" select="copy-of($p)"/>
    <xsl:sequence select="$n[1] = 1 and $n[5] = 5 and $n[6] instance of function(*) and local-name-from-QName(function-name($n[6])) = 'name' "/>
  </xsl:function>
  
</xsl:stylesheet>