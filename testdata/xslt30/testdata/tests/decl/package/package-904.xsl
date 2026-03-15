<!-- misplaced xsl:use-package -->
<xsl:package
  name="http://www.w3.org/xslt30tests/package-904" 
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-001"
  exclude-result-prefixes="xs p">
  
  
  
    <xsl:template name="main">
      <not-ok/>
       <xsl:use-package name="http://www.w3.org/xslt30tests/package-903"/>  
    </xsl:template>
    
 
  
 
  
</xsl:package>   