<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-032a"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/override-f-032"
  xmlns:p="http://www.w3.org/xslt30tests/override-f-033b"
  exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Uses two packages, xsl:override appears under the wrong xsl:use-package declaration -->
  
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-f-032b" package-version="*">
    
  </xsl:use-package>
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-f-033b" package-version="1.0.0">
    <xsl:override>
      <xsl:function name="p:f" as="xs:integer" visibility="public">
        <xsl:param name="in" as="xs:integer*"/>
        <xsl:sequence select="p:g() + (if (empty($in)) then 23 else head($in) + p:h(tail($in)))"/>
      </xsl:function>
      
      <xsl:function name="p:g" as="xs:integer" visibility="public">
        <xsl:sequence select="if (current-date() lt xs:date('1900-01-01')) then p:g()+1 else 0"/>
      </xsl:function>

      <xsl:function name="f:g" as="xs:integer" visibility="public">
        <xsl:sequence select="4"/>
      </xsl:function>
      
    </xsl:override>
   
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <out>{f:f()()}</out>
  </xsl:template>

 
  
</xsl:package>   