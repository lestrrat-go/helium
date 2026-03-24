<xsl:package 
  name="http://www.w3.org/xslt30tests/override-v-014b"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/override-v-014"
  exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Override a global variable referenced from an inline function that has a closure -->
  
  <xsl:variable name="var" as="xs:integer" select="3" visibility="public"/>
  
  <xsl:function name="f:f" visibility="public">
    <xsl:variable name="w" select="$var * 2"/>
    <xsl:sequence select="function() { $w + $var }"/>
  </xsl:function>
  
  
</xsl:package>   