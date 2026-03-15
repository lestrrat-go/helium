<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-301"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  
  
  <xsl:attribute-set name="as-public" visibility="public" use-attribute-sets="as-private">
    <xsl:attribute name="pub1" select="'pub1'"/>
    <xsl:attribute name="pub2" select="'pub2'"/>
  </xsl:attribute-set>
    
  <xsl:attribute-set name="as-private" visibility="private" use-attribute-sets="as-base">
    <xsl:attribute name="priv1" select="'priv1'"/>
  </xsl:attribute-set>
    
  <xsl:attribute-set name="as-final" visibility="final">
    <xsl:attribute name="final1" select="'final1'"/>
  </xsl:attribute-set>
    
  <xsl:attribute-set name="as-abstract" visibility="abstract" />
    
  <xsl:attribute-set name="as-base" visibility="public">
    <xsl:attribute name="base1" select="'base1'"/>
  </xsl:attribute-set>
    
  
  
</xsl:package>   