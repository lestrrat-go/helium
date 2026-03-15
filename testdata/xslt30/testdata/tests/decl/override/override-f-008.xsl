<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-008"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <!-- empty xsl:override is OK  -->
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-001"
    package-version="1.0.0">
  
    <xsl:override/>
    <xsl:accept component="function" names="*:f-abstract" visibility="hidden"/>
  </xsl:use-package>
  
  
   
    <xsl:template name="main" visibility="public">
      <out/>
    </xsl:template>
    
 
  
</xsl:package>   