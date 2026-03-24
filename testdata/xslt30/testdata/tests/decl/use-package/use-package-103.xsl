<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-103"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:q="http://www.w3.org/xslt30tests/use-package-103"
  exclude-result-prefixes="xs p q">
  
  <!-- tests that namespace aliases are local to a package -->
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-base-002" package-version="1.0.0"/>
  
 
  
    <xsl:namespace-alias stylesheet-prefix="xs" result-prefix="q"/>
    
    <xsl:function name="q:alias" as="element()" visibility="public">
      <xs:test/>
    </xsl:function>
    
    <xsl:template name="main" visibility="public">
      <xsl:element name="out">
        <xsl:sequence select="q:alias(), p:alias()"/>
      </xsl:element>
    </xsl:template>
    
  
  
</xsl:package>   