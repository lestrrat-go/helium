<xsl:package 
  name="http://www.w3.org/xslt30tests/override-v-013b"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/override-v-013"
  exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Override a global variable referenced from an inline function -->
  
  <xsl:variable name="var" as="xs:integer" select="3" visibility="public"/>
  
  <xsl:function name="f:f" visibility="public">
    <xsl:sequence select="function() { $var }"/>
  </xsl:function>
  
  
</xsl:package>   