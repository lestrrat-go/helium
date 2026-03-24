<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
  name="http://www.w3.org/xslt30tests/accept-007"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/accept-A"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/accept-A"
     package-version="1.0.0">
     
    <xsl:accept component="variable" names="p:v2" visibility="private"/>
    
    <xsl:accept component="variable" names="*" visibility="hidden"/>
    <xsl:accept component="template" names="*" visibility="hidden"/>
    <xsl:accept component="function" names="*" visibility="hidden"/>
    <xsl:accept component="attribute-set" names="*" visibility="hidden"/>
    <xsl:accept component="mode" names="*" visibility="hidden"/>
         
  </xsl:use-package>  
  
  <xsl:template name="main" visibility="public">
    <out>
      <v2><xsl:value-of select="$p:v2"/></v2>
    </out>
  </xsl:template>  
  

</xsl:package>   