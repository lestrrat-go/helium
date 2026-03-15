<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-106"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:q="http://www.w3.org/xslt30tests/use-package-106"
  exclude-result-prefixes="xs p q">
  
  <!-- tests that character maps are local to a package -->
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-base-002" package-version="1.0.0"/>
  
  
    
    <xsl:output use-character-maps="cm"/>
    
    <xsl:template name="main" visibility="public">
      <xsl:result-document><out/></xsl:result-document>
    </xsl:template>

 
  
</xsl:package>   