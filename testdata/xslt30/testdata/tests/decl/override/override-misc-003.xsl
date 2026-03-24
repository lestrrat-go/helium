<xsl:package 
  name="http://www.w3.org/xslt30tests/override-001"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <xsl:variable name="data"><e/><f/><g/></xsl:variable>
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-901"
    package-version="1.0.0">
    
    <!-- Cannot override a decimal format -->
  
    <xsl:override>
  
      <xsl:decimal-format name="d" NaN="0"/>
      
    </xsl:override>  
    
  </xsl:use-package>
  
 
   
    <xsl:template name="main" visibility="public">
      <out><xsl:value-of select="format-number(17, '999', 'd')"/></out>
    </xsl:template>
    
 
  
</xsl:package>   