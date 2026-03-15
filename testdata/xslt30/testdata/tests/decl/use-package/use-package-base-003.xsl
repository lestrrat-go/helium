<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-base-003"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
 
  
    <xsl:variable name="v-public" as="xs:string" visibility="public" select="'v/public'"/>
    
    <xsl:variable name="v-private" as="xs:string" visibility="private" select="'v/private'"/>
    
    <xsl:variable name="v-final" as="xs:string" visibility="final" select="'v/final'"/>
    
 
  
</xsl:package>   