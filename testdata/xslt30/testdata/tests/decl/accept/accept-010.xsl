<!-- xsl:accept - reduce the visibility to hidden. Component becomes inaccessible -->

<xsl:package
  name="http://www.w3.org/xslt30tests/accept-003"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/accept-A"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/accept-A"
     package-version="1.0.0">
     
    <!-- Higher priority wildcard wins even if not last --> 
        
    <xsl:accept component="*" names="p:*" visibility="hidden"/>
    <xsl:accept component="*" names="*:v2" visibility="private"/>
    <xsl:accept component="variable" names="*" visibility="hidden"/> 
 
         
  </xsl:use-package>  
  
  <xsl:template name="main" visibility="public">
    <out foo="{$p:v2}"/>
  </xsl:template>  
  

</xsl:package>   