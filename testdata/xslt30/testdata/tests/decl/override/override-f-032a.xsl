<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-032a"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/override-f-032"
  exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Override a global variable referenced from an inline function -->
  
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-f-032b" package-version="*">
    <xsl:override>
      <xsl:function name="f:g" as="xs:integer" visibility="public">
        <xsl:sequence select="4"/>
      </xsl:function>
    </xsl:override>
  </xsl:use-package>
  
    <xsl:template name="xsl:initial-template" visibility="public">
    <out>{f:f()()}</out>
  </xsl:template>

 
  
</xsl:package>   