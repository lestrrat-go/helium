<xsl:package 
  name="http://www.w3.org/xslt30tests/override-103"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-101"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-101"
    package-version="1.0.0">
    
    <xsl:accept component="template" names="t-abstract" visibility="abstract"/>
  
    <xsl:override>
  
      <xsl:template name="t" as="xs:string" visibility="public">
        <xsl:param name="in" as="xs:string"/>
        <xsl:param name="count" as="xs:integer"/>
        <xsl:value-of>***</xsl:value-of>
      </xsl:template>

    </xsl:override>
    
    
  </xsl:use-package>
  
  
  
    <!-- Demonstrate that abstract components make a package non-executable, even if they aren't referenced anywhere -->
   
    <xsl:template name="main" visibility="public">
        <out/>
    </xsl:template>
    
  
  
</xsl:package>   