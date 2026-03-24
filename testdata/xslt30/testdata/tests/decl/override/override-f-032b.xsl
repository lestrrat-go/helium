<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-032b"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/override-f-032"
  exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Override a named function referenced from an inline function -->
  
  <xsl:function name="f:g" as="xs:integer" visibility="public">
    <xsl:sequence select="3"/>
  </xsl:function>
  
  <xsl:function name="f:f" visibility="public">
    <xsl:sequence select="function() { f:g() }"/>
  </xsl:function>
  
  
</xsl:package>   